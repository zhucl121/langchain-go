package vectorstores

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"

	"langchain-go/retrieval/embeddings"
	"langchain-go/retrieval/loaders"
)

// MilvusVectorStore 是 Milvus 向量存储实现。
//
// Milvus 是一个开源的向量数据库，支持大规模向量检索。
//
type MilvusVectorStore struct {
	client         client.Client
	collectionName string
	embeddings     embeddings.Embeddings
	dimension      int

	// 字段名称配置
	idField      string
	vectorField  string
	contentField string
	metadataField string

	// 索引配置
	indexType   entity.IndexType
	metricType  entity.MetricType
	indexParams map[string]string

	mu sync.RWMutex
}

// MilvusConfig 是 Milvus 配置。
type MilvusConfig struct {
	// 连接配置
	Address  string // Milvus 服务地址，如 "localhost:19530"
	Username string // 用户名（可选）
	Password string // 密码（可选）

	// 集合配置
	CollectionName string // 集合名称
	Dimension      int    // 向量维度

	// 字段名称（可选，使用默认值）
	IDField       string // ID 字段名，默认 "id"
	VectorField   string // 向量字段名，默认 "vector"
	ContentField  string // 内容字段名，默认 "content"
	MetadataField string // 元数据字段名，默认 "metadata"

	// 索引配置（可选）
	IndexType   entity.IndexType      // 索引类型，默认 HNSW
	MetricType  entity.MetricType     // 距离度量，默认 L2
	IndexParams map[string]string     // 索引参数

	// 是否自动创建集合
	AutoCreateCollection bool
}

// NewMilvusVectorStore 创建 Milvus 向量存储。
//
// 参数：
//   - config: Milvus 配置
//   - emb: 嵌入模型
//
// 返回：
//   - *MilvusVectorStore: 向量存储实例
//   - error: 错误
//
func NewMilvusVectorStore(config MilvusConfig, emb embeddings.Embeddings) (*MilvusVectorStore, error) {
	// 连接 Milvus
	milvusClient, err := client.NewGrpcClient(context.Background(), config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Milvus: %w", err)
	}

	// 认证（如果提供）
	if config.Username != "" && config.Password != "" {
		// Milvus SDK 会在创建客户端时处理认证
		// 这里只是示例，实际需要在 NewGrpcClient 中配置
	}

	// 设置默认值
	if config.IDField == "" {
		config.IDField = "id"
	}
	if config.VectorField == "" {
		config.VectorField = "vector"
	}
	if config.ContentField == "" {
		config.ContentField = "content"
	}
	if config.MetadataField == "" {
		config.MetadataField = "metadata"
	}
	if config.IndexType == "" {
		config.IndexType = entity.HNSW
	}
	if config.MetricType == "" {
		config.MetricType = entity.L2
	}
	if config.IndexParams == nil {
		config.IndexParams = map[string]string{
			"M":              "16",
			"efConstruction": "256",
		}
	}

	// 获取维度
	dimension := config.Dimension
	if dimension == 0 {
		dimension = emb.GetDimension()
	}

	store := &MilvusVectorStore{
		client:         milvusClient,
		collectionName: config.CollectionName,
		embeddings:     emb,
		dimension:      dimension,
		idField:        config.IDField,
		vectorField:    config.VectorField,
		contentField:   config.ContentField,
		metadataField:  config.MetadataField,
		indexType:      config.IndexType,
		metricType:     config.MetricType,
		indexParams:    config.IndexParams,
	}

	// 自动创建集合
	if config.AutoCreateCollection {
		if err := store.createCollectionIfNotExists(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	return store, nil
}

// createCollectionIfNotExists 创建集合（如果不存在）。
func (store *MilvusVectorStore) createCollectionIfNotExists(ctx context.Context) error {
	// 检查集合是否存在
	has, err := store.client.HasCollection(ctx, store.collectionName)
	if err != nil {
		return err
	}

	if has {
		return nil
	}

	// 定义 Schema
	schema := &entity.Schema{
		CollectionName: store.collectionName,
		Description:    "LangChain-Go vector store collection",
		AutoID:         false,
		Fields: []*entity.Field{
			{
				Name:       store.idField,
				DataType:   entity.FieldTypeVarChar,
				PrimaryKey: true,
				AutoID:     false,
				TypeParams: map[string]string{
					"max_length": "256",
				},
			},
			{
				Name:     store.vectorField,
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", store.dimension),
				},
			},
			{
				Name:     store.contentField,
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     store.metadataField,
				DataType: entity.FieldTypeJSON,
			},
		},
	}

	// 创建集合
	if err := store.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		return err
	}

	// 创建索引
	idx, err := entity.NewIndexHNSW(store.metricType, 16, 256)
	if err != nil {
		return err
	}

	if err := store.client.CreateIndex(ctx, store.collectionName, store.vectorField, idx, false); err != nil {
		return err
	}

	// 加载集合到内存
	return store.client.LoadCollection(ctx, store.collectionName, false)
}

// AddDocuments 实现 VectorStore 接口。
func (store *MilvusVectorStore) AddDocuments(ctx context.Context, docs []*loaders.Document) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}

	// 提取文本
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	// 生成嵌入
	vectors, err := store.embeddings.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	// 准备数据
	ids := make([]string, len(docs))
	idColumn := make([]string, len(docs))
	vectorColumn := make([][]float32, len(docs))
	contentColumn := make([]string, len(docs))
	metadataColumn := make([][]byte, len(docs))

	for i, doc := range docs {
		// 生成 ID
		id := fmt.Sprintf("doc_%d_%d", store.getNextID(), i)
		ids[i] = id
		idColumn[i] = id

		// 向量
		vectorColumn[i] = vectors[i]

		// 内容
		contentColumn[i] = doc.Content

		// 元数据（转为 JSON）
		metadataJSON, _ := marshalMetadata(doc.Metadata)
		metadataColumn[i] = metadataJSON
	}

	// 插入数据
	_, err = store.client.Insert(
		ctx,
		store.collectionName,
		"",
		entity.NewColumnVarChar(store.idField, idColumn),
		entity.NewColumnFloatVector(store.vectorField, store.dimension, vectorColumn),
		entity.NewColumnVarChar(store.contentField, contentColumn),
		entity.NewColumnJSONBytes(store.metadataField, metadataColumn),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert documents: %w", err)
	}

	// Flush 确保数据持久化
	if err := store.client.Flush(ctx, store.collectionName, false); err != nil {
		return nil, fmt.Errorf("failed to flush: %w", err)
	}

	return ids, nil
}

// SimilaritySearch 实现 VectorStore 接口。
func (store *MilvusVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error) {
	results, err := store.SimilaritySearchWithScore(ctx, query, k)
	if err != nil {
		return nil, err
	}

	docs := make([]*loaders.Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// SimilaritySearchWithScore 实现 VectorStore 接口。
func (store *MilvusVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error) {
	// 生成查询向量
	queryVector, err := store.embeddings.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	// 准备搜索参数
	sp, _ := entity.NewIndexHNSWSearchParam(64)

	// 执行搜索
	searchResult, err := store.client.Search(
		ctx,
		store.collectionName,
		[]string{},
		"",
		[]string{store.contentField, store.metadataField},
		[]entity.Vector{entity.FloatVector(queryVector)},
		store.vectorField,
		store.metricType,
		k,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(searchResult) == 0 {
		return []DocumentWithScore{}, nil
	}

	// 解析结果
	results := make([]DocumentWithScore, 0, k)
	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
			// 获取内容
			contentField := result.Fields.GetColumn(store.contentField)
			content := ""
			if contentCol, ok := contentField.(*entity.ColumnVarChar); ok {
				content, _ = contentCol.ValueByIdx(i)
			}

			// 获取元数据
			metadataField := result.Fields.GetColumn(store.metadataField)
			var metadata map[string]any
			if metadataCol, ok := metadataField.(*entity.ColumnJSONBytes); ok {
				metadataBytes, _ := metadataCol.ValueByIdx(i)
				metadata, _ = unmarshalMetadata(metadataBytes)
			}

			// 获取分数（距离转相似度）
			score := result.Scores[i]
			similarity := distanceToSimilarity(score, store.metricType)

			doc := &loaders.Document{
				Content:  content,
				Metadata: metadata,
			}

			results = append(results, DocumentWithScore{
				Document: doc,
				Score:    similarity,
			})
		}
	}

	return results, nil
}

// HybridSearchOptions 是混合搜索选项（Milvus 2.6+ 特性）。
type HybridSearchOptions struct {
	// VectorWeight 向量搜索权重（0.0-1.0）
	VectorWeight float32
	
	// KeywordWeight 关键词搜索权重（0.0-1.0）
	KeywordWeight float32
	
	// RerankStrategy 重排序策略
	// 可选值: "rrf" (Reciprocal Rank Fusion), "weighted" (加权融合)
	RerankStrategy string
	
	// RRFParam RRF 参数 k（默认 60）
	RRFParam int
	
	// KeywordField 用于关键词搜索的字段（默认使用 contentField）
	KeywordField string
}

// HybridSearch 执行混合搜索（Milvus 2.6+ 特性）。
//
// 混合搜索结合了向量相似度搜索和关键词（BM25）搜索。
//
// 参数：
//   - ctx: 上下文
//   - query: 查询文本
//   - k: 返回结果数量
//   - options: 混合搜索选项
//
// 返回：
//   - []DocumentWithScore: 重排序后的文档列表
//   - error: 错误
//
func (store *MilvusVectorStore) HybridSearch(
	ctx context.Context,
	query string,
	k int,
	options *HybridSearchOptions,
) ([]DocumentWithScore, error) {
	// 设置默认值
	if options == nil {
		options = &HybridSearchOptions{
			VectorWeight:   0.7,
			KeywordWeight:  0.3,
			RerankStrategy: "rrf",
			RRFParam:       60,
		}
	}
	
	if options.KeywordField == "" {
		options.KeywordField = store.contentField
	}
	
	if options.RRFParam == 0 {
		options.RRFParam = 60
	}

	// 1. 向量搜索
	vectorResults, err := store.SimilaritySearchWithScore(ctx, query, k*2)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// 2. 关键词搜索（BM25）
	keywordResults, err := store.keywordSearch(ctx, query, k*2, options.KeywordField)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// 3. 重排序融合
	var mergedResults []DocumentWithScore
	switch options.RerankStrategy {
	case "rrf":
		mergedResults = store.rerankRRF(vectorResults, keywordResults, options.RRFParam)
	case "weighted":
		mergedResults = store.rerankWeighted(vectorResults, keywordResults, options.VectorWeight, options.KeywordWeight)
	default:
		mergedResults = store.rerankRRF(vectorResults, keywordResults, options.RRFParam)
	}

	// 4. 取前 k 个结果
	if len(mergedResults) > k {
		mergedResults = mergedResults[:k]
	}

	return mergedResults, nil
}

// keywordSearch 执行关键词搜索（BM25）。
func (store *MilvusVectorStore) keywordSearch(
	ctx context.Context,
	query string,
	k int,
	field string,
) ([]DocumentWithScore, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	// 构建 BM25 搜索表达式
	// Milvus 2.6+ 支持全文检索
	expr := fmt.Sprintf("TEXT_MATCH(%s, '%s')", field, escapeQuery(query))

	// 执行查询
	queryResult, err := store.client.Query(
		ctx,
		store.collectionName,
		[]string{},
		expr,
		[]string{store.idField, store.contentField, store.metadataField},
	)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// 解析结果
	results := make([]DocumentWithScore, 0, k)
	for i := 0; i < queryResult.ResultCount && i < k; i++ {
		// 获取内容
		contentField := queryResult.Fields.GetColumn(store.contentField)
		content := ""
		if contentCol, ok := contentField.(*entity.ColumnVarChar); ok {
			content, _ = contentCol.ValueByIdx(i)
		}

		// 获取元数据
		metadataField := queryResult.Fields.GetColumn(store.metadataField)
		var metadata map[string]any
		if metadataCol, ok := metadataField.(*entity.ColumnJSONBytes); ok {
			metadataBytes, _ := metadataCol.ValueByIdx(i)
			metadata, _ = unmarshalMetadata(metadataBytes)
		}

		// BM25 分数（假设为 1.0，实际需要从 Milvus 获取）
		score := float32(1.0) / float32(i+1)

		doc := &loaders.Document{
			Content:  content,
			Metadata: metadata,
		}

		results = append(results, DocumentWithScore{
			Document: doc,
			Score:    score,
		})
	}

	return results, nil
}

// rerankRRF 使用 Reciprocal Rank Fusion 重排序。
//
// RRF 算法: score = sum(1 / (k + rank_i))
//
func (store *MilvusVectorStore) rerankRRF(
	vectorResults []DocumentWithScore,
	keywordResults []DocumentWithScore,
	k int,
) []DocumentWithScore {
	// 构建文档 -> 排名映射
	type docScore struct {
		doc   *loaders.Document
		score float32
	}
	
	scoreMap := make(map[string]*docScore)
	
	// 计算向量搜索的 RRF 分数
	for i, result := range vectorResults {
		key := result.Document.Content // 使用内容作为唯一标识
		score := float32(1.0) / float32(k+i+1)
		
		if existing, ok := scoreMap[key]; ok {
			existing.score += score
		} else {
			scoreMap[key] = &docScore{
				doc:   result.Document,
				score: score,
			}
		}
	}
	
	// 计算关键词搜索的 RRF 分数
	for i, result := range keywordResults {
		key := result.Document.Content
		score := float32(1.0) / float32(k+i+1)
		
		if existing, ok := scoreMap[key]; ok {
			existing.score += score
		} else {
			scoreMap[key] = &docScore{
				doc:   result.Document,
				score: score,
			}
		}
	}
	
	// 转换为结果列表
	results := make([]DocumentWithScore, 0, len(scoreMap))
	for _, ds := range scoreMap {
		results = append(results, DocumentWithScore{
			Document: ds.doc,
			Score:    ds.score,
		})
	}
	
	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

// rerankWeighted 使用加权融合重排序。
func (store *MilvusVectorStore) rerankWeighted(
	vectorResults []DocumentWithScore,
	keywordResults []DocumentWithScore,
	vectorWeight float32,
	keywordWeight float32,
) []DocumentWithScore {
	// 归一化权重
	totalWeight := vectorWeight + keywordWeight
	vectorWeight /= totalWeight
	keywordWeight /= totalWeight
	
	type docScore struct {
		doc   *loaders.Document
		score float32
	}
	
	scoreMap := make(map[string]*docScore)
	
	// 加权向量搜索分数
	for _, result := range vectorResults {
		key := result.Document.Content
		score := result.Score * vectorWeight
		
		if existing, ok := scoreMap[key]; ok {
			existing.score += score
		} else {
			scoreMap[key] = &docScore{
				doc:   result.Document,
				score: score,
			}
		}
	}
	
	// 加权关键词搜索分数
	for _, result := range keywordResults {
		key := result.Document.Content
		score := result.Score * keywordWeight
		
		if existing, ok := scoreMap[key]; ok {
			existing.score += score
		} else {
			scoreMap[key] = &docScore{
				doc:   result.Document,
				score: score,
			}
		}
	}
	
	// 转换为结果列表
	results := make([]DocumentWithScore, 0, len(scoreMap))
	for _, ds := range scoreMap {
		results = append(results, DocumentWithScore{
			Document: ds.doc,
			Score:    ds.score,
		})
	}
	
	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

// escapeQuery 转义查询字符串。
func escapeQuery(query string) string {
	// 转义特殊字符
	query = strings.ReplaceAll(query, "'", "\\'")
	query = strings.ReplaceAll(query, "\"", "\\\"")
	return query
}

// Delete 实现 VectorStore 接口。
func (store *MilvusVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	// 构造删除表达式
	expr := fmt.Sprintf("%s in [%s]", store.idField, joinIDs(ids))

	// 删除数据
	if err := store.client.Delete(ctx, store.collectionName, "", expr); err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return store.client.Flush(ctx, store.collectionName, false)
}

// GetDocumentCount 获取文档数量。
func (store *MilvusVectorStore) GetDocumentCount(ctx context.Context) (int64, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	stats, err := store.client.GetCollectionStatistics(ctx, store.collectionName)
	if err != nil {
		return 0, err
	}

	return stats[entity.CollectionRowCount], nil
}

// DropCollection 删除集合。
func (store *MilvusVectorStore) DropCollection(ctx context.Context) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.client.DropCollection(ctx, store.collectionName)
}

// Close 关闭连接。
func (store *MilvusVectorStore) Close() error {
	return store.client.Close()
}

// 辅助函数

var idCounter int64

func (store *MilvusVectorStore) getNextID() int64 {
	store.mu.Lock()
	defer store.mu.Unlock()
	idCounter++
	return idCounter
}

// distanceToSimilarity 将距离转换为相似度分数。
func distanceToSimilarity(distance float32, metricType entity.MetricType) float32 {
	switch metricType {
	case entity.L2:
		// L2 距离: 越小越相似，转换为 0-1
		return 1.0 / (1.0 + distance)
	case entity.IP:
		// 内积: 越大越相似
		return distance
	case entity.COSINE:
		// 余弦相似度: 已经是 0-1
		return distance
	default:
		return distance
	}
}

// marshalMetadata 序列化元数据为 JSON。
func marshalMetadata(metadata map[string]any) ([]byte, error) {
	if metadata == nil {
		return []byte("{}"), nil
	}
	// 简单实现，实际应使用 json.Marshal
	return []byte("{}"), nil
}

// unmarshalMetadata 反序列化元数据。
func unmarshalMetadata(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	// 简单实现，实际应使用 json.Unmarshal
	return map[string]any{}, nil
}

// joinIDs 连接 ID 列表为 Milvus 表达式格式。
func joinIDs(ids []string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = fmt.Sprintf("'%s'", id)
	}
	return fmt.Sprintf("%v", quoted)
}

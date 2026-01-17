package vectorstores

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// MilvusVectorStore 是 Milvus 向量存储实现 (使用 SDK v2.6.x)
type MilvusVectorStore struct {
	client         milvusclient.Client
	collectionName string
	embeddings     embeddings.Embeddings
	dimension      int

	// 字段名称配置
	idField       string
	vectorField   string
	contentField  string
	metadataField string

	idCounter int64
	mu        sync.RWMutex
}

// HybridSearchOptions 混合检索选项
type HybridSearchOptions struct {
	// RRF (Reciprocal Rank Fusion) 参数
	RRFRankConstant int // RRF k 参数,默认 60
	// 可以扩展其他参数
}

// HybridSearchResult 混合检索结果
type HybridSearchResult struct {
	Document      *loaders.Document
	VectorScore   float32 // 向量检索分数
	KeywordScore  float32 // 关键词检索分数 (如果支持)
	FusionScore   float32 // RRF 融合后的分数
}

// MilvusConfig 是 Milvus 配置
type MilvusConfig struct {
	Address              string // Milvus 服务地址，如 "localhost:19530"
	CollectionName       string // 集合名称
	Dimension            int    // 向量维度
	AutoCreateCollection bool   // 是否自动创建集合

	// 字段名称（可选）
	IDField       string
	VectorField   string
	ContentField  string
	MetadataField string
}

// NewMilvusVectorStore 创建新的 Milvus 向量存储 (使用 SDK v2.6.x)
func NewMilvusVectorStore(config MilvusConfig, emb embeddings.Embeddings) (*MilvusVectorStore, error) {
	ctx := context.Background()

	// 使用新 SDK v2.6.x API 连接
	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: config.Address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Milvus: %w", err)
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

	// 获取维度
	dimension := config.Dimension
	if dimension == 0 {
		dimension = emb.GetDimension()
	}

	store := &MilvusVectorStore{
		client:         *cli, // 解引用
		collectionName: config.CollectionName,
		embeddings:     emb,
		dimension:      dimension,
		idField:        config.IDField,
		vectorField:    config.VectorField,
		contentField:   config.ContentField,
		metadataField:  config.MetadataField,
	}

	// 自动创建集合
	if config.AutoCreateCollection {
		if err := store.createCollectionIfNotExists(ctx); err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// 确保集合已加载到内存
	has, err := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(config.CollectionName))
	if err == nil && has {
		// 加载集合 (关键:避免首次 Insert 时的延迟)
		_, _ = cli.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(config.CollectionName))
	}

	return store, nil
}

// createCollectionIfNotExists 创建集合（如果不存在）
func (store *MilvusVectorStore) createCollectionIfNotExists(ctx context.Context) error {
	// 检查集合是否存在
	has, err := store.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(store.collectionName))
	if err != nil {
		return err
	}

	if has {
		return nil
	}

	// 创建 schema
	schema := entity.NewSchema().
		WithName(store.collectionName).
		WithDescription("LangChain-Go vector store collection").
		WithField(entity.NewField().WithName(store.idField).WithDataType(entity.FieldTypeVarChar).WithMaxLength(256).WithIsPrimaryKey(true)).
		WithField(entity.NewField().WithName(store.vectorField).WithDataType(entity.FieldTypeFloatVector).WithDim(int64(store.dimension))).
		WithField(entity.NewField().WithName(store.contentField).WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535)).
		WithField(entity.NewField().WithName(store.metadataField).WithDataType(entity.FieldTypeJSON))

	// 创建集合
	err = store.client.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(store.collectionName, schema))
	if err != nil {
		return err
	}

	// 创建 HNSW 索引
	idx := index.NewHNSWIndex(entity.L2, 16, 256)
	indexOptions := milvusclient.NewCreateIndexOption(store.collectionName, store.vectorField, idx).
		WithIndexName(store.vectorField + "_idx")

	_, err = store.client.CreateIndex(ctx, indexOptions)
	if err != nil {
		return err
	}

	// 加载集合到内存
	_, err = store.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(store.collectionName))
	return err
}

// AddDocuments 添加文档
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
		vectorColumn[i] = vectors[i]
		contentColumn[i] = doc.Content

		// 元数据转 JSON
		metadataJSON, _ := marshalMetadata(doc.Metadata)
		metadataColumn[i] = metadataJSON
	}

	// 插入数据 (使用新 SDK v2.6.x API)
	jsonColumn := column.NewColumnJSONBytes(store.metadataField, metadataColumn)
	
	insertOption := milvusclient.NewColumnBasedInsertOption(store.collectionName).
		WithVarcharColumn(store.idField, idColumn).
		WithFloatVectorColumn(store.vectorField, store.dimension, vectorColumn).
		WithVarcharColumn(store.contentField, contentColumn).
		WithColumns(jsonColumn)

	_, err = store.client.Insert(ctx, insertOption)
	if err != nil {
		return nil, fmt.Errorf("failed to insert documents: %w", err)
	}

	// 注意:移除同步 Flush,Milvus 会自动在后台 flush
	// 如果需要立即可见性,可以显式调用: store.client.Flush(ctx, ...)

	return ids, nil
}

// SimilaritySearch 相似度搜索
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

// SimilaritySearchWithScore 带分数的相似度搜索
func (store *MilvusVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error) {
	// 生成查询向量
	queryVector, err := store.embeddings.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	store.mu.RLock()
	defer store.mu.RUnlock()

	// 执行搜索 (使用新 SDK v2.6.x API)
	searchOption := milvusclient.NewSearchOption(
		store.collectionName,
		k,
		[]entity.Vector{entity.FloatVector(queryVector)},
	).
		WithANNSField(store.vectorField).
		WithOutputFields(store.idField, store.contentField)

	searchResults, err := store.client.Search(ctx, searchOption)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 解析结果
	var results []DocumentWithScore
	if len(searchResults) > 0 {
		resultSet := searchResults[0]
		
		// 获取内容列
		contentColumn := resultSet.GetColumn(store.contentField)
		metadataColumn := resultSet.GetColumn(store.metadataField)
		
		for i := 0; i < resultSet.ResultCount; i++ {
			doc := &loaders.Document{
				Content:  "",
				Metadata: make(map[string]interface{}),
			}
			
			// 获取内容
			if contentColumn != nil && i < contentColumn.Len() {
				if varcharCol, ok := contentColumn.(*column.ColumnVarChar); ok {
					content, err := varcharCol.Get(i)
					if err == nil {
						if contentStr, ok := content.(string); ok {
							doc.Content = contentStr
						}
					}
				}
			}
			
			// 获取元数据
			if metadataColumn != nil && i < metadataColumn.Len() {
				if jsonCol, ok := metadataColumn.(*column.ColumnJSONBytes); ok {
					metadataBytes, err := jsonCol.Get(i)
					if err == nil {
						if bytesData, ok := metadataBytes.([]byte); ok && len(bytesData) > 0 {
							json.Unmarshal(bytesData, &doc.Metadata)
						}
					}
				}
			}
			
			// 获取分数
			score := float32(0)
			if i < len(resultSet.Scores) {
				score = resultSet.Scores[i]
			}
			
			results = append(results, DocumentWithScore{
				Document: doc,
				Score:    score,
			})
		}
	}

	return results, nil
}

// Close 关闭连接
func (store *MilvusVectorStore) Close() error {
	return nil // v2.6.x Client 是值类型,无需关闭
}

// DropCollection 删除集合
func (store *MilvusVectorStore) DropCollection(ctx context.Context) error {
	return store.client.DropCollection(ctx, milvusclient.NewDropCollectionOption(store.collectionName))
}

// GetDocumentCount 获取文档数量
func (store *MilvusVectorStore) GetDocumentCount() int {
	// 新 SDK 需要使用 Query 来获取 count
	// 这里简化实现
	return 0
}

// getNextID 生成下一个 ID
func (store *MilvusVectorStore) getNextID() int64 {
	return atomic.AddInt64(&store.idCounter, 1)
}

// marshalMetadata 序列化元数据为 JSON
func marshalMetadata(metadata map[string]interface{}) ([]byte, error) {
	if metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(metadata)
}

// unmarshalMetadata 反序列化元数据
func unmarshalMetadata(data []byte) (map[string]interface{}, error) {
	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

// HybridSearch 混合检索 (向量搜索 + 可选的全文搜索,使用 RRF 融合)
// Milvus 2.4+ 支持混合检索,这里实现基于多次搜索的 RRF 融合
func (store *MilvusVectorStore) HybridSearch(ctx context.Context, query string, k int, opts *HybridSearchOptions) ([]HybridSearchResult, error) {
	if opts == nil {
		opts = &HybridSearchOptions{
			RRFRankConstant: 60,
		}
	}
	if opts.RRFRankConstant == 0 {
		opts.RRFRankConstant = 60
	}

	// 1. 执行向量搜索
	vectorResults, err := store.SimilaritySearchWithScore(ctx, query, k*2) // 取 2k 个结果用于 RRF
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// 2. 可以在这里添加其他搜索方法 (如 BM25 全文搜索)
	// 目前只使用向量搜索,但架构支持扩展
	
	// 3. 使用 RRF 融合结果
	results := store.applyRRF([][]DocumentWithScore{vectorResults}, opts.RRFRankConstant, k)
	
	return results, nil
}

// applyRRF 应用 Reciprocal Rank Fusion 算法融合多个排序列表
// RRF score = sum(1 / (k + rank_i)) for all lists
func (store *MilvusVectorStore) applyRRF(resultSets [][]DocumentWithScore, k int, topK int) []HybridSearchResult {
	// 使用 map 存储每个文档的分数
	docScores := make(map[string]*HybridSearchResult)
	
	// 遍历每个结果集
	for setIdx, results := range resultSets {
		for rank, docWithScore := range results {
			// 使用文档内容作为唯一标识 (简化实现)
			docKey := docWithScore.Document.Content
			
			if _, exists := docScores[docKey]; !exists {
				docScores[docKey] = &HybridSearchResult{
					Document:     docWithScore.Document,
					VectorScore:  0,
					KeywordScore: 0,
					FusionScore:  0,
				}
			}
			
			// 计算 RRF 分数: 1 / (k + rank)
			rrfScore := 1.0 / float32(k+rank+1)
			docScores[docKey].FusionScore += rrfScore
			
			// 记录各个搜索的原始分数
			if setIdx == 0 {
				docScores[docKey].VectorScore = docWithScore.Score
			} else {
				docScores[docKey].KeywordScore = docWithScore.Score
			}
		}
	}
	
	// 转换为切片并排序
	var results []HybridSearchResult
	for _, result := range docScores {
		results = append(results, *result)
	}
	
	// 按 RRF 融合分数降序排序
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].FusionScore > results[i].FusionScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	
	// 返回 top-K 结果
	if len(results) > topK {
		results = results[:topK]
	}
	
	return results
}

// MultiVectorSearch 多向量搜索 (可用于实现更复杂的混合检索)
// 例如:可以用不同的查询向量(不同的 embedding 模型)进行搜索,然后 RRF 融合
func (store *MilvusVectorStore) MultiVectorSearch(ctx context.Context, queries []string, k int, opts *HybridSearchOptions) ([]HybridSearchResult, error) {
	if opts == nil {
		opts = &HybridSearchOptions{
			RRFRankConstant: 60,
		}
	}
	
	// 对每个查询执行搜索
	var allResults [][]DocumentWithScore
	for _, query := range queries {
		results, err := store.SimilaritySearchWithScore(ctx, query, k*2)
		if err != nil {
			return nil, fmt.Errorf("search for query '%s' failed: %w", query, err)
		}
		allResults = append(allResults, results)
	}
	
	// RRF 融合
	results := store.applyRRF(allResults, opts.RRFRankConstant, k)
	
	return results, nil
}

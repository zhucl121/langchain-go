package vectorstores

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// ChromaVectorStore 实现 Chroma 向量存储集成
//
// Chroma 是一个开源的嵌入式向量数据库，专为 AI 应用设计。
//
// 特点:
//   - 轻量级，易于部署
//   - 支持持久化和内存模式
//   - 简单的 REST API
//   - 支持多种距离度量
//
// 使用示例:
//
//	config := vectorstores.ChromaConfig{
//	    URL:            "http://localhost:8000",
//	    CollectionName: "my_collection",
//	}
//	store := vectorstores.NewChromaVectorStore(config, embedder)
//
type ChromaVectorStore struct {
	config    ChromaConfig
	embedder  embeddings.Embeddings
	httpClient *http.Client
	
	// 集合信息缓存
	collectionID string
}

// ChromaConfig 是 Chroma 的配置
type ChromaConfig struct {
	// URL Chroma 服务器地址
	URL string
	
	// CollectionName 集合名称
	CollectionName string
	
	// Metadata 集合元数据
	Metadata map[string]interface{}
	
	// DistanceMetric 距离度量方式
	// 支持: "l2" (欧几里得距离), "ip" (内积), "cosine" (余弦相似度)
	DistanceMetric string
	
	// HTTPClient 自定义 HTTP 客户端
	HTTPClient *http.Client
	
	// Timeout 请求超时时间
	Timeout time.Duration
}

// DefaultChromaConfig 返回默认配置
func DefaultChromaConfig() ChromaConfig {
	return ChromaConfig{
		URL:            "http://localhost:8000",
		CollectionName: "langchain_collection",
		DistanceMetric: "l2",
		Timeout:        30 * time.Second,
	}
}

// NewChromaVectorStore 创建 Chroma 向量存储实例
//
// 参数:
//   - config: Chroma 配置
//   - embedder: 嵌入模型
//
// 返回:
//   - *ChromaVectorStore: Chroma 向量存储实例
//   - error: 错误
//
func NewChromaVectorStore(config ChromaConfig, embedder embeddings.Embeddings) (*ChromaVectorStore, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("chroma: URL is required")
	}
	
	if config.CollectionName == "" {
		return nil, fmt.Errorf("chroma: collection name is required")
	}
	
	if embedder == nil {
		return nil, fmt.Errorf("chroma: embedder is required")
	}
	
	// 设置默认值
	if config.DistanceMetric == "" {
		config.DistanceMetric = "l2"
	}
	
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}
	
	store := &ChromaVectorStore{
		config:     config,
		embedder:   embedder,
		httpClient: httpClient,
	}
	
	return store, nil
}

// Initialize 初始化 Chroma 集合
//
// 如果集合不存在则创建，如果存在则获取集合信息
//
func (c *ChromaVectorStore) Initialize(ctx context.Context) error {
	// 尝试获取集合
	collection, err := c.getCollection(ctx)
	if err == nil {
		c.collectionID = collection.ID
		return nil
	}
	
	// 集合不存在，创建新集合
	return c.createCollection(ctx)
}

// AddDocuments 添加文档到向量存储
//
// 参数:
//   - ctx: 上下文
//   - docs: 文档列表
//
// 返回:
//   - []string: 文档 ID 列表
//   - error: 错误
//
func (c *ChromaVectorStore) AddDocuments(ctx context.Context, docs []*loaders.Document) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}
	
	// 确保集合已初始化
	if c.collectionID == "" {
		if err := c.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("chroma: failed to initialize: %w", err)
		}
	}
	
	// 提取文档内容
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	
	// 生成嵌入向量
	embeddings, err := c.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("chroma: failed to embed documents: %w", err)
	}
	
	// 准备文档 ID 和元数据
	ids := make([]string, len(docs))
	metadatas := make([]map[string]interface{}, len(docs))
	
	for i, doc := range docs {
		// 生成或使用文档 ID
		if id, ok := doc.Metadata["id"].(string); ok {
			ids[i] = id
		} else {
		ids[i] = generateChromaID()
		}
		
		// 准备元数据
		meta := make(map[string]interface{})
		for k, v := range doc.Metadata {
			// Chroma 只支持基本类型
			switch v.(type) {
			case string, int, int64, float32, float64, bool:
				meta[k] = v
			default:
				// 转换为字符串
				meta[k] = fmt.Sprintf("%v", v)
			}
		}
		metadatas[i] = meta
	}
	
	// 调用 Chroma API 添加文档
	if err := c.addEmbeddings(ctx, ids, embeddings, texts, metadatas); err != nil {
		return nil, fmt.Errorf("chroma: failed to add embeddings: %w", err)
	}
	
	return ids, nil
}

// SimilaritySearch 相似度搜索
//
// 参数:
//   - ctx: 上下文
//   - query: 查询文本
//   - k: 返回结果数量
//
// 返回:
//   - []*loaders.Document: 相似文档列表（包含分数）
//   - error: 错误
//
func (c *ChromaVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("chroma: query is required")
	}
	
	if k <= 0 {
		k = 4 // 默认返回 4 个结果
	}
	
	// 确保集合已初始化
	if c.collectionID == "" {
		if err := c.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("chroma: failed to initialize: %w", err)
		}
	}
	
	// 生成查询向量
	embeddings, err := c.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("chroma: failed to embed query: %w", err)
	}
	
	// 执行查询
	results, err := c.queryEmbeddings(ctx, embeddings, k)
	if err != nil {
		return nil, fmt.Errorf("chroma: failed to query: %w", err)
	}
	
	// 转换为文档格式
	docs := make([]*loaders.Document, 0, len(results.IDs))
	for i := range results.IDs {
		doc := &loaders.Document{
			Content:  results.Documents[i],
			Metadata: make(map[string]interface{}),
		}
		
		// 添加 ID
		doc.Metadata["id"] = results.IDs[i]
		
		// 添加分数
		if i < len(results.Distances) {
			doc.Metadata["score"] = c.convertDistance(results.Distances[i])
		}
		
		// 添加原始元数据
		if i < len(results.Metadatas) && results.Metadatas[i] != nil {
			for k, v := range results.Metadatas[i] {
				doc.Metadata[k] = v
			}
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// SimilaritySearchWithScore 带分数的相似度搜索
//
// 参数:
//   - ctx: 上下文
//   - query: 查询文本
//   - k: 返回结果数量
//   - scoreThreshold: 分数阈值（0-1，越高越相似）
//
// 返回:
//   - []*loaders.Document: 相似文档列表
//   - error: 错误
//
func (c *ChromaVectorStore) SimilaritySearchWithScore(
	ctx context.Context,
	query string,
	k int,
	scoreThreshold float32,
) ([]*loaders.Document, error) {
	// 执行搜索
	docs, err := c.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, err
	}
	
	// 过滤低分文档
	filtered := make([]*loaders.Document, 0, len(docs))
	for _, doc := range docs {
		if score, ok := doc.Metadata["score"].(float32); ok {
			if score >= scoreThreshold {
				filtered = append(filtered, doc)
			}
		}
	}
	
	return filtered, nil
}

// DeleteDocuments 删除文档
//
// 参数:
//   - ctx: 上下文
//   - ids: 文档 ID 列表
//
// 返回:
//   - error: 错误
//
func (c *ChromaVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	
	// 确保集合已初始化
	if c.collectionID == "" {
		if err := c.Initialize(ctx); err != nil {
			return fmt.Errorf("chroma: failed to initialize: %w", err)
		}
	}
	
	// 构建请求
	reqBody := map[string]interface{}{
		"ids": ids,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("chroma: failed to marshal request: %w", err)
	}
	
	// 发送删除请求
	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", c.config.URL, c.collectionID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("chroma: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("chroma: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma: delete failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// GetCollectionInfo 获取集合信息
func (c *ChromaVectorStore) GetCollectionInfo(ctx context.Context) (*ChromaCollection, error) {
	if c.collectionID == "" {
		if err := c.Initialize(ctx); err != nil {
			return nil, err
		}
	}
	
	return c.getCollection(ctx)
}

// ==================== 内部方法 ====================

// createCollection 创建集合
func (c *ChromaVectorStore) createCollection(ctx context.Context) error {
	reqBody := map[string]interface{}{
		"name": c.config.CollectionName,
		"metadata": map[string]interface{}{
			"hnsw:space": c.config.DistanceMetric,
		},
	}
	
	// 添加用户自定义元数据
	if c.config.Metadata != nil {
		for k, v := range c.config.Metadata {
			reqBody["metadata"].(map[string]interface{})[k] = v
		}
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("chroma: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/api/v1/collections", c.config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("chroma: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("chroma: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma: create collection failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应获取集合 ID
	var collection ChromaCollection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return fmt.Errorf("chroma: failed to decode response: %w", err)
	}
	
	c.collectionID = collection.ID
	return nil
}

// getCollection 获取集合信息
func (c *ChromaVectorStore) getCollection(ctx context.Context) (*ChromaCollection, error) {
	url := fmt.Sprintf("%s/api/v1/collections/%s", c.config.URL, c.config.CollectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("chroma: failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chroma: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("chroma: collection not found")
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chroma: get collection failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var collection ChromaCollection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, fmt.Errorf("chroma: failed to decode response: %w", err)
	}
	
	return &collection, nil
}

// addEmbeddings 添加嵌入向量
func (c *ChromaVectorStore) addEmbeddings(
	ctx context.Context,
	ids []string,
	embeddings [][]float32,
	documents []string,
	metadatas []map[string]interface{},
) error {
	reqBody := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"documents":  documents,
		"metadatas":  metadatas,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("chroma: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/api/v1/collections/%s/add", c.config.URL, c.collectionID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("chroma: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("chroma: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma: add embeddings failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// queryEmbeddings 查询嵌入向量
func (c *ChromaVectorStore) queryEmbeddings(
	ctx context.Context,
	queryEmbedding []float32,
	k int,
) (*ChromaQueryResult, error) {
	reqBody := map[string]interface{}{
		"query_embeddings": [][]float32{queryEmbedding},
		"n_results":        k,
		"include":          []string{"metadatas", "documents", "distances"},
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("chroma: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/api/v1/collections/%s/query", c.config.URL, c.collectionID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("chroma: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chroma: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chroma: query failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var rawResult struct {
		IDs        [][]string                   `json:"ids"`
		Distances  [][]float32                  `json:"distances"`
		Documents  [][]string                   `json:"documents"`
		Metadatas  [][]map[string]interface{}   `json:"metadatas"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, fmt.Errorf("chroma: failed to decode response: %w", err)
	}
	
	// 展平结果
	result := &ChromaQueryResult{}
	if len(rawResult.IDs) > 0 {
		result.IDs = rawResult.IDs[0]
		result.Distances = rawResult.Distances[0]
		result.Documents = rawResult.Documents[0]
		result.Metadatas = rawResult.Metadatas[0]
	}
	
	return result, nil
}

// convertDistance 转换距离为相似度分数
//
// Chroma 返回的是距离，我们需要转换为相似度分数（0-1，越高越相似）
//
func (c *ChromaVectorStore) convertDistance(distance float32) float32 {
	switch c.config.DistanceMetric {
	case "l2":
		// L2 距离：distance 越小越相似
		// 使用指数函数转换: score = exp(-distance)
		// 限制在 [0, 1] 范围
		score := float32(1.0) / (1.0 + distance)
		return score
		
	case "ip":
		// 内积：值越大越相似，已经是相似度
		return distance
		
	case "cosine":
		// 余弦距离：distance = 1 - cosine_similarity
		// 所以 similarity = 1 - distance
		return 1.0 - distance
		
	default:
		// 默认使用 L2 转换
		return float32(1.0) / (1.0 + distance)
	}
}

// ==================== 辅助类型 ====================

// ChromaCollection Chroma 集合信息
type ChromaCollection struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ChromaQueryResult Chroma 查询结果
type ChromaQueryResult struct {
	IDs       []string                 `json:"ids"`
	Distances []float32                `json:"distances"`
	Documents []string                 `json:"documents"`
	Metadatas []map[string]interface{} `json:"metadatas"`
}

// generateChromaID 生成唯一 ID
func generateChromaID() string {
	return fmt.Sprintf("doc_%d", time.Now().UnixNano())
}

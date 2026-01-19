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

// QdrantVectorStore 实现 Qdrant 向量存储集成
//
// Qdrant 是一个高性能的向量相似度搜索引擎，采用 Rust 编写。
//
// 特点:
//   - 高性能 (使用 Rust 编写)
//   - 支持过滤和有效载荷
//   - 支持多种距离度量
//   - RESTful API 和 gRPC支持
//
// 使用示例:
//
//	config := vectorstores.QdrantConfig{
//	    URL:            "http://localhost:6333",
//	    CollectionName: "my_collection",
//	    VectorSize:     384,
//	}
//	store := vectorstores.NewQdrantVectorStore(config, embedder)
//
type QdrantVectorStore struct {
	config     QdrantConfig
	embedder   embeddings.Embeddings
	httpClient *http.Client
}

// QdrantConfig 是 Qdrant 的配置
type QdrantConfig struct {
	// URL Qdrant 服务器地址
	URL string
	
	// APIKey API 密钥 (如果需要)
	APIKey string
	
	// CollectionName 集合名称
	CollectionName string
	
	// VectorSize 向量维度
	VectorSize int
	
	// Distance 距离度量方式
	// 支持: "Cosine", "Euclid", "Dot"
	Distance string
	
	// OnDiskPayload 是否将有效载荷存储在磁盘上
	OnDiskPayload bool
	
	// HTTPClient 自定义 HTTP 客户端
	HTTPClient *http.Client
	
	// Timeout 请求超时时间
	Timeout time.Duration
}

// DefaultQdrantConfig 返回默认配置
func DefaultQdrantConfig() QdrantConfig {
	return QdrantConfig{
		URL:            "http://localhost:6333",
		CollectionName: "langchain_collection",
		VectorSize:     384, // 常见的嵌入维度
		Distance:       "Cosine",
		OnDiskPayload:  false,
		Timeout:        30 * time.Second,
	}
}

// NewQdrantVectorStore 创建 Qdrant 向量存储实例
func NewQdrantVectorStore(config QdrantConfig, embedder embeddings.Embeddings) (*QdrantVectorStore, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("qdrant: URL is required")
	}
	
	if config.CollectionName == "" {
		return nil, fmt.Errorf("qdrant: collection name is required")
	}
	
	if config.VectorSize <= 0 {
		return nil, fmt.Errorf("qdrant: vector size must be positive")
	}
	
	if embedder == nil {
		return nil, fmt.Errorf("qdrant: embedder is required")
	}
	
	// 设置默认值
	if config.Distance == "" {
		config.Distance = "Cosine"
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
	
	return &QdrantVectorStore{
		config:     config,
		embedder:   embedder,
		httpClient: httpClient,
	}, nil
}

// Initialize 初始化 Qdrant 集合
func (q *QdrantVectorStore) Initialize(ctx context.Context) error {
	// 检查集合是否存在
	exists, err := q.collectionExists(ctx)
	if err != nil {
		return fmt.Errorf("qdrant: failed to check collection: %w", err)
	}
	
	if exists {
		return nil
	}
	
	// 创建集合
	return q.createCollection(ctx)
}

// AddDocuments 添加文档到向量存储
func (q *QdrantVectorStore) AddDocuments(ctx context.Context, docs []*loaders.Document) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}
	
	// 提取文档内容
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	
	// 生成嵌入向量
	embeddings, err := q.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("qdrant: failed to embed documents: %w", err)
	}
	
	// 准备点数据
	points := make([]QdrantPoint, len(docs))
	ids := make([]string, len(docs))
	
	for i, doc := range docs {
		// 生成或使用文档 ID
		if id, ok := doc.Metadata["id"].(string); ok {
			ids[i] = id
		} else {
			ids[i] = generateID(i)
		}
		
		// 准备有效载荷
		payload := map[string]interface{}{
			"document": doc.Content,
		}
		for k, v := range doc.Metadata {
			if k != "id" {
				payload[k] = v
			}
		}
		
		points[i] = QdrantPoint{
			ID:      ids[i],
			Vector:  embeddings[i],
			Payload: payload,
		}
	}
	
	// 上传点
	if err := q.upsertPoints(ctx, points); err != nil {
		return nil, fmt.Errorf("qdrant: failed to upsert points: %w", err)
	}
	
	return ids, nil
}

// SimilaritySearch 相似度搜索
func (q *QdrantVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("qdrant: query is required")
	}
	
	if k <= 0 {
		k = 4
	}
	
	// 生成查询向量
	embedding, err := q.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("qdrant: failed to embed query: %w", err)
	}
	
	// 执行搜索
	results, err := q.searchPoints(ctx, embedding, k, nil)
	if err != nil {
		return nil, fmt.Errorf("qdrant: failed to search: %w", err)
	}
	
	// 转换为文档格式
	docs := make([]*loaders.Document, 0, len(results))
	for _, result := range results {
		doc := &loaders.Document{
			Metadata: make(map[string]interface{}),
		}
		
		// 提取文档内容
		if content, ok := result.Payload["document"].(string); ok {
			doc.Content = content
		}
		
		// 添加分数
		doc.Metadata["score"] = result.Score
		doc.Metadata["id"] = result.ID
		
		// 添加其他元数据
		for k, v := range result.Payload {
			if k != "document" {
				doc.Metadata[k] = v
			}
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// SimilaritySearchWithScore 带分数的相似度搜索
func (q *QdrantVectorStore) SimilaritySearchWithScore(
	ctx context.Context,
	query string,
	k int,
	scoreThreshold float32,
) ([]*loaders.Document, error) {
	docs, err := q.SimilaritySearch(ctx, query, k)
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

// SearchWithFilter 带过滤条件的搜索
//
// filter 示例:
//
//	{
//	    "must": [
//	        {"key": "category", "match": {"value": "science"}}
//	    ]
//	}
//
func (q *QdrantVectorStore) SearchWithFilter(
	ctx context.Context,
	query string,
	k int,
	filter map[string]interface{},
) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("qdrant: query is required")
	}
	
	if k <= 0 {
		k = 4
	}
	
	// 生成查询向量
	embedding, err := q.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("qdrant: failed to embed query: %w", err)
	}
	
	// 执行带过滤的搜索
	results, err := q.searchPoints(ctx, embedding, k, filter)
	if err != nil {
		return nil, fmt.Errorf("qdrant: failed to search: %w", err)
	}
	
	// 转换为文档格式
	docs := make([]*loaders.Document, 0, len(results))
	for _, result := range results {
		doc := &loaders.Document{
			Metadata: make(map[string]interface{}),
		}
		
		if content, ok := result.Payload["document"].(string); ok {
			doc.Content = content
		}
		
		doc.Metadata["score"] = result.Score
		doc.Metadata["id"] = result.ID
		
		for k, v := range result.Payload {
			if k != "document" {
				doc.Metadata[k] = v
			}
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// DeleteDocuments 删除文档
func (q *QdrantVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	
	reqBody := map[string]interface{}{
		"points": ids,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("qdrant: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/collections/%s/points/delete", q.config.URL, q.config.CollectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("qdrant: failed to create request: %w", err)
	}
	
	q.setHeaders(req)
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant: delete failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// ==================== 内部方法 ====================

func (q *QdrantVectorStore) collectionExists(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/collections/%s", q.config.URL, q.config.CollectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("qdrant: failed to create request: %w", err)
	}
	
	q.setHeaders(req)
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("qdrant: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("qdrant: check collection failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return true, nil
}

func (q *QdrantVectorStore) createCollection(ctx context.Context) error {
	reqBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     q.config.VectorSize,
			"distance": q.config.Distance,
		},
		"on_disk_payload": q.config.OnDiskPayload,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("qdrant: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/collections/%s", q.config.URL, q.config.CollectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("qdrant: failed to create request: %w", err)
	}
	
	q.setHeaders(req)
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant: create collection failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (q *QdrantVectorStore) upsertPoints(ctx context.Context, points []QdrantPoint) error {
	reqBody := map[string]interface{}{
		"points": points,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("qdrant: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/collections/%s/points", q.config.URL, q.config.CollectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("qdrant: failed to create request: %w", err)
	}
	
	q.setHeaders(req)
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant: upsert points failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (q *QdrantVectorStore) searchPoints(
	ctx context.Context,
	vector []float32,
	limit int,
	filter map[string]interface{},
) ([]QdrantSearchResult, error) {
	reqBody := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	
	if filter != nil {
		reqBody["filter"] = filter
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("qdrant: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/collections/%s/points/search", q.config.URL, q.config.CollectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("qdrant: failed to create request: %w", err)
	}
	
	q.setHeaders(req)
	
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant: search failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var response struct {
		Result []QdrantSearchResult `json:"result"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("qdrant: failed to decode response: %w", err)
	}
	
	return response.Result, nil
}

func (q *QdrantVectorStore) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if q.config.APIKey != "" {
		req.Header.Set("api-key", q.config.APIKey)
	}
}

// ==================== 辅助类型 ====================

// QdrantPoint 表示 Qdrant 中的一个点
type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantSearchResult 表示 Qdrant 搜索结果
type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Score   float32                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

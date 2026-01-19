package vectorstores

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// WeaviateVectorStore 实现 Weaviate 向量存储集成
//
// Weaviate 是一个开源的云原生向量搜索引擎，支持多种模块和集成。
//
// 特点:
//   - GraphQL API
//   - 支持多租户
//   - 内置向量化模块
//   - 支持混合搜索
//   - 强大的过滤能力
//
// 使用示例:
//
//	config := vectorstores.WeaviateConfig{
//	    URL:       "http://localhost:8080",
//	    ClassName: "Document",
//	}
//	store := vectorstores.NewWeaviateVectorStore(config, embedder)
//
type WeaviateVectorStore struct {
	config     WeaviateConfig
	embedder   Embedder
	httpClient *http.Client
}

// WeaviateConfig 是 Weaviate 的配置
type WeaviateConfig struct {
	// URL Weaviate 服务器地址
	URL string
	
	// APIKey API 密钥（如果需要）
	APIKey string
	
	// ClassName 类名称（相当于集合）
	ClassName string
	
	// TextKey 文本内容的属性名
	TextKey string
	
	// VectorIndexConfig 向量索引配置
	VectorIndexConfig map[string]interface{}
	
	// Properties 额外的属性定义
	Properties []WeaviateProperty
	
	// Tenant 租户名称（多租户模式）
	Tenant string
	
	// HTTPClient 自定义 HTTP 客户端
	HTTPClient *http.Client
	
	// Timeout 请求超时时间
	Timeout time.Duration
}

// WeaviateProperty 定义 Weaviate 属性
type WeaviateProperty struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"` // "text", "string", "int", "number", "boolean", "date"
}

// DefaultWeaviateConfig 返回默认配置
func DefaultWeaviateConfig() WeaviateConfig {
	return WeaviateConfig{
		URL:       "http://localhost:8080",
		ClassName: "Document",
		TextKey:   "text",
		Properties: []WeaviateProperty{
			{Name: "text", DataType: "text"},
		},
		Timeout: 30 * time.Second,
	}
}

// NewWeaviateVectorStore 创建 Weaviate 向量存储实例
func NewWeaviateVectorStore(config WeaviateConfig, embedder Embedder) (*WeaviateVectorStore, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("weaviate: URL is required")
	}
	
	if config.ClassName == "" {
		return nil, fmt.Errorf("weaviate: class name is required")
	}
	
	if embedder == nil {
		return nil, fmt.Errorf("weaviate: embedder is required")
	}
	
	// 设置默认值
	if config.TextKey == "" {
		config.TextKey = "text"
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
	
	return &WeaviateVectorStore{
		config:     config,
		embedder:   embedder,
		httpClient: httpClient,
	}, nil
}

// Initialize 初始化 Weaviate 类（Schema）
func (w *WeaviateVectorStore) Initialize(ctx context.Context) error {
	// 检查类是否存在
	exists, err := w.classExists(ctx)
	if err != nil {
		return fmt.Errorf("weaviate: failed to check class: %w", err)
	}
	
	if exists {
		return nil
	}
	
	// 创建类
	return w.createClass(ctx)
}

// AddDocuments 添加文档到向量存储
func (w *WeaviateVectorStore) AddDocuments(ctx context.Context, docs []*loaders.Document) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}
	
	// 提取文档内容
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	
	// 生成嵌入向量
	embeddings, err := w.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("weaviate: failed to embed documents: %w", err)
	}
	
	// 准备对象
	objects := make([]WeaviateObject, len(docs))
	ids := make([]string, len(docs))
	
	for i, doc := range docs {
		// 生成或使用文档 ID
		if id, ok := doc.Metadata["id"].(string); ok {
			ids[i] = id
		} else {
			ids[i] = generateID()
		}
		
		// 准备属性
		properties := map[string]interface{}{
			w.config.TextKey: doc.Content,
		}
		
		// 添加其他元数据
		for k, v := range doc.Metadata {
			if k != "id" && k != w.config.TextKey {
				properties[k] = w.convertValue(v)
			}
		}
		
		objects[i] = WeaviateObject{
			Class:      w.config.ClassName,
			ID:         ids[i],
			Properties: properties,
			Vector:     embeddings[i],
		}
		
		// 添加租户信息
		if w.config.Tenant != "" {
			objects[i].Tenant = w.config.Tenant
		}
	}
	
	// 批量创建对象
	if err := w.batchCreateObjects(ctx, objects); err != nil {
		return nil, fmt.Errorf("weaviate: failed to create objects: %w", err)
	}
	
	return ids, nil
}

// SimilaritySearch 相似度搜索
func (w *WeaviateVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("weaviate: query is required")
	}
	
	if k <= 0 {
		k = 4
	}
	
	// 生成查询向量
	embedding, err := w.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("weaviate: failed to embed query: %w", err)
	}
	
	// 执行搜索
	results, err := w.searchByVector(ctx, embedding, k, nil)
	if err != nil {
		return nil, fmt.Errorf("weaviate: failed to search: %w", err)
	}
	
	// 转换为文档格式
	return w.resultsToDocuments(results), nil
}

// SimilaritySearchWithScore 带分数的相似度搜索
func (w *WeaviateVectorStore) SimilaritySearchWithScore(
	ctx context.Context,
	query string,
	k int,
	scoreThreshold float32,
) ([]*loaders.Document, error) {
	docs, err := w.SimilaritySearch(ctx, query, k)
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
// 使用 Weaviate 的 GraphQL where 过滤器
//
// filter 示例:
//
//	{
//	    "operator": "Equal",
//	    "path": ["category"],
//	    "valueString": "science"
//	}
//
func (w *WeaviateVectorStore) SearchWithFilter(
	ctx context.Context,
	query string,
	k int,
	filter map[string]interface{},
) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("weaviate: query is required")
	}
	
	if k <= 0 {
		k = 4
	}
	
	// 生成查询向量
	embedding, err := w.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("weaviate: failed to embed query: %w", err)
	}
	
	// 执行带过滤的搜索
	results, err := w.searchByVector(ctx, embedding, k, filter)
	if err != nil {
		return nil, fmt.Errorf("weaviate: failed to search: %w", err)
	}
	
	return w.resultsToDocuments(results), nil
}

// HybridSearch 混合搜索（向量 + BM25）
func (w *WeaviateVectorStore) HybridSearch(
	ctx context.Context,
	query string,
	k int,
	alpha float32, // 0 = pure BM25, 1 = pure vector
) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("weaviate: query is required")
	}
	
	if k <= 0 {
		k = 4
	}
	
	// 生成查询向量
	embedding, err := w.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("weaviate: failed to embed query: %w", err)
	}
	
	// 构建 GraphQL 查询
	gqlQuery := w.buildHybridQuery(query, embedding, k, alpha, nil)
	
	// 执行查询
	results, err := w.executeGraphQL(ctx, gqlQuery)
	if err != nil {
		return nil, fmt.Errorf("weaviate: failed to execute hybrid search: %w", err)
	}
	
	return w.parseGraphQLResults(results), nil
}

// DeleteDocuments 删除文档
func (w *WeaviateVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	
	// Weaviate 需要逐个删除
	for _, id := range ids {
		if err := w.deleteObject(ctx, id); err != nil {
			return fmt.Errorf("weaviate: failed to delete object %s: %w", id, err)
		}
	}
	
	return nil
}

// ==================== 内部方法 ====================

func (w *WeaviateVectorStore) classExists(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/v1/schema/%s", w.config.URL, w.config.ClassName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}
	
	w.setHeaders(req)
	
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	
	return true, nil
}

func (w *WeaviateVectorStore) createClass(ctx context.Context) error {
	schema := map[string]interface{}{
		"class": w.config.ClassName,
		"properties": w.config.Properties,
	}
	
	// 添加向量索引配置
	if w.config.VectorIndexConfig != nil {
		schema["vectorIndexConfig"] = w.config.VectorIndexConfig
	}
	
	// 添加多租户配置
	if w.config.Tenant != "" {
		schema["multiTenancyConfig"] = map[string]interface{}{
			"enabled": true,
		}
	}
	
	bodyBytes, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	
	url := fmt.Sprintf("%s/v1/schema", w.config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	
	w.setHeaders(req)
	
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (w *WeaviateVectorStore) batchCreateObjects(ctx context.Context, objects []WeaviateObject) error {
	reqBody := map[string]interface{}{
		"objects": objects,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	
	url := fmt.Sprintf("%s/v1/batch/objects", w.config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	
	w.setHeaders(req)
	
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (w *WeaviateVectorStore) searchByVector(
	ctx context.Context,
	vector []float32,
	limit int,
	filter map[string]interface{},
) ([]WeaviateSearchResult, error) {
	gqlQuery := w.buildVectorQuery(vector, limit, filter)
	
	results, err := w.executeGraphQL(ctx, gqlQuery)
	if err != nil {
		return nil, err
	}
	
	return w.parseSearchResults(results), nil
}

func (w *WeaviateVectorStore) buildVectorQuery(
	vector []float32,
	limit int,
	filter map[string]interface{},
) string {
	// 构建 GraphQL 查询
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf(`{
  Get {
    %s(
      nearVector: {
        vector: %s
      }
      limit: %d`, w.config.ClassName, w.formatVector(vector), limit))
	
	// 添加过滤条件
	if filter != nil {
		filterJSON, _ := json.Marshal(filter)
		builder.WriteString(fmt.Sprintf(`
      where: %s`, string(filterJSON)))
	}
	
	// 添加租户
	if w.config.Tenant != "" {
		builder.WriteString(fmt.Sprintf(`
      tenant: "%s"`, w.config.Tenant))
	}
	
	builder.WriteString(`
    ) {
      _additional {
        id
        distance
      }`)
	
	// 添加属性
	for _, prop := range w.config.Properties {
		builder.WriteString(fmt.Sprintf(`
      %s`, prop.Name))
	}
	
	builder.WriteString(`
    }
  }
}`)
	
	return builder.String()
}

func (w *WeaviateVectorStore) buildHybridQuery(
	query string,
	vector []float32,
	limit int,
	alpha float32,
	filter map[string]interface{},
) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf(`{
  Get {
    %s(
      hybrid: {
        query: "%s"
        vector: %s
        alpha: %.2f
      }
      limit: %d`, w.config.ClassName, query, w.formatVector(vector), alpha, limit))
	
	if filter != nil {
		filterJSON, _ := json.Marshal(filter)
		builder.WriteString(fmt.Sprintf(`
      where: %s`, string(filterJSON)))
	}
	
	if w.config.Tenant != "" {
		builder.WriteString(fmt.Sprintf(`
      tenant: "%s"`, w.config.Tenant))
	}
	
	builder.WriteString(`
    ) {
      _additional {
        id
        score
      }`)
	
	for _, prop := range w.config.Properties {
		builder.WriteString(fmt.Sprintf(`
      %s`, prop.Name))
	}
	
	builder.WriteString(`
    }
  }
}`)
	
	return builder.String()
}

func (w *WeaviateVectorStore) executeGraphQL(ctx context.Context, query string) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"query": query,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/v1/graphql", w.config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	
	w.setHeaders(req)
	
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}

func (w *WeaviateVectorStore) deleteObject(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/v1/objects/%s/%s", w.config.URL, w.config.ClassName, id)
	
	// 添加租户参数
	if w.config.Tenant != "" {
		url += fmt.Sprintf("?tenant=%s", w.config.Tenant)
	}
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	
	w.setHeaders(req)
	
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func (w *WeaviateVectorStore) parseSearchResults(results map[string]interface{}) []WeaviateSearchResult {
	// 解析 GraphQL 响应
	data, ok := results["data"].(map[string]interface{})
	if !ok {
		return []WeaviateSearchResult{}
	}
	
	get, ok := data["Get"].(map[string]interface{})
	if !ok {
		return []WeaviateSearchResult{}
	}
	
	objects, ok := get[w.config.ClassName].([]interface{})
	if !ok {
		return []WeaviateSearchResult{}
	}
	
	searchResults := make([]WeaviateSearchResult, 0, len(objects))
	for _, obj := range objects {
		objMap, ok := obj.(map[string]interface{})
		if !ok {
			continue
		}
		
		result := WeaviateSearchResult{
			Properties: make(map[string]interface{}),
		}
		
		// 提取 _additional 信息
		if additional, ok := objMap["_additional"].(map[string]interface{}); ok {
			if id, ok := additional["id"].(string); ok {
				result.ID = id
			}
			if distance, ok := additional["distance"].(float64); ok {
				result.Distance = float32(distance)
			}
			if score, ok := additional["score"].(float64); ok {
				result.Score = float32(score)
			}
		}
		
		// 提取属性
		for k, v := range objMap {
			if k != "_additional" {
				result.Properties[k] = v
			}
		}
		
		searchResults = append(searchResults, result)
	}
	
	return searchResults
}

func (w *WeaviateVectorStore) parseGraphQLResults(results map[string]interface{}) []*loaders.Document {
	searchResults := w.parseSearchResults(results)
	return w.resultsToDocuments(searchResults)
}

func (w *WeaviateVectorStore) resultsToDocuments(results []WeaviateSearchResult) []*loaders.Document {
	docs := make([]*loaders.Document, 0, len(results))
	
	for _, result := range results {
		doc := &loaders.Document{
			Metadata: make(map[string]interface{}),
		}
		
		// 提取文本内容
		if text, ok := result.Properties[w.config.TextKey].(string); ok {
			doc.Content = text
		}
		
		// 添加分数
		if result.Distance > 0 {
			// 转换距离为相似度分数（0-1，越高越相似）
			doc.Metadata["score"] = float32(1.0) / (1.0 + result.Distance)
		} else if result.Score > 0 {
			doc.Metadata["score"] = result.Score
		}
		
		doc.Metadata["id"] = result.ID
		
		// 添加其他属性
		for k, v := range result.Properties {
			if k != w.config.TextKey {
				doc.Metadata[k] = v
			}
		}
		
		docs = append(docs, doc)
	}
	
	return docs
}

func (w *WeaviateVectorStore) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if w.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.config.APIKey))
	}
}

func (w *WeaviateVectorStore) formatVector(vector []float32) string {
	parts := make([]string, len(vector))
	for i, v := range vector {
		parts[i] = fmt.Sprintf("%.6f", v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (w *WeaviateVectorStore) convertValue(v interface{}) interface{} {
	// Weaviate 需要特定的数据类型
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float32:
		return float64(val)
	case float64:
		return val
	case string:
		return val
	case bool:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ==================== 辅助类型 ====================

// WeaviateObject 表示 Weaviate 对象
type WeaviateObject struct {
	Class      string                 `json:"class"`
	ID         string                 `json:"id,omitempty"`
	Properties map[string]interface{} `json:"properties"`
	Vector     []float32              `json:"vector,omitempty"`
	Tenant     string                 `json:"tenant,omitempty"`
}

// WeaviateSearchResult 表示 Weaviate 搜索结果
type WeaviateSearchResult struct {
	ID         string
	Distance   float32
	Score      float32
	Properties map[string]interface{}
}

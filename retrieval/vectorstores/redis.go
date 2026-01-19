package vectorstores

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// RedisVectorStore 实现 Redis Vector 向量存储集成
//
// Redis Stack 提供了向量相似度搜索功能（RediSearch 模块）。
//
// 特点:
//   - 基于内存的高性能搜索
//   - 支持多种距离度量
//   - 支持混合查询（向量 + 过滤）
//   - 利用 Redis 的持久化和集群能力
//
// 使用示例:
//
//	config := vectorstores.RedisConfig{
//	    URL:        "redis://localhost:6379",
//	    IndexName:  "documents",
//	    VectorDim:  384,
//	}
//	store := vectorstores.NewRedisVectorStore(config, embedder)
//
type RedisVectorStore struct {
	config   RedisConfig
	embedder embeddings.Embeddings
	client   RedisClient
}

// RedisConfig 是 Redis 的配置
type RedisConfig struct {
	// URL Redis 连接字符串
	URL string
	
	// Password Redis 密码
	Password string
	
	// DB 数据库编号
	DB int
	
	// IndexName 索引名称
	IndexName string
	
	// Prefix 键前缀
	Prefix string
	
	// VectorDim 向量维度
	VectorDim int
	
	// DistanceMetric 距离度量
	// 支持: "COSINE", "IP" (内积), "L2"
	DistanceMetric string
	
	// VectorAlgorithm 向量算法
	// "FLAT" 或 "HNSW"
	VectorAlgorithm string
	
	// HNSWConfig HNSW 配置（如果使用 HNSW）
	HNSWConfig *RedisHNSWConfig
}

// RedisHNSWConfig HNSW 算法配置
type RedisHNSWConfig struct {
	M              int // HNSW M 参数
	EFConstruction int // 构建时的 ef 参数
	EFRuntime      int // 运行时的 ef 参数
}

// DefaultRedisConfig 返回默认配置
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		URL:             "redis://localhost:6379",
		DB:              0,
		IndexName:       "langchain_index",
		Prefix:          "doc:",
		VectorDim:       384,
		DistanceMetric:  "COSINE",
		VectorAlgorithm: "FLAT",
	}
}

// RedisClient Redis 客户端接口（允许使用不同的 Redis 客户端库）
type RedisClient interface {
	// Do 执行 Redis 命令
	Do(ctx context.Context, args ...interface{}) (interface{}, error)
	
	// Close 关闭连接
	Close() error
}

// NewRedisVectorStore 创建 Redis 向量存储实例
//
// 注意：需要提供一个实现了 RedisClient 接口的 Redis 客户端
// 推荐使用 github.com/redis/go-redis 或 github.com/gomodule/redigo
//
func NewRedisVectorStore(config RedisConfig, embedder embeddings.Embeddings, client RedisClient) (*RedisVectorStore, error) {
	if config.IndexName == "" {
		return nil, fmt.Errorf("redis: index name is required")
	}
	
	if config.VectorDim <= 0 {
		return nil, fmt.Errorf("redis: vector dimension must be positive")
	}
	
	if embedder == nil {
		return nil, fmt.Errorf("redis: embedder is required")
	}
	
	if client == nil {
		return nil, fmt.Errorf("redis: redis client is required")
	}
	
	// 设置默认值
	if config.Prefix == "" {
		config.Prefix = "doc:"
	}
	
	if config.DistanceMetric == "" {
		config.DistanceMetric = "COSINE"
	}
	
	if config.VectorAlgorithm == "" {
		config.VectorAlgorithm = "FLAT"
	}
	
	return &RedisVectorStore{
		config:   config,
		embedder: embedder,
		client:   client,
	}, nil
}

// Initialize 初始化 Redis 索引
func (r *RedisVectorStore) Initialize(ctx context.Context) error {
	// 检查索引是否存在
	exists, err := r.indexExists(ctx)
	if err != nil {
		return fmt.Errorf("redis: failed to check index: %w", err)
	}
	
	if exists {
		return nil
	}
	
	// 创建索引
	return r.createIndex(ctx)
}

// AddDocuments 添加文档到向量存储
func (r *RedisVectorStore) AddDocuments(ctx context.Context, docs []*loaders.Document) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}
	
	// 提取文档内容
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	
	// 生成嵌入向量
	embeddings, err := r.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("redis: failed to embed documents: %w", err)
	}
	
	// 存储文档
	ids := make([]string, len(docs))
	for i, doc := range docs {
		// 生成或使用文档 ID
		if id, ok := doc.Metadata["id"].(string); ok {
			ids[i] = id
		} else {
			ids[i] = generateID(i)
		}
		
		// 准备数据
		key := r.config.Prefix + ids[i]
		
		// 构建 HSET 命令参数
		args := []interface{}{"HSET", key}
		
		// 添加文本内容
		args = append(args, "content", doc.Content)
		
		// 添加向量
		vectorBytes := r.serializeVector(embeddings[i])
		args = append(args, "vector", vectorBytes)
		
		// 添加元数据
		for k, v := range doc.Metadata {
			if k != "id" {
				args = append(args, k, r.serializeValue(v))
			}
		}
		
		// 执行存储
		if _, err := r.client.Do(ctx, args...); err != nil {
			return nil, fmt.Errorf("redis: failed to store document %s: %w", ids[i], err)
		}
	}
	
	return ids, nil
}

// SimilaritySearch 相似度搜索
func (r *RedisVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("redis: query is required")
	}
	
	if k <= 0 {
		k = 4
	}
	
	// 生成查询向量
	embedding, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("redis: failed to embed query: %w", err)
	}
	
	// 执行搜索
	return r.searchByVector(ctx, embedding, k, "")
}

// SimilaritySearchWithScore 带分数的相似度搜索
func (r *RedisVectorStore) SimilaritySearchWithScore(
	ctx context.Context,
	query string,
	k int,
	scoreThreshold float32,
) ([]*loaders.Document, error) {
	docs, err := r.SimilaritySearch(ctx, query, k)
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
// filter 使用 RediSearch 的过滤语法，例如:
//   "@category:{science}"
//   "@year:[2020 2023]"
//
func (r *RedisVectorStore) SearchWithFilter(
	ctx context.Context,
	query string,
	k int,
	filter string,
) ([]*loaders.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("redis: query is required")
	}
	
	if k <= 0 {
		k = 4
	}
	
	// 生成查询向量
	embedding, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("redis: failed to embed query: %w", err)
	}
	
	// 执行带过滤的搜索
	return r.searchByVector(ctx, embedding, k, filter)
}

// DeleteDocuments 删除文档
func (r *RedisVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	
	// 批量删除
	keys := make([]interface{}, len(ids)+1)
	keys[0] = "DEL"
	for i, id := range ids {
		keys[i+1] = r.config.Prefix + id
	}
	
	_, err := r.client.Do(ctx, keys...)
	if err != nil {
		return fmt.Errorf("redis: failed to delete documents: %w", err)
	}
	
	return nil
}

// ==================== 内部方法 ====================

func (r *RedisVectorStore) indexExists(ctx context.Context) (bool, error) {
	result, err := r.client.Do(ctx, "FT._LIST")
	if err != nil {
		return false, fmt.Errorf("redis: failed to list indices: %w", err)
	}
	
	// 检查索引列表
	indices, ok := result.([]interface{})
	if !ok {
		return false, nil
	}
	
	for _, index := range indices {
		if indexName, ok := index.(string); ok && indexName == r.config.IndexName {
			return true, nil
		}
	}
	
	return false, nil
}

func (r *RedisVectorStore) createIndex(ctx context.Context) error {
	// 构建 FT.CREATE 命令
	args := []interface{}{
		"FT.CREATE", r.config.IndexName,
		"ON", "HASH",
		"PREFIX", "1", r.config.Prefix,
		"SCHEMA",
		"content", "TEXT",
	}
	
	// 添加向量字段
	args = append(args,
		"vector", "VECTOR",
		r.config.VectorAlgorithm,
	)
	
	// 添加向量参数
	vectorParams := []string{
		"TYPE", "FLOAT32",
		"DIM", strconv.Itoa(r.config.VectorDim),
		"DISTANCE_METRIC", r.config.DistanceMetric,
	}
	
	// 如果使用 HNSW，添加额外参数
	if r.config.VectorAlgorithm == "HNSW" && r.config.HNSWConfig != nil {
		vectorParams = append(vectorParams,
			"M", strconv.Itoa(r.config.HNSWConfig.M),
			"EF_CONSTRUCTION", strconv.Itoa(r.config.HNSWConfig.EFConstruction),
			"EF_RUNTIME", strconv.Itoa(r.config.HNSWConfig.EFRuntime),
		)
	}
	
	args = append(args, len(vectorParams))
	for _, param := range vectorParams {
		args = append(args, param)
	}
	
	// 执行创建命令
	_, err := r.client.Do(ctx, args...)
	if err != nil {
		return fmt.Errorf("redis: failed to create index: %w", err)
	}
	
	return nil
}

func (r *RedisVectorStore) searchByVector(
	ctx context.Context,
	vector []float32,
	k int,
	filter string,
) ([]*loaders.Document, error) {
	// 序列化查询向量
	vectorBytes := r.serializeVector(vector)
	
	// 构建查询
	query := "*"
	if filter != "" {
		query = filter
	}
	
	// 构建 FT.SEARCH 命令
	args := []interface{}{
		"FT.SEARCH", r.config.IndexName,
		query,
		"LIMIT", "0", strconv.Itoa(k),
		"SORTBY", "__vector_score",
		"PARAMS", "2", "vector_query", vectorBytes,
		"DIALECT", "2",
	}
	
	// 执行搜索
	result, err := r.client.Do(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("redis: search failed: %w", err)
	}
	
	// 解析结果
	return r.parseSearchResults(result)
}

func (r *RedisVectorStore) parseSearchResults(result interface{}) ([]*loaders.Document, error) {
	// Redis 返回格式: [总数, key1, [field1, value1, ...], key2, [field2, value2, ...], ...]
	results, ok := result.([]interface{})
	if !ok || len(results) < 1 {
		return []*loaders.Document{}, nil
	}
	
	// 第一个元素是总数
	totalCount, ok := results[0].(int64)
	if !ok {
		return []*loaders.Document{}, nil
	}
	
	if totalCount == 0 {
		return []*loaders.Document{}, nil
	}
	
	// 解析文档
	docs := make([]*loaders.Document, 0, totalCount)
	
	for i := 1; i < len(results); i += 2 {
		if i+1 >= len(results) {
			break
		}
		
		// 获取 key
		key, ok := results[i].(string)
		if !ok {
			continue
		}
		
		// 获取字段
		fields, ok := results[i+1].([]interface{})
		if !ok {
			continue
		}
		
		doc := &loaders.Document{
			Metadata: make(map[string]interface{}),
		}
		
		// 解析字段
		for j := 0; j < len(fields); j += 2 {
			if j+1 >= len(fields) {
				break
			}
			
			fieldName, ok := fields[j].(string)
			if !ok {
				continue
			}
			
			fieldValue := fields[j+1]
			
			switch fieldName {
			case "content":
				if content, ok := fieldValue.(string); ok {
					doc.Content = content
				}
			case "vector":
				// 跳过向量数据
				continue
			case "__vector_score":
				if score, ok := fieldValue.(string); ok {
					if scoreFloat, err := strconv.ParseFloat(score, 32); err == nil {
						// 转换距离为相似度分数
						doc.Metadata["score"] = r.convertDistanceToScore(float32(scoreFloat))
					}
				}
			default:
				// 其他元数据
				if value, ok := fieldValue.(string); ok {
					doc.Metadata[fieldName] = value
				}
			}
		}
		
		// 从 key 中提取 ID
		if strings.HasPrefix(key, r.config.Prefix) {
			doc.Metadata["id"] = strings.TrimPrefix(key, r.config.Prefix)
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

func (r *RedisVectorStore) serializeVector(vector []float32) []byte {
	// 将 float32 数组转换为字节数组
	bytes := make([]byte, len(vector)*4)
	for i, v := range vector {
		bits := uint32(0)
		if v != 0 {
			// 简单的 float32 到 bytes 转换
			// 注意：实际应用中应该使用更可靠的方法，例如 encoding/binary
			bits = *(*uint32)(unsafe.Pointer(&v))
		}
		bytes[i*4] = byte(bits)
		bytes[i*4+1] = byte(bits >> 8)
		bytes[i*4+2] = byte(bits >> 16)
		bytes[i*4+3] = byte(bits >> 24)
	}
	return bytes
}

func (r *RedisVectorStore) serializeValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64, float32, float64, bool:
		return fmt.Sprintf("%v", val)
	default:
		// 对于复杂类型，使用 JSON
		bytes, _ := json.Marshal(v)
		return string(bytes)
	}
}

func (r *RedisVectorStore) convertDistanceToScore(distance float32) float32 {
	// 根据距离度量转换为相似度分数
	switch r.config.DistanceMetric {
	case "COSINE":
		// Cosine 距离范围 [0, 2]，0 表示完全相似
		// 转换为 [0, 1]，1 表示完全相似
		return 1.0 - (distance / 2.0)
	case "IP":
		// 内积，值越大越相似
		return distance
	case "L2":
		// L2 距离，值越小越相似
		// 使用指数衰减转换
		return 1.0 / (1.0 + distance)
	default:
		return 1.0 / (1.0 + distance)
	}
}

// Note: 为了简化，这里省略了 unsafe.Pointer 的导入
// 实际使用时需要添加: import "unsafe"
// 或者使用 encoding/binary 包进行更安全的字节转换

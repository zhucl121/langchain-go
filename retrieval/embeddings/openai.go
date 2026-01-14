package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIEmbeddings 是 OpenAI 嵌入模型。
//
// 支持的模型：
//   - text-embedding-ada-002 (1536 维)
//   - text-embedding-3-small (1536 维)
//   - text-embedding-3-large (3072 维)
//
type OpenAIEmbeddings struct {
	*BaseEmbeddings
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// OpenAIEmbeddingsConfig 是 OpenAI 嵌入配置。
type OpenAIEmbeddingsConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

// NewOpenAIEmbeddings 创建 OpenAI 嵌入模型。
//
// 参数：
//   - config: OpenAI 配置
//
// 返回：
//   - *OpenAIEmbeddings: OpenAI 嵌入模型实例
//
func NewOpenAIEmbeddings(config OpenAIEmbeddingsConfig) *OpenAIEmbeddings {
	if config.Model == "" {
		config.Model = "text-embedding-ada-002"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	
	dimension := 1536 // 默认维度
	if config.Model == "text-embedding-3-large" {
		dimension = 3072
	}
	
	return &OpenAIEmbeddings{
		BaseEmbeddings: NewBaseEmbeddings(config.Model, dimension),
		apiKey:         config.APIKey,
		baseURL:        config.BaseURL,
		model:          config.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// openAIEmbeddingRequest 是请求结构
type openAIEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// openAIEmbeddingResponse 是响应结构
type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbedDocuments 实现 Embeddings 接口。
func (oai *OpenAIEmbeddings) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	
	// 构建请求
	reqBody := openAIEmbeddingRequest{
		Input: texts,
		Model: oai.model,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: marshal request failed: %w", err)
	}
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", 
		oai.baseURL+"/embeddings", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: create request failed: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+oai.apiKey)
	
	resp, err := oai.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: read response failed: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embeddings: HTTP %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var result openAIEmbeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("openai embeddings: unmarshal response failed: %w", err)
	}
	
	// 提取向量
	embeddings := make([][]float32, len(result.Data))
	for _, item := range result.Data {
		embeddings[item.Index] = item.Embedding
	}
	
	return embeddings, nil
}

// EmbedQuery 实现 Embeddings 接口。
func (oai *OpenAIEmbeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := oai.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("openai embeddings: no embedding returned")
	}
	
	return embeddings[0], nil
}

// FakeEmbeddings 是用于测试的假嵌入模型。
//
// 生成随机或固定的向量，不调用实际的 API。
//
type FakeEmbeddings struct {
	*BaseEmbeddings
}

// NewFakeEmbeddings 创建假嵌入模型。
//
// 参数：
//   - dimension: 向量维度
//
// 返回：
//   - *FakeEmbeddings: 假嵌入模型实例
//
func NewFakeEmbeddings(dimension int) *FakeEmbeddings {
	return &FakeEmbeddings{
		BaseEmbeddings: NewBaseEmbeddings("fake", dimension),
	}
}

// EmbedDocuments 实现 Embeddings 接口。
func (fe *FakeEmbeddings) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	
	for i := range texts {
		// 生成简单的确定性向量（基于文本长度）
		vector := make([]float32, fe.dimension)
		for j := range vector {
			// 使用文本长度和索引生成值
			vector[j] = float32(len(texts[i])+j) / 100.0
		}
		embeddings[i] = vector
	}
	
	return embeddings, nil
}

// EmbedQuery 实现 Embeddings 接口。
func (fe *FakeEmbeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := fe.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

// CachedEmbeddings 是带缓存的嵌入模型包装器。
type CachedEmbeddings struct {
	underlying Embeddings
	cache      map[string][]float32
}

// NewCachedEmbeddings 创建缓存嵌入模型。
func NewCachedEmbeddings(underlying Embeddings) *CachedEmbeddings {
	return &CachedEmbeddings{
		underlying: underlying,
		cache:      make(map[string][]float32),
	}
}

// EmbedDocuments 实现 Embeddings 接口。
func (ce *CachedEmbeddings) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	var toEmbed []string
	var toEmbedIndices []int
	
	// 检查缓存
	for i, text := range texts {
		if cached, ok := ce.cache[text]; ok {
			embeddings[i] = cached
		} else {
			toEmbed = append(toEmbed, text)
			toEmbedIndices = append(toEmbedIndices, i)
		}
	}
	
	// 嵌入未缓存的文本
	if len(toEmbed) > 0 {
		newEmbeddings, err := ce.underlying.EmbedDocuments(ctx, toEmbed)
		if err != nil {
			return nil, err
		}
		
		// 更新缓存和结果
		for i, embedding := range newEmbeddings {
			idx := toEmbedIndices[i]
			embeddings[idx] = embedding
			ce.cache[toEmbed[i]] = embedding
		}
	}
	
	return embeddings, nil
}

// EmbedQuery 实现 Embeddings 接口。
func (ce *CachedEmbeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if cached, ok := ce.cache[text]; ok {
		return cached, nil
	}
	
	embedding, err := ce.underlying.EmbedQuery(ctx, text)
	if err != nil {
		return nil, err
	}
	
	ce.cache[text] = embedding
	return embedding, nil
}

// GetDimension 实现 Embeddings 接口。
func (ce *CachedEmbeddings) GetDimension() int {
	return ce.underlying.GetDimension()
}

// ClearCache 清空缓存。
func (ce *CachedEmbeddings) ClearCache() {
	ce.cache = make(map[string][]float32)
}

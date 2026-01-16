package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaEmbeddings 实现了 Ollama API 的 embedding 接口
type OllamaEmbeddings struct {
	*BaseEmbeddings
	baseURL   string
	model     string
	client    *http.Client
	dimension int // 缓存维度
}

// OllamaEmbeddingsConfig Ollama embeddings 配置
type OllamaEmbeddingsConfig struct {
	BaseURL   string // Ollama 服务地址,默认 http://localhost:11434
	Model     string // 模型名称,如 bge-m3, nomic-embed-text 等
	Dimension int    // 向量维度(可选,如果不设置会自动检测)
}

// ollamaEmbeddingRequest Ollama API 请求结构
type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbeddingResponse Ollama API 响应结构
type ollamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaEmbeddings 创建 Ollama embeddings 实例
func NewOllamaEmbeddings(config OllamaEmbeddingsConfig) *OllamaEmbeddings {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	
	model := config.Model
	if model == "" {
		model = "bge-m3"
	}
	
	dimension := config.Dimension
	if dimension == 0 {
		// 常见模型的默认维度
		switch model {
		case "bge-m3":
			dimension = 1024
		case "bge-large-zh":
			dimension = 1024
		case "bge-base-zh":
			dimension = 768
		case "nomic-embed-text":
			dimension = 768
		case "all-minilm":
			dimension = 384
		default:
			// 如果不知道维度,设为 0,后续会自动检测
			dimension = 0
		}
	}
	
	return &OllamaEmbeddings{
		BaseEmbeddings: NewBaseEmbeddings(model, dimension),
		baseURL:        baseURL,
		model:          model,
		client:         &http.Client{},
		dimension:      dimension,
	}
}

// EmbedDocuments 批量生成文档 embeddings
func (oe *OllamaEmbeddings) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	
	for i, text := range texts {
		embedding, err := oe.embedSingle(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed document %d failed: %w", i, err)
		}
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}

// EmbedQuery 生成查询 embedding
func (oe *OllamaEmbeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return oe.embedSingle(ctx, text)
}

// embedSingle 生成单个文本的 embedding
func (oe *OllamaEmbeddings) embedSingle(ctx context.Context, text string) ([]float32, error) {
	// 构建请求
	reqBody := ollamaEmbeddingRequest{
		Model:  oe.model,
		Prompt: text,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}
	
	// 发送请求
	url := fmt.Sprintf("%s/api/embeddings", oe.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := oe.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var response ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	
	return response.Embedding, nil
}

// GetDimension 返回 embedding 维度 (实现 Embeddings 接口)
func (oe *OllamaEmbeddings) GetDimension() int {
	// 如果已经知道维度,直接返回
	if oe.dimension > 0 {
		return oe.dimension
	}
	
	// 否则通过生成测试 embedding 来获取维度
	ctx := context.Background()
	embedding, err := oe.embedSingle(ctx, "test")
	if err != nil {
		// 如果失败,返回 BGE-M3 的默认维度
		return 1024
	}
	
	// 缓存维度
	oe.dimension = len(embedding)
	oe.BaseEmbeddings.dimension = oe.dimension
	
	return oe.dimension
}

// GetModelName 返回模型名称
func (oe *OllamaEmbeddings) GetModelName() string {
	return oe.model
}

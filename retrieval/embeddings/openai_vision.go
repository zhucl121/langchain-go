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

// OpenAIVisionConfig OpenAI Vision 配置
type OpenAIVisionConfig struct {
	// APIKey OpenAI API 密钥
	APIKey string
	
	// BaseURL API 基础 URL
	BaseURL string
	
	// Model 模型名称 (默认: clip-vit-base-patch32)
	Model string
	
	// Timeout 请求超时时间
	Timeout time.Duration
	
	// MaxRetries 最大重试次数
	MaxRetries int
}

// DefaultOpenAIVisionConfig 返回默认配置
func DefaultOpenAIVisionConfig(apiKey string) OpenAIVisionConfig {
	return OpenAIVisionConfig{
		APIKey:     apiKey,
		BaseURL:    "https://api.openai.com/v1",
		Model:      "clip-vit-base-patch32",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// OpenAIVisionEmbedder OpenAI Vision 图像嵌入器
//
// 使用 OpenAI 的 CLIP 模型对图像进行向量化。
type OpenAIVisionEmbedder struct {
	config OpenAIVisionConfig
	client *http.Client
}

// NewOpenAIVisionEmbedder 创建 OpenAI Vision 嵌入器
func NewOpenAIVisionEmbedder(config OpenAIVisionConfig) *OpenAIVisionEmbedder {
	return &OpenAIVisionEmbedder{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (e *OpenAIVisionEmbedder) GetDimension() int {
	// CLIP-ViT-B/32 的输出维度
	return 512
}

func (e *OpenAIVisionEmbedder) GetName() string {
	return "openai-vision-" + e.config.Model
}

// EmbedImage 对图像进行向量化
func (e *OpenAIVisionEmbedder) EmbedImage(ctx context.Context, imageData []byte) ([]float64, error) {
	// 注意: 这是一个简化实现
	// 实际的 OpenAI API 可能需要不同的调用方式
	
	// 构造请求
	req := map[string]interface{}{
		"model": e.config.Model,
		"input": map[string]interface{}{
			"image": imageData,
		},
	}
	
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 发送请求
	url := e.config.BaseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed: status=%d, body=%s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	
	return result.Data[0].Embedding, nil
}

// EmbedImageBatch 批量对图像进行向量化
func (e *OpenAIVisionEmbedder) EmbedImageBatch(ctx context.Context, images [][]byte) ([][]float64, error) {
	embeddings := make([][]float64, len(images))
	
	// 逐个处理（OpenAI API 限制）
	// 生产环境应该实现批处理和并发控制
	for i, image := range images {
		embedding, err := e.EmbedImage(ctx, image)
		if err != nil {
			return nil, fmt.Errorf("failed to embed image %d: %w", i, err)
		}
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}

// MockImageEmbedder Mock 图像嵌入器（用于测试）
type MockImageEmbedder struct {
	dimension int
}

// NewMockImageEmbedder 创建 Mock 图像嵌入器
func NewMockImageEmbedder(dimension int) *MockImageEmbedder {
	return &MockImageEmbedder{dimension: dimension}
}

func (e *MockImageEmbedder) GetDimension() int {
	return e.dimension
}

func (e *MockImageEmbedder) GetName() string {
	return "mock-image-embedder"
}

func (e *MockImageEmbedder) EmbedImage(ctx context.Context, imageData []byte) ([]float64, error) {
	// 生成确定性的假向量（基于数据大小）
	embedding := make([]float64, e.dimension)
	seed := float64(len(imageData))
	
	for i := 0; i < e.dimension; i++ {
		embedding[i] = (seed + float64(i)) / float64(e.dimension)
	}
	
	// 归一化
	norm := 0.0
	for _, v := range embedding {
		norm += v * v
	}
	norm = 1.0 / (norm + 1e-8)
	
	for i := range embedding {
		embedding[i] *= norm
	}
	
	return embedding, nil
}

func (e *MockImageEmbedder) EmbedImageBatch(ctx context.Context, images [][]byte) ([][]float64, error) {
	embeddings := make([][]float64, len(images))
	for i, image := range images {
		embedding, err := e.EmbedImage(ctx, image)
		if err != nil {
			return nil, err
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

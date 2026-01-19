package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// AzureOpenAIClient 实现 Azure OpenAI Service 集成
//
// Azure OpenAI Service 是 Microsoft Azure 上的 OpenAI 模型托管服务。
//
// 支持的模型:
//   - GPT-3.5-Turbo
//   - GPT-4
//   - GPT-4-32k
//   - GPT-4 Turbo
//   - GPT-4 Vision
//
// 使用示例:
//
//	config := azure.Config{
//	    Endpoint:     "https://your-resource.openai.azure.com",
//	    APIKey:       "your-api-key",
//	    Deployment:   "gpt-35-turbo",
//	    APIVersion:   "2024-02-01",
//	}
//	client := azure.New(config)
//
type AzureOpenAIClient struct {
	config     Config
	httpClient *http.Client
}

// Config 是 Azure OpenAI 的配置
type Config struct {
	// Endpoint Azure OpenAI 资源端点
	// 格式: https://<your-resource-name>.openai.azure.com
	Endpoint string
	
	// APIKey API 密钥
	APIKey string
	
	// Deployment 部署名称（模型部署的名称）
	Deployment string
	
	// APIVersion API 版本
	// 推荐: "2024-02-01", "2023-12-01-preview"
	APIVersion string
	
	// Temperature 温度参数 (0.0-2.0)
	Temperature float32
	
	// TopP 核采样参数
	TopP float32
	
	// MaxTokens 最大输出 token 数
	MaxTokens int
	
	// PresencePenalty 存在惩罚 (-2.0 到 2.0)
	PresencePenalty float32
	
	// FrequencyPenalty 频率惩罚 (-2.0 到 2.0)
	FrequencyPenalty float32
	
	// Stop 停止序列
	Stop []string
	
	// HTTPClient 自定义 HTTP 客户端
	HTTPClient *http.Client
	
	// Timeout 请求超时时间
	Timeout time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		APIVersion:  "2024-02-01",
		Temperature: 0.7,
		TopP:        1.0,
		MaxTokens:   2048,
		Timeout:     60 * time.Second,
	}
}

// New 创建新的 Azure OpenAI 客户端
func New(config Config) (*AzureOpenAIClient, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("azure: endpoint is required")
	}
	
	if config.APIKey == "" {
		return nil, fmt.Errorf("azure: API key is required")
	}
	
	if config.Deployment == "" {
		return nil, fmt.Errorf("azure: deployment name is required")
	}
	
	if config.APIVersion == "" {
		config.APIVersion = "2024-02-01"
	}
	
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	
	// 确保 endpoint 不以斜杠结尾
	config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")
	
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}
	
	return &AzureOpenAIClient{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// Invoke 调用 Azure OpenAI API 生成响应
func (c *AzureOpenAIClient) Invoke(ctx context.Context, messages []types.Message, opts ...Option) (types.Message, error) {
	// 应用选项
	config := c.config
	for _, opt := range opts {
		opt(&config)
	}
	
	// 转换消息格式
	azureMessages, err := c.convertMessages(messages)
	if err != nil {
		return types.Message{}, fmt.Errorf("azure: failed to convert messages: %w", err)
	}
	
	// 构建请求
	reqBody := AzureRequest{
		Messages:         azureMessages,
		Temperature:      config.Temperature,
		TopP:             config.TopP,
		MaxTokens:        config.MaxTokens,
		PresencePenalty:  config.PresencePenalty,
		FrequencyPenalty: config.FrequencyPenalty,
		Stop:             config.Stop,
	}
	
	// 发送请求
	response, err := c.chatCompletion(ctx, &config, &reqBody)
	if err != nil {
		return types.Message{}, err
	}
	
	// 解析响应
	return c.parseResponse(response)
}

// Stream 流式生成响应
func (c *AzureOpenAIClient) Stream(ctx context.Context, messages []types.Message, opts ...Option) (<-chan types.StreamEvent, error) {
	// 应用选项
	config := c.config
	for _, opt := range opts {
		opt(&config)
	}
	
	// 转换消息格式
	azureMessages, err := c.convertMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("azure: failed to convert messages: %w", err)
	}
	
	// 构建请求
	reqBody := AzureRequest{
		Messages:         azureMessages,
		Temperature:      config.Temperature,
		TopP:             config.TopP,
		MaxTokens:        config.MaxTokens,
		PresencePenalty:  config.PresencePenalty,
		FrequencyPenalty: config.FrequencyPenalty,
		Stop:             config.Stop,
		Stream:           true,
	}
	
	// 创建流式响应通道
	eventChan := make(chan types.StreamEvent, 10)
	
	go func() {
		defer close(eventChan)
		
		if err := c.chatCompletionStream(ctx, &config, &reqBody, eventChan); err != nil {
			eventChan <- types.StreamEvent{
				Error: err,
			}
		}
	}()
	
	return eventChan, nil
}

// Batch 批量处理消息
func (c *AzureOpenAIClient) Batch(ctx context.Context, messagesList [][]types.Message, opts ...Option) ([]types.Message, error) {
	results := make([]types.Message, len(messagesList))
	
	for i, messages := range messagesList {
		result, err := c.Invoke(ctx, messages, opts...)
		if err != nil {
			return nil, fmt.Errorf("azure: batch request %d failed: %w", i, err)
		}
		results[i] = result
	}
	
	return results, nil
}

// ==================== 内部方法 ====================

func (c *AzureOpenAIClient) convertMessages(messages []types.Message) ([]AzureMessage, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("azure: messages are required")
	}
	
	azureMessages := make([]AzureMessage, 0, len(messages))
	
	for _, msg := range messages {
		azureMessages = append(azureMessages, AzureMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}
	
	return azureMessages, nil
}

func (c *AzureOpenAIClient) chatCompletion(ctx context.Context, config *Config, reqBody *AzureRequest) (*AzureResponse, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("azure: failed to marshal request: %w", err)
	}
	
	// 构建 URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		config.Endpoint, config.Deployment, config.APIVersion)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("azure: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", config.APIKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("azure: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("azure: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	var response AzureResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("azure: failed to decode response: %w", err)
	}
	
	return &response, nil
}

func (c *AzureOpenAIClient) chatCompletionStream(ctx context.Context, config *Config, reqBody *AzureRequest, eventChan chan<- types.StreamEvent) error {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("azure: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		config.Endpoint, config.Deployment, config.APIVersion)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("azure: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", config.APIKey)
	req.Header.Set("Accept", "text/event-stream")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("azure: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("azure: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// 解析 SSE 流
	return c.parseSSEStream(resp.Body, eventChan)
}

func (c *AzureOpenAIClient) parseSSEStream(reader io.Reader, eventChan chan<- types.StreamEvent) error {
	buf := make([]byte, 4096)
	leftover := ""
	
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("azure: failed to read stream: %w", err)
		}
		
		if n == 0 {
			break
		}
		
		data := leftover + string(buf[:n])
		lines := strings.Split(data, "\n")
		
		// 保留最后一个不完整的行
		if !strings.HasSuffix(data, "\n") {
			leftover = lines[len(lines)-1]
			lines = lines[:len(lines)-1]
		} else {
			leftover = ""
		}
		
		// 处理每一行
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}
			
			jsonData := strings.TrimPrefix(line, "data: ")
			if jsonData == "[DONE]" {
				break
			}
			
			var chunk AzureStreamChunk
			if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
				continue
			}
			
			// 提取文本
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				eventChan <- types.StreamEvent{
					Data: types.Message{
						Role:    types.RoleAssistant,
						Content: chunk.Choices[0].Delta.Content,
					},
				}
			}
		}
		
		if err == io.EOF {
			break
		}
	}
	
	return nil
}

func (c *AzureOpenAIClient) parseResponse(response *AzureResponse) (types.Message, error) {
	if len(response.Choices) == 0 {
		return types.Message{}, fmt.Errorf("azure: no choices in response")
	}
	
	choice := response.Choices[0]
	
	return types.Message{
		Role:    types.RoleAssistant,
		Content: choice.Message.Content,
	}, nil
}

// ==================== 选项模式 ====================

// Option 配置选项
type Option func(*Config)

// WithTemperature 设置温度
func WithTemperature(temp float32) Option {
	return func(c *Config) {
		c.Temperature = temp
	}
}

// WithTopP 设置 TopP
func WithTopP(topP float32) Option {
	return func(c *Config) {
		c.TopP = topP
	}
}

// WithMaxTokens 设置最大 token 数
func WithMaxTokens(maxTokens int) Option {
	return func(c *Config) {
		c.MaxTokens = maxTokens
	}
}

// WithPresencePenalty 设置存在惩罚
func WithPresencePenalty(penalty float32) Option {
	return func(c *Config) {
		c.PresencePenalty = penalty
	}
}

// WithFrequencyPenalty 设置频率惩罚
func WithFrequencyPenalty(penalty float32) Option {
	return func(c *Config) {
		c.FrequencyPenalty = penalty
	}
}

// WithStop 设置停止序列
func WithStop(stop []string) Option {
	return func(c *Config) {
		c.Stop = stop
	}
}

// ==================== API 类型定义 ====================

// AzureRequest Azure OpenAI API 请求
type AzureRequest struct {
	Messages         []AzureMessage `json:"messages"`
	Temperature      float32        `json:"temperature,omitempty"`
	TopP             float32        `json:"top_p,omitempty"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	PresencePenalty  float32        `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32        `json:"frequency_penalty,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
}

// AzureMessage Azure 消息格式
type AzureMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AzureResponse Azure OpenAI API 响应
type AzureResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []AzureChoice `json:"choices"`
	Usage   AzureUsage    `json:"usage"`
}

// AzureChoice 选择项
type AzureChoice struct {
	Index        int          `json:"index"`
	Message      AzureMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// AzureUsage token 使用情况
type AzureUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AzureStreamChunk 流式响应块
type AzureStreamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []AzureStreamChoice `json:"choices"`
}

// AzureStreamChoice 流式选择项
type AzureStreamChoice struct {
	Index        int               `json:"index"`
	Delta        AzureMessageDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

// AzureMessageDelta 流式消息增量
type AzureMessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

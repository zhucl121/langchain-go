package gemini

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

// GeminiClient 实现 Google Gemini API 集成
//
// Google Gemini 是 Google 的多模态大语言模型系列。
//
// 支持的模型:
//   - gemini-pro: 文本生成
//   - gemini-pro-vision: 多模态（文本+图像）
//   - gemini-1.5-pro: 最新版本，更长上下文
//   - gemini-1.5-flash: 快速响应版本
//
// 使用示例:
//
//	config := gemini.Config{
//	    APIKey: "your-api-key",
//	    Model:  "gemini-pro",
//	}
//	client := gemini.New(config)
//
type GeminiClient struct {
	config     Config
	httpClient *http.Client
}

// Config 是 Gemini 的配置
type Config struct {
	// APIKey Google API 密钥
	APIKey string
	
	// Model 模型名称
	// 支持: "gemini-pro", "gemini-pro-vision", "gemini-1.5-pro", "gemini-1.5-flash"
	Model string
	
	// Temperature 温度参数 (0.0-2.0)
	Temperature float32
	
	// TopP 核采样参数
	TopP float32
	
	// TopK Top-K 采样参数
	TopK int
	
	// MaxTokens 最大输出 token 数
	MaxTokens int
	
	// StopSequences 停止序列
	StopSequences []string
	
	// SafetySettings 安全设置
	SafetySettings []SafetySetting
	
	// HTTPClient 自定义 HTTP 客户端
	HTTPClient *http.Client
	
	// Timeout 请求超时时间
	Timeout time.Duration
	
	// BaseURL API 基础 URL（用于自定义端点）
	BaseURL string
}

// SafetySetting 安全设置
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Model:       "gemini-pro",
		Temperature: 0.7,
		TopP:        0.95,
		TopK:        40,
		MaxTokens:   2048,
		Timeout:     60 * time.Second,
		BaseURL:     "https://generativelanguage.googleapis.com/v1beta",
	}
}

// New 创建新的 Gemini 客户端
func New(config Config) (*GeminiClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("gemini: API key is required")
	}
	
	if config.Model == "" {
		config.Model = "gemini-pro"
	}
	
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}
	
	return &GeminiClient{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// Invoke 调用 Gemini API 生成响应
func (c *GeminiClient) Invoke(ctx context.Context, messages []types.Message, opts ...Option) (types.Message, error) {
	// 应用选项
	config := c.config
	for _, opt := range opts {
		opt(&config)
	}
	
	// 转换消息格式
	contents, err := c.convertMessages(messages)
	if err != nil {
		return types.Message{}, fmt.Errorf("gemini: failed to convert messages: %w", err)
	}
	
	// 构建请求
	reqBody := GeminiRequest{
		Contents: contents,
		GenerationConfig: GenerationConfig{
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			TopK:           config.TopK,
			MaxOutputTokens: config.MaxTokens,
			StopSequences:  config.StopSequences,
		},
	}
	
	// 添加安全设置
	if len(config.SafetySettings) > 0 {
		reqBody.SafetySettings = config.SafetySettings
	}
	
	// 发送请求
	response, err := c.generateContent(ctx, &config, &reqBody)
	if err != nil {
		return types.Message{}, err
	}
	
	// 解析响应
	return c.parseResponse(response)
}

// Stream 流式生成响应
func (c *GeminiClient) Stream(ctx context.Context, messages []types.Message, opts ...Option) (<-chan types.StreamEvent, error) {
	// 应用选项
	config := c.config
	for _, opt := range opts {
		opt(&config)
	}
	
	// 转换消息格式
	contents, err := c.convertMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to convert messages: %w", err)
	}
	
	// 构建请求
	reqBody := GeminiRequest{
		Contents: contents,
		GenerationConfig: GenerationConfig{
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			TopK:           config.TopK,
			MaxOutputTokens: config.MaxTokens,
			StopSequences:  config.StopSequences,
		},
	}
	
	if len(config.SafetySettings) > 0 {
		reqBody.SafetySettings = config.SafetySettings
	}
	
	// 创建流式响应通道
	eventChan := make(chan types.StreamEvent, 10)
	
	go func() {
		defer close(eventChan)
		
		if err := c.streamContent(ctx, &config, &reqBody, eventChan); err != nil {
			eventChan <- types.StreamEvent{
				Error: err,
			}
		}
	}()
	
	return eventChan, nil
}

// Batch 批量处理消息
func (c *GeminiClient) Batch(ctx context.Context, messagesList [][]types.Message, opts ...Option) ([]types.Message, error) {
	results := make([]types.Message, len(messagesList))
	
	for i, messages := range messagesList {
		result, err := c.Invoke(ctx, messages, opts...)
		if err != nil {
			return nil, fmt.Errorf("gemini: batch request %d failed: %w", i, err)
		}
		results[i] = result
	}
	
	return results, nil
}

// ==================== 内部方法 ====================

func (c *GeminiClient) convertMessages(messages []types.Message) ([]Content, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("gemini: messages are required")
	}
	
	contents := make([]Content, 0, len(messages))
	
	for _, msg := range messages {
		// Gemini 使用 "user" 和 "model" 角色
		role := "user"
		if msg.Role == types.RoleAssistant {
			role = "model"
		} else if msg.Role == types.RoleSystem {
			// 系统消息作为用户消息的一部分
			role = "user"
		}
		
		parts := []Part{
			{Text: msg.Content},
		}
		
		contents = append(contents, Content{
			Role:  role,
			Parts: parts,
		})
	}
	
	return contents, nil
}

func (c *GeminiClient) generateContent(ctx context.Context, config *Config, reqBody *GeminiRequest) (*GeminiResponse, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		config.BaseURL, config.Model, config.APIKey)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	var response GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("gemini: failed to decode response: %w", err)
	}
	
	return &response, nil
}

func (c *GeminiClient) streamContent(ctx context.Context, config *Config, reqBody *GeminiRequest, eventChan chan<- types.StreamEvent) error {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("gemini: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse",
		config.BaseURL, config.Model, config.APIKey)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("gemini: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// 解析 SSE 流
	return c.parseSSEStream(resp.Body, eventChan)
}

func (c *GeminiClient) parseSSEStream(reader io.Reader, eventChan chan<- types.StreamEvent) error {
	buf := make([]byte, 4096)
	leftover := ""
	
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("gemini: failed to read stream: %w", err)
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
			
			var chunk GeminiResponse
			if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
				continue
			}
			
			// 提取文本
			if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
				text := chunk.Candidates[0].Content.Parts[0].Text
				eventChan <- types.StreamEvent{
					Data: types.Message{
						Role:    types.RoleAssistant,
						Content: text,
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

func (c *GeminiClient) parseResponse(response *GeminiResponse) (types.Message, error) {
	if len(response.Candidates) == 0 {
		return types.Message{}, fmt.Errorf("gemini: no candidates in response")
	}
	
	candidate := response.Candidates[0]
	
	// 检查安全评级
	if candidate.FinishReason == "SAFETY" {
		return types.Message{}, fmt.Errorf("gemini: content blocked due to safety settings")
	}
	
	// 提取文本
	var content strings.Builder
	for _, part := range candidate.Content.Parts {
		content.WriteString(part.Text)
	}
	
	return types.Message{
		Role:    types.RoleAssistant,
		Content: content.String(),
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

// WithTopK 设置 TopK
func WithTopK(topK int) Option {
	return func(c *Config) {
		c.TopK = topK
	}
}

// WithMaxTokens 设置最大 token 数
func WithMaxTokens(maxTokens int) Option {
	return func(c *Config) {
		c.MaxTokens = maxTokens
	}
}

// WithStopSequences 设置停止序列
func WithStopSequences(sequences []string) Option {
	return func(c *Config) {
		c.StopSequences = sequences
	}
}

// WithSafetySettings 设置安全级别
func WithSafetySettings(settings []SafetySetting) Option {
	return func(c *Config) {
		c.SafetySettings = settings
	}
}

// ==================== API 类型定义 ====================

// GeminiRequest Gemini API 请求
type GeminiRequest struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []SafetySetting  `json:"safetySettings,omitempty"`
}

// Content 内容
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// Part 内容部分
type Part struct {
	Text string `json:"text,omitempty"`
}

// GenerationConfig 生成配置
type GenerationConfig struct {
	Temperature     float32  `json:"temperature,omitempty"`
	TopP            float32  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// GeminiResponse Gemini API 响应
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate 候选响应
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
	Index        int     `json:"index"`
}

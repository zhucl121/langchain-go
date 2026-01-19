package bedrock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// BedrockClient 实现 AWS Bedrock API 集成
//
// AWS Bedrock 提供托管的基础模型服务，支持多个提供商的模型。
//
// 支持的模型系列:
//   - anthropic.claude-v2, anthropic.claude-3-*
//   - amazon.titan-*
//   - ai21.j2-*
//   - cohere.command-*
//   - meta.llama2-*
//
// 使用示例:
//
//	config := bedrock.Config{
//	    Region:    "us-east-1",
//	    AccessKey: "your-access-key",
//	    SecretKey: "your-secret-key",
//	    Model:     "anthropic.claude-v2",
//	}
//	client := bedrock.New(config)
//
type BedrockClient struct {
	config     Config
	httpClient *http.Client
}

// Config 是 AWS Bedrock 的配置
type Config struct {
	// Region AWS 区域
	Region string
	
	// AccessKey AWS 访问密钥 ID
	AccessKey string
	
	// SecretKey AWS 秘密访问密钥
	SecretKey string
	
	// SessionToken 会话令牌（如果使用临时凭证）
	SessionToken string
	
	// Model 模型 ID
	Model string
	
	// Temperature 温度参数
	Temperature float32
	
	// TopP 核采样参数
	TopP float32
	
	// MaxTokens 最大输出 token 数
	MaxTokens int
	
	// StopSequences 停止序列
	StopSequences []string
	
	// HTTPClient 自定义 HTTP 客户端
	HTTPClient *http.Client
	
	// Timeout 请求超时时间
	Timeout time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Region:      "us-east-1",
		Model:       "anthropic.claude-v2",
		Temperature: 0.7,
		TopP:        0.9,
		MaxTokens:   2048,
		Timeout:     60 * time.Second,
	}
}

// New 创建新的 Bedrock 客户端
func New(config Config) (*BedrockClient, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("bedrock: region is required")
	}
	
	if config.AccessKey == "" {
		return nil, fmt.Errorf("bedrock: access key is required")
	}
	
	if config.SecretKey == "" {
		return nil, fmt.Errorf("bedrock: secret key is required")
	}
	
	if config.Model == "" {
		config.Model = "anthropic.claude-v2"
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
	
	return &BedrockClient{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// Invoke 调用 Bedrock API 生成响应
func (c *BedrockClient) Invoke(ctx context.Context, messages []types.Message, opts ...Option) (types.Message, error) {
	// 应用选项
	config := c.config
	for _, opt := range opts {
		opt(&config)
	}
	
	// 根据模型类型构建不同的请求
	reqBody, err := c.buildRequest(&config, messages)
	if err != nil {
		return types.Message{}, fmt.Errorf("bedrock: failed to build request: %w", err)
	}
	
	// 发送请求
	response, err := c.invokeModel(ctx, &config, reqBody)
	if err != nil {
		return types.Message{}, err
	}
	
	// 解析响应
	return c.parseResponse(&config, response)
}

// Stream 流式生成响应
func (c *BedrockClient) Stream(ctx context.Context, messages []types.Message, opts ...Option) (<-chan types.StreamEvent, error) {
	// 应用选项
	config := c.config
	for _, opt := range opts {
		opt(&config)
	}
	
	// 构建请求
	reqBody, err := c.buildRequest(&config, messages)
	if err != nil {
		return nil, fmt.Errorf("bedrock: failed to build request: %w", err)
	}
	
	// 创建流式响应通道
	eventChan := make(chan types.StreamEvent, 10)
	
	go func() {
		defer close(eventChan)
		
		if err := c.invokeModelStream(ctx, &config, reqBody, eventChan); err != nil {
			eventChan <- types.StreamEvent{
				Error: err,
			}
		}
	}()
	
	return eventChan, nil
}

// Batch 批量处理消息
func (c *BedrockClient) Batch(ctx context.Context, messagesList [][]types.Message, opts ...Option) ([]types.Message, error) {
	results := make([]types.Message, len(messagesList))
	
	for i, messages := range messagesList {
		result, err := c.Invoke(ctx, messages, opts...)
		if err != nil {
			return nil, fmt.Errorf("bedrock: batch request %d failed: %w", i, err)
		}
		results[i] = result
	}
	
	return results, nil
}

// ==================== 内部方法 ====================

func (c *BedrockClient) buildRequest(config *Config, messages []types.Message) (map[string]interface{}, error) {
	// 根据模型提供商构建不同格式的请求
	if isAnthropicModel(config.Model) {
		return c.buildAnthropicRequest(config, messages)
	} else if isTitanModel(config.Model) {
		return c.buildTitanRequest(config, messages)
	} else if isLlamaModel(config.Model) {
		return c.buildLlamaRequest(config, messages)
	}
	
	// 默认使用 Anthropic 格式
	return c.buildAnthropicRequest(config, messages)
}

func (c *BedrockClient) buildAnthropicRequest(config *Config, messages []types.Message) (map[string]interface{}, error) {
	// 转换消息格式
	anthropicMessages := make([]map[string]interface{}, 0, len(messages))
	var systemPrompt string
	
	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			systemPrompt = msg.Content
			continue
		}
		
		role := "user"
		if msg.Role == types.RoleAssistant {
			role = "assistant"
		}
		
		anthropicMessages = append(anthropicMessages, map[string]interface{}{
			"role":    role,
			"content": msg.Content,
		})
	}
	
	reqBody := map[string]interface{}{
		"messages":    anthropicMessages,
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
		"top_p":       config.TopP,
	}
	
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}
	
	if len(config.StopSequences) > 0 {
		reqBody["stop_sequences"] = config.StopSequences
	}
	
	// Anthropic 格式需要 anthropic_version
	reqBody["anthropic_version"] = "bedrock-2023-05-31"
	
	return reqBody, nil
}

func (c *BedrockClient) buildTitanRequest(config *Config, messages []types.Message) (map[string]interface{}, error) {
	// 合并所有消息为单个输入文本
	var promptBuilder bytes.Buffer
	for _, msg := range messages {
		promptBuilder.WriteString(msg.Content)
		promptBuilder.WriteString("\n")
	}
	
	return map[string]interface{}{
		"inputText": promptBuilder.String(),
		"textGenerationConfig": map[string]interface{}{
			"temperature":    config.Temperature,
			"topP":           config.TopP,
			"maxTokenCount":  config.MaxTokens,
			"stopSequences":  config.StopSequences,
		},
	}, nil
}

func (c *BedrockClient) buildLlamaRequest(config *Config, messages []types.Message) (map[string]interface{}, error) {
	// Llama 使用简单的提示格式
	var promptBuilder bytes.Buffer
	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			promptBuilder.WriteString("[INST] <<SYS>>\n")
			promptBuilder.WriteString(msg.Content)
			promptBuilder.WriteString("\n<</SYS>>\n\n")
		} else if msg.Role == types.RoleUser {
			promptBuilder.WriteString("[INST] ")
			promptBuilder.WriteString(msg.Content)
			promptBuilder.WriteString(" [/INST]")
		} else if msg.Role == types.RoleAssistant {
			promptBuilder.WriteString(" ")
			promptBuilder.WriteString(msg.Content)
			promptBuilder.WriteString(" ")
		}
	}
	
	return map[string]interface{}{
		"prompt":      promptBuilder.String(),
		"temperature": config.Temperature,
		"top_p":       config.TopP,
		"max_gen_len": config.MaxTokens,
	}, nil
}

func (c *BedrockClient) invokeModel(ctx context.Context, config *Config, reqBody map[string]interface{}) (map[string]interface{}, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("bedrock: failed to marshal request: %w", err)
	}
	
	// 构建 Bedrock Runtime API 端点
	url := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke",
		config.Region, config.Model)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("bedrock: failed to create request: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// 签名请求（AWS Signature V4）
	if err := c.signRequest(req, bodyBytes, config); err != nil {
		return nil, fmt.Errorf("bedrock: failed to sign request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bedrock: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bedrock: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("bedrock: failed to decode response: %w", err)
	}
	
	return response, nil
}

func (c *BedrockClient) invokeModelStream(ctx context.Context, config *Config, reqBody map[string]interface{}, eventChan chan<- types.StreamEvent) error {
	// Bedrock 流式 API 实现
	// 注意：流式 API 需要特殊的事件流处理
	// 这里提供基础实现框架
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("bedrock: failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke-with-response-stream",
		config.Region, config.Model)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("bedrock: failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.amazon.eventstream")
	
	// 签名请求
	if err := c.signRequest(req, bodyBytes, config); err != nil {
		return fmt.Errorf("bedrock: failed to sign request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bedrock: request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bedrock: API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// 解析事件流
	// AWS Event Stream 格式比较复杂，这里提供简化版本
	// 实际使用时建议使用 AWS SDK
	return c.parseEventStream(resp.Body, eventChan)
}

func (c *BedrockClient) parseEventStream(reader io.Reader, eventChan chan<- types.StreamEvent) error {
	// 简化的事件流解析
	// 实际实现需要处理 AWS Event Stream 二进制格式
	decoder := json.NewDecoder(reader)
	
	for {
		var event map[string]interface{}
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		
		// 提取文本内容
		if completion, ok := event["completion"].(string); ok {
			eventChan <- types.StreamEvent{
				Data: types.Message{
					Role:    types.RoleAssistant,
					Content: completion,
				},
			}
		}
	}
	
	return nil
}

func (c *BedrockClient) parseResponse(config *Config, response map[string]interface{}) (types.Message, error) {
	// 根据模型类型解析不同格式的响应
	if isAnthropicModel(config.Model) {
		return c.parseAnthropicResponse(response)
	} else if isTitanModel(config.Model) {
		return c.parseTitanResponse(response)
	} else if isLlamaModel(config.Model) {
		return c.parseLlamaResponse(response)
	}
	
	return c.parseAnthropicResponse(response)
}

func (c *BedrockClient) parseAnthropicResponse(response map[string]interface{}) (types.Message, error) {
	// Anthropic 响应格式
	content, ok := response["content"].([]interface{})
	if !ok || len(content) == 0 {
		return types.Message{}, fmt.Errorf("bedrock: invalid response format")
	}
	
	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return types.Message{}, fmt.Errorf("bedrock: invalid content format")
	}
	
	text, ok := firstContent["text"].(string)
	if !ok {
		return types.Message{}, fmt.Errorf("bedrock: no text in response")
	}
	
	return types.Message{
		Role:    types.RoleAssistant,
		Content: text,
	}, nil
}

func (c *BedrockClient) parseTitanResponse(response map[string]interface{}) (types.Message, error) {
	results, ok := response["results"].([]interface{})
	if !ok || len(results) == 0 {
		return types.Message{}, fmt.Errorf("bedrock: invalid Titan response")
	}
	
	firstResult, ok := results[0].(map[string]interface{})
	if !ok {
		return types.Message{}, fmt.Errorf("bedrock: invalid Titan result")
	}
	
	text, ok := firstResult["outputText"].(string)
	if !ok {
		return types.Message{}, fmt.Errorf("bedrock: no output text in Titan response")
	}
	
	return types.Message{
		Role:    types.RoleAssistant,
		Content: text,
	}, nil
}

func (c *BedrockClient) parseLlamaResponse(response map[string]interface{}) (types.Message, error) {
	generation, ok := response["generation"].(string)
	if !ok {
		return types.Message{}, fmt.Errorf("bedrock: invalid Llama response")
	}
	
	return types.Message{
		Role:    types.RoleAssistant,
		Content: generation,
	}, nil
}

func (c *BedrockClient) signRequest(req *http.Request, body []byte, config *Config) error {
	// AWS Signature V4 签名
	// 注意：这是简化实现，实际使用建议使用 AWS SDK
	// 完整的签名算法参考：https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html
	
	// 这里提供基础框架，实际实现需要完整的签名逻辑
	// 或者使用 github.com/aws/aws-sdk-go-v2
	
	return fmt.Errorf("bedrock: AWS Signature V4 implementation required - please use AWS SDK")
}

// ==================== 辅助函数 ====================

func isAnthropicModel(model string) bool {
	return len(model) >= 10 && model[:10] == "anthropic."
}

func isTitanModel(model string) bool {
	return len(model) >= 7 && model[:7] == "amazon."
}

func isLlamaModel(model string) bool {
	return len(model) >= 5 && model[:5] == "meta."
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

// WithStopSequences 设置停止序列
func WithStopSequences(sequences []string) Option {
	return func(c *Config) {
		c.StopSequences = sequences
	}
}

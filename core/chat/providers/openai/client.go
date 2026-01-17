package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
)

const (
	// DefaultBaseURL 是 OpenAI API 的默认地址
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultModel 是默认使用的模型
	DefaultModel = "gpt-4o-mini"

	// DefaultTimeout 是默认的请求超时时间
	DefaultTimeout = 60 * time.Second

	// DefaultMaxTokens 是默认的最大生成 token 数
	DefaultMaxTokens = 4096
)

// Config 是 OpenAI ChatModel 的配置。
type Config struct {
	// APIKey 是 OpenAI API 密钥（必需）
	APIKey string

	// BaseURL 是 API 基础地址（可选，默认为 OpenAI 官方地址）
	// 可用于配置代理或使用兼容 OpenAI API 的服务
	BaseURL string

	// Model 是模型名称（可选，默认为 gpt-4o-mini）
	// 支持的模型：gpt-3.5-turbo, gpt-4, gpt-4-turbo, gpt-4o 等
	Model string

	// Temperature 控制输出的随机性（可选，0.0-2.0，默认 0.7）
	// 较低的值使输出更确定，较高的值使输出更随机
	Temperature float64

	// MaxTokens 是最大生成 token 数（可选）
	MaxTokens int

	// TopP 是核采样参数（可选，0.0-1.0）
	TopP float64

	// FrequencyPenalty 是频率惩罚（可选，-2.0-2.0）
	FrequencyPenalty float64

	// PresencePenalty 是存在惩罚（可选，-2.0-2.0）
	PresencePenalty float64

	// Timeout 是请求超时时间（可选，默认 60 秒）
	Timeout time.Duration

	// User 是用户标识（可选，用于追踪滥用）
	User string

	// Organization 是组织 ID（可选）
	Organization string

	// Seed 是随机种子（可选，用于可复现的输出）
	Seed *int
}

// Validate 验证配置的有效性。
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("APIKey is required")
	}

	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("Temperature must be between 0 and 2")
	}

	if c.TopP < 0 || c.TopP > 1 {
		return fmt.Errorf("TopP must be between 0 and 1")
	}

	return nil
}

// ChatModel 是 OpenAI 的 ChatModel 实现。
type ChatModel struct {
	*chat.BaseChatModel
	config Config
	client *http.Client
}

// New 创建一个新的 OpenAI ChatModel。
//
// 参数：
//   - config: OpenAI 配置
//
// 返回：
//   - *ChatModel: ChatModel 实例
//   - error: 配置错误
//
func New(config Config) (*ChatModel, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 设置默认值
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Model == "" {
		config.Model = DefaultModel
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: config.Timeout,
	}

	model := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(config.Model, "openai"),
		config:        config,
		client:        client,
	}

	return model, nil
}

// Invoke 实现 Runnable 接口，执行单次调用。
func (m *ChatModel) Invoke(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
	// 验证消息
	if err := chat.ValidateMessages(messages); err != nil {
		return types.Message{}, err
	}

	// 构建请求
	reqBody, err := m.buildRequest(messages, false)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to build request: %w", err)
	}

	// 发送请求
	respBody, err := m.doRequest(ctx, reqBody)
	if err != nil {
		return types.Message{}, err
	}

	// 解析响应
	var response openAIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return types.Message{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return types.Message{}, fmt.Errorf("no choices in response")
	}

	// 转换为 Message
	message, err := chat.OpenAIResponseToMessage(response.Choices[0].Message)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to convert response: %w", err)
	}

	return message, nil
}

// Stream 实现 Runnable 接口，执行流式调用。
func (m *ChatModel) Stream(ctx context.Context, messages []types.Message, opts ...runnable.Option) (<-chan runnable.StreamEvent[types.Message], error) {
	// 验证消息
	if err := chat.ValidateMessages(messages); err != nil {
		return nil, err
	}

	// 构建请求
	reqBody, err := m.buildRequest(messages, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// 创建输出 channel
	out := make(chan runnable.StreamEvent[types.Message], 10)

	// 启动 goroutine 处理流式响应
	go func() {
		defer close(out)

		// 发送开始事件
		out <- runnable.StreamEvent[types.Message]{
			Type: runnable.EventStart,
			Name: m.GetName(),
		}

		// 发送请求
		resp, err := m.doStreamRequest(ctx, reqBody)
		if err != nil {
			out <- runnable.StreamEvent[types.Message]{
				Type:  runnable.EventError,
				Error: err,
			}
			return
		}
		defer resp.Body.Close()

		// 读取流式响应
		if err := m.processStream(resp.Body, out); err != nil {
			out <- runnable.StreamEvent[types.Message]{
				Type:  runnable.EventError,
				Error: err,
			}
			return
		}
	}()

	return out, nil
}

// BindTools 实现 ChatModel 接口，绑定工具。
func (m *ChatModel) BindTools(tools []types.Tool) chat.ChatModel {
	// 创建新实例
	newModel := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "openai"),
		config:        m.config,
		client:        m.client,
	}

	// 设置工具
	newModel.SetBoundTools(tools)

	return newModel
}

// WithStructuredOutput 实现 ChatModel 接口，配置结构化输出。
func (m *ChatModel) WithStructuredOutput(schema types.Schema) chat.ChatModel {
	// 创建新实例
	newModel := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "openai"),
		config:        m.config,
		client:        m.client,
	}

	// 设置 Schema
	newModel.SetOutputSchema(schema)

	// 复制已绑定的工具
	if len(m.GetBoundTools()) > 0 {
		newModel.SetBoundTools(m.GetBoundTools())
	}

	return newModel
}

// Batch 实现 Runnable 接口的批量执行。
func (m *ChatModel) Batch(ctx context.Context, inputs [][]types.Message, opts ...runnable.Option) ([]types.Message, error) {
	if len(inputs) == 0 {
		return []types.Message{}, nil
	}

	results := make([]types.Message, len(inputs))
	errors := make([]error, len(inputs))

	// 使用 channel 进行并行执行
	type result struct {
		index int
		msg   types.Message
		err   error
	}

	resultChan := make(chan result, len(inputs))

	// 启动 goroutines
	for i, input := range inputs {
		go func(idx int, msgs []types.Message) {
			msg, err := m.Invoke(ctx, msgs, opts...)
			resultChan <- result{index: idx, msg: msg, err: err}
		}(i, input)
	}

	// 收集结果
	for i := 0; i < len(inputs); i++ {
		res := <-resultChan
		results[res.index] = res.msg
		errors[res.index] = res.err
	}

	close(resultChan)

	// 检查是否有错误
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("batch failed at index %d: %w", i, err)
		}
	}

	return results, nil
}

// WithConfig 实现 Runnable 接口。
func (m *ChatModel) WithConfig(config *types.Config) runnable.Runnable[[]types.Message, types.Message] {
	newModel := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "openai"),
		config:        m.config,
		client:        m.client,
	}
	newModel.SetConfig(config)
	newModel.SetBoundTools(m.GetBoundTools())
	if schema := m.GetOutputSchema(); schema != nil {
		newModel.SetOutputSchema(*schema)
	}
	return newModel
}

// WithRetry 实现 Runnable 接口。
func (m *ChatModel) WithRetry(policy types.RetryPolicy) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewRetryRunnable[[]types.Message, types.Message](m, policy)
}

// WithFallbacks 实现 Runnable 接口。
func (m *ChatModel) WithFallbacks(fallbacks ...runnable.Runnable[[]types.Message, types.Message]) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewFallbackRunnable[[]types.Message, types.Message](m, fallbacks)
}

// buildRequest 构建 OpenAI API 请求体。
func (m *ChatModel) buildRequest(messages []types.Message, stream bool) ([]byte, error) {
	// 转换消息格式
	openaiMessages, err := chat.MessagesToOpenAI(messages)
	if err != nil {
		return nil, err
	}

	// 构建请求
	request := map[string]any{
		"model":    m.config.Model,
		"messages": openaiMessages,
		"stream":   stream,
	}

	// 添加可选参数
	if m.config.Temperature > 0 {
		request["temperature"] = m.config.Temperature
	}
	if m.config.MaxTokens > 0 {
		request["max_tokens"] = m.config.MaxTokens
	}
	if m.config.TopP > 0 {
		request["top_p"] = m.config.TopP
	}
	if m.config.FrequencyPenalty != 0 {
		request["frequency_penalty"] = m.config.FrequencyPenalty
	}
	if m.config.PresencePenalty != 0 {
		request["presence_penalty"] = m.config.PresencePenalty
	}
	if m.config.User != "" {
		request["user"] = m.config.User
	}
	if m.config.Seed != nil {
		request["seed"] = *m.config.Seed
	}

	// 添加工具
	tools := m.GetBoundTools()
	if len(tools) > 0 {
		request["tools"] = chat.ConvertToolsToOpenAI(tools)
		request["tool_choice"] = "auto"
	}

	// 添加结构化输出
	if schema := m.GetOutputSchema(); schema != nil {
		request["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "response",
				"schema": schema.ToMap(),
				"strict": true,
			},
		}
	}

	return json.Marshal(request)
}

// doRequest 发送 HTTP 请求（非流式）。
func (m *ChatModel) doRequest(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		m.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.config.APIKey)
	if m.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", m.config.Organization)
	}

	// 发送请求
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, m.parseError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// doStreamRequest 发送流式 HTTP 请求。
func (m *ChatModel) doStreamRequest(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		m.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.config.APIKey)
	req.Header.Set("Accept", "text/event-stream")
	if m.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", m.config.Organization)
	}

	// 发送请求
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, m.parseError(resp.StatusCode, respBody)
	}

	return resp, nil
}

// processStream 处理流式响应。
func (m *ChatModel) processStream(reader io.Reader, out chan<- runnable.StreamEvent[types.Message]) error {
	scanner := bufio.NewScanner(reader)
	var fullMessage types.Message
	fullMessage.Role = types.RoleAssistant

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 解析 SSE 数据
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 检查结束标记
		if data == "[DONE]" {
			// 发送结束事件
			out <- runnable.StreamEvent[types.Message]{
				Type: runnable.EventEnd,
				Data: fullMessage,
				Name: m.GetName(),
			}
			return nil
		}

		// 解析 JSON
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// 累积内容
		if delta.Content != "" {
			fullMessage.Content += delta.Content

			// 发送流式数据事件
			out <- runnable.StreamEvent[types.Message]{
				Type: runnable.EventStream,
				Data: types.Message{
					Role:    types.RoleAssistant,
					Content: delta.Content,
				},
				Name: m.GetName(),
			}
		}

		// 处理工具调用
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				// 查找或创建对应的 ToolCall
				if tc.Index >= len(fullMessage.ToolCalls) {
					// 扩展数组
					for i := len(fullMessage.ToolCalls); i <= tc.Index; i++ {
						fullMessage.ToolCalls = append(fullMessage.ToolCalls, types.ToolCall{})
					}
				}

				// 累积工具调用信息
				if tc.ID != "" {
					fullMessage.ToolCalls[tc.Index].ID = tc.ID
				}
				if tc.Type != "" {
					fullMessage.ToolCalls[tc.Index].Type = tc.Type
				}
				if tc.Function.Name != "" {
					fullMessage.ToolCalls[tc.Index].Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					fullMessage.ToolCalls[tc.Index].Function.Arguments += tc.Function.Arguments
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

// parseError 解析错误响应。
func (m *ChatModel) parseError(statusCode int, body []byte) error {
	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}

	return fmt.Errorf("OpenAI API error (HTTP %d): %s", statusCode, errResp.Error.Message)
}

// openAIResponse 是 OpenAI API 的响应结构。
type openAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int            `json:"index"`
	Message      map[string]any `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// streamChunk 是流式响应的数据块。
type streamChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []streamChoice `json:"choices"`
}

type streamChoice struct {
	Index        int         `json:"index"`
	Delta        streamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type streamDelta struct {
	Role      string            `json:"role,omitempty"`
	Content   string            `json:"content,omitempty"`
	ToolCalls []streamToolCall  `json:"tool_calls,omitempty"`
}

type streamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

// errorResponse 是 OpenAI API 的错误响应。
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

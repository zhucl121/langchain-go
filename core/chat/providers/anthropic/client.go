package anthropic

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
	// DefaultBaseURL 是 Anthropic API 的默认地址
	DefaultBaseURL = "https://api.anthropic.com"

	// DefaultModel 是默认使用的模型
	DefaultModel = "claude-3-5-sonnet-20241022"

	// DefaultTimeout 是默认的请求超时时间
	DefaultTimeout = 60 * time.Second

	// DefaultMaxTokens 是默认的最大生成 token 数
	DefaultMaxTokens = 4096

	// APIVersion 是 Anthropic API 版本
	APIVersion = "2023-06-01"
)

// Config 是 Anthropic ChatModel 的配置。
type Config struct {
	// APIKey 是 Anthropic API 密钥（必需）
	APIKey string

	// BaseURL 是 API 基础地址（可选，默认为 Anthropic 官方地址）
	BaseURL string

	// Model 是模型名称（可选，默认为 claude-3-5-sonnet-20241022）
	// 支持的模型：
	//   - claude-3-opus-20240229
	//   - claude-3-sonnet-20240229
	//   - claude-3-haiku-20240307
	//   - claude-3-5-sonnet-20241022
	Model string

	// Temperature 控制输出的随机性（可选，0.0-1.0，默认 1.0）
	Temperature float64

	// MaxTokens 是最大生成 token 数（必需，Anthropic 要求显式指定）
	MaxTokens int

	// TopP 是核采样参数（可选，0.0-1.0）
	TopP float64

	// TopK 是 top-k 采样参数（可选）
	TopK int

	// Timeout 是请求超时时间（可选，默认 60 秒）
	Timeout time.Duration
}

// Validate 验证配置的有效性。
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("APIKey is required")
	}

	// Anthropic 要求必须指定 MaxTokens
	if c.MaxTokens <= 0 {
		return fmt.Errorf("MaxTokens is required and must be positive")
	}

	if c.Temperature < 0 || c.Temperature > 1 {
		return fmt.Errorf("Temperature must be between 0 and 1")
	}

	if c.TopP < 0 || c.TopP > 1 {
		return fmt.Errorf("TopP must be between 0 and 1")
	}

	return nil
}

// ChatModel 是 Anthropic 的 ChatModel 实现。
type ChatModel struct {
	*chat.BaseChatModel
	config Config
	client *http.Client
}

// New 创建一个新的 Anthropic ChatModel。
//
// 参数：
//   - config: Anthropic 配置
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
		config.Temperature = 1.0
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: config.Timeout,
	}

	model := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(config.Model, "anthropic"),
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
	var response anthropicResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return types.Message{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换为 Message
	message, err := chat.AnthropicResponseToMessage(response.Content)
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
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "anthropic"),
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
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "anthropic"),
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
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "anthropic"),
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

// buildRequest 构建 Anthropic API 请求体。
func (m *ChatModel) buildRequest(messages []types.Message, stream bool) ([]byte, error) {
	// 转换消息格式（Anthropic 要求提取系统消息）
	systemMessage, anthropicMessages, err := chat.MessagesToAnthropic(messages)
	if err != nil {
		return nil, err
	}

	// 构建请求
	request := map[string]any{
		"model":      m.config.Model,
		"messages":   anthropicMessages,
		"max_tokens": m.config.MaxTokens,
		"stream":     stream,
	}

	// 添加系统消息
	if systemMessage != "" {
		request["system"] = systemMessage
	}

	// 添加可选参数
	if m.config.Temperature > 0 && m.config.Temperature != 1.0 {
		request["temperature"] = m.config.Temperature
	}
	if m.config.TopP > 0 {
		request["top_p"] = m.config.TopP
	}
	if m.config.TopK > 0 {
		request["top_k"] = m.config.TopK
	}

	// 添加工具
	tools := m.GetBoundTools()
	if len(tools) > 0 {
		request["tools"] = chat.ConvertToolsToAnthropic(tools)
	}

	return json.Marshal(request)
}

// doRequest 发送 HTTP 请求（非流式）。
func (m *ChatModel) doRequest(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		m.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", m.config.APIKey)
	req.Header.Set("anthropic-version", APIVersion)

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
		m.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", m.config.APIKey)
	req.Header.Set("anthropic-version", APIVersion)
	req.Header.Set("Accept", "text/event-stream")

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

	var currentToolCalls []types.ToolCall

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 解析 SSE 事件
		if !strings.HasPrefix(line, "event: ") {
			continue
		}

		eventType := strings.TrimPrefix(line, "event: ")

		// 读取下一行（data）
		if !scanner.Scan() {
			break
		}

		dataLine := scanner.Text()
		if !strings.HasPrefix(dataLine, "data: ") {
			continue
		}

		data := strings.TrimPrefix(dataLine, "data: ")

		// 根据事件类型处理
		switch eventType {
		case "message_start":
			// 消息开始，暂不处理

		case "content_block_start":
			// 内容块开始
			var event contentBlockStartEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			// 如果是工具使用块，初始化
			if event.ContentBlock.Type == "tool_use" {
				tc := types.ToolCall{
					Type: "function",
					ID:   event.ContentBlock.ID,
					Function: types.FunctionCall{
						Name:      event.ContentBlock.Name,
						Arguments: "",
					},
				}
				currentToolCalls = append(currentToolCalls, tc)
			}

		case "content_block_delta":
			// 内容块增量更新
			var event contentBlockDeltaEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if event.Delta.Type == "text_delta" {
				// 文本内容
				fullMessage.Content += event.Delta.Text

				// 发送流式数据事件
				out <- runnable.StreamEvent[types.Message]{
					Type: runnable.EventStream,
					Data: types.Message{
						Role:    types.RoleAssistant,
						Content: event.Delta.Text,
					},
					Name: m.GetName(),
				}
			} else if event.Delta.Type == "input_json_delta" {
				// 工具参数增量
				if event.Index < len(currentToolCalls) {
					currentToolCalls[event.Index].Function.Arguments += event.Delta.PartialJSON
				}
			}

		case "content_block_stop":
			// 内容块结束，暂不处理

		case "message_delta":
			// 消息级别的增量更新，暂不处理

		case "message_stop":
			// 消息结束
			// 将工具调用添加到完整消息
			if len(currentToolCalls) > 0 {
				fullMessage.ToolCalls = currentToolCalls
			}

			// 发送结束事件
			out <- runnable.StreamEvent[types.Message]{
				Type: runnable.EventEnd,
				Data: fullMessage,
				Name: m.GetName(),
			}
			return nil

		case "error":
			// 错误事件
			var errEvent errorEvent
			if err := json.Unmarshal([]byte(data), &errEvent); err != nil {
				return fmt.Errorf("stream error")
			}
			return fmt.Errorf("stream error: %s", errEvent.Error.Message)
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

	return fmt.Errorf("Anthropic API error (HTTP %d): %s", statusCode, errResp.Error.Message)
}

// anthropicResponse 是 Anthropic API 的响应结构。
type anthropicResponse struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Role         string  `json:"role"`
	Content      []any   `json:"content"`
	Model        string  `json:"model"`
	StopReason   string  `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
	Usage        usage   `json:"usage"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// 流式事件结构

type contentBlockStartEvent struct {
	Index        int          `json:"index"`
	ContentBlock contentBlock `json:"content_block"`
}

type contentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type contentBlockDeltaEvent struct {
	Index int   `json:"index"`
	Delta delta `json:"delta"`
}

type delta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type errorEvent struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// errorResponse 是 Anthropic API 的错误响应。
type errorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

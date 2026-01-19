package ollama

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// StreamTokens 返回 token 级别的流式响应。
//
// 参数：
//   - ctx: 上下文
//   - messages: 输入消息列表
//
// 返回：
//   - <-chan types.StreamEvent: 流式事件 channel
//   - error: 错误
//
func (m *ChatModel) StreamTokens(ctx context.Context, messages []types.Message) (<-chan types.StreamEvent, error) {
	// 构建请求体
	reqBody, err := m.buildRequestBody(messages, true)
	if err != nil {
		return nil, err
	}

	// 创建输出 channel
	out := make(chan types.StreamEvent, 100)

	go func() {
		defer close(out)

		// 发送开始事件
		out <- types.StreamEvent{
			Type: types.StreamEventStart,
			Metadata: map[string]any{
				"model":    m.GetModelName(),
				"provider": m.GetProvider(),
			},
		}

		// 发送请求
		resp, err := m.doStreamRequest(ctx, reqBody)
		if err != nil {
			out <- types.NewErrorEvent(err)
			return
		}
		defer resp.Body.Close()

		// 处理流式响应
		if err := m.processTokenStream(resp.Body, out); err != nil {
			out <- types.NewErrorEvent(err)
			return
		}

		// 发送结束事件
		out <- types.StreamEvent{
			Type: types.StreamEventEnd,
			Done: true,
		}
	}()

	return out, nil
}

// processTokenStream 处理 token 级别的流式响应。
func (m *ChatModel) processTokenStream(reader io.Reader, out chan<- types.StreamEvent) error {
	scanner := bufio.NewScanner(reader)
	var fullContent strings.Builder
	index := 0

	for scanner.Scan() {
		line := scanner.Text()

		// 解析 JSON
		var chunk ollamaStreamResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		// 发送 token 事件
		if chunk.Message.Content != "" {
			fullContent.WriteString(chunk.Message.Content)

			event := types.NewTokenEvent(chunk.Message.Content)
			event.Index = index
			event.Metadata = map[string]any{
				"model":        m.GetModelName(),
				"total_length": fullContent.Len(),
				"done":         chunk.Done,
			}

			out <- event
			index++
		}

		// 处理完成
		if chunk.Done {
			// 发送最终内容事件
			event := types.StreamEvent{
				Type:    types.StreamEventContent,
				Content: fullContent.String(),
				Metadata: map[string]any{
					"model":         m.GetModelName(),
					"total_duration": chunk.TotalDuration,
					"load_duration":  chunk.LoadDuration,
					"prompt_eval_count": chunk.PromptEvalCount,
					"eval_count":    chunk.EvalCount,
				},
			}
			out <- event
			return nil
		}
	}

	return scanner.Err()
}

// StreamWithAggregation 返回聚合后的流式响应。
//
// 与 StreamTokens 不同，此方法返回完整的消息块而非单个 token。
//
// 参数：
//   - ctx: 上下文
//   - messages: 输入消息列表
//
// 返回：
//   - <-chan types.StreamEvent: 流式事件 channel
//   - error: 错误
//
func (m *ChatModel) StreamWithAggregation(ctx context.Context, messages []types.Message) (<-chan types.StreamEvent, error) {
	tokenStream, err := m.StreamTokens(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 创建聚合输出 channel
	out := make(chan types.StreamEvent, 100)

	go func() {
		defer close(out)

		var contentBuilder strings.Builder

		for event := range tokenStream {
			switch event.Type {
			case types.StreamEventStart:
				out <- event

			case types.StreamEventToken:
				// 累积内容
				contentBuilder.WriteString(event.Token)

				// 发送聚合的内容事件
				aggregatedEvent := types.StreamEvent{
					Type:     types.StreamEventContent,
					Content:  contentBuilder.String(),
					Index:    event.Index,
					Metadata: event.Metadata,
				}
				out <- aggregatedEvent

			case types.StreamEventEnd:
				// 发送最终事件
				finalEvent := types.StreamEvent{
					Type:     types.StreamEventContent,
					Content:  contentBuilder.String(),
					Done:     true,
					Metadata: map[string]any{},
				}
				out <- finalEvent
				out <- event

			case types.StreamEventError:
				out <- event
				return
			}
		}
	}()

	return out, nil
}

// buildRequestBody 构建请求体（支持流式）。
func (m *ChatModel) buildRequestBody(messages []types.Message, stream bool) ([]byte, error) {
	// 转换消息格式
	apiMessages := make([]map[string]string, len(messages))
	for i, msg := range messages {
		apiMessages[i] = map[string]string{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}

	// 构建请求
	reqMap := map[string]any{
		"model":    m.config.Model,
		"messages": apiMessages,
		"stream":   stream,
	}

	// 添加选项
	options := map[string]any{}
	if m.config.Temperature > 0 {
		options["temperature"] = m.config.Temperature
	}
	if m.config.NumPredict > 0 {
		options["num_predict"] = m.config.NumPredict
	}
	if m.config.TopK > 0 {
		options["top_k"] = m.config.TopK
	}
	if m.config.TopP > 0 {
		options["top_p"] = m.config.TopP
	}
	if m.config.RepeatPenalty > 0 {
		options["repeat_penalty"] = m.config.RepeatPenalty
	}
	if m.config.Seed != nil {
		options["seed"] = *m.config.Seed
	}

	if len(options) > 0 {
		reqMap["options"] = options
	}

	// 添加格式（JSON 模式）
	if m.config.Format != "" {
		reqMap["format"] = m.config.Format
	}

	// 添加结构化输出
	if schema := m.GetOutputSchema(); schema != nil {
		reqMap["format"] = "json"
	}

	return json.Marshal(reqMap)
}

// ollamaStreamResponse 表示 Ollama 流式响应。
type ollamaStreamResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done             bool   `json:"done"`
	TotalDuration    int64  `json:"total_duration,omitempty"`
	LoadDuration     int64  `json:"load_duration,omitempty"`
	PromptEvalCount  int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount        int    `json:"eval_count,omitempty"`
	EvalDuration     int64  `json:"eval_duration,omitempty"`
}

package openai

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
	var toolCalls []types.ToolCall
	index := 0

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

		// 发送 token 事件
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)

			event := types.NewTokenEvent(delta.Content)
			event.Index = index
			event.Metadata = map[string]any{
				"model":        m.GetModelName(),
				"total_length": fullContent.Len(),
			}
			if chunk.Choices[0].FinishReason != nil {
				event.Metadata["finish_reason"] = *chunk.Choices[0].FinishReason
			}

			out <- event
			index++
		}

		// 处理工具调用
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				// 查找或创建对应的 ToolCall
				if tc.Index >= len(toolCalls) {
					// 扩展数组
					for i := len(toolCalls); i <= tc.Index; i++ {
						toolCalls = append(toolCalls, types.ToolCall{})
					}
				}

				// 累积工具调用数据
				if tc.ID != "" {
					toolCalls[tc.Index].ID = tc.ID
				}
				if tc.Type != "" {
					toolCalls[tc.Index].Type = tc.Type
				}
				if tc.Function.Name != "" {
					toolCalls[tc.Index].Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					toolCalls[tc.Index].Function.Arguments += tc.Function.Arguments
				}
			}

			// 发送工具调用事件
			if tc := delta.ToolCalls[0]; tc.Function.Arguments != "" {
				// 工具调用完整时发送
				if toolCalls[tc.Index].Function.Arguments != "" {
					event := types.NewToolCallEvent(&toolCalls[tc.Index])
					event.Index = index
					out <- event
					index++
				}
			}
		}

		// 处理结束原因
		if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason != "" {
			event := types.StreamEvent{
				Type: types.StreamEventContent,
				Content: fullContent.String(),
				Metadata: map[string]any{
					"finish_reason": *chunk.Choices[0].FinishReason,
					"tool_calls":    len(toolCalls),
				},
			}
			out <- event
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
		var toolCalls []types.ToolCall

		for event := range tokenStream {
			switch event.Type {
			case types.StreamEventStart:
				out <- event

			case types.StreamEventToken:
				// 累积内容
				contentBuilder.WriteString(event.Token)

				// 发送聚合的内容事件
				aggregatedEvent := types.StreamEvent{
					Type:    types.StreamEventContent,
					Content: contentBuilder.String(),
					Index:   event.Index,
					Metadata: event.Metadata,
				}
				out <- aggregatedEvent

			case types.StreamEventToolCall:
				if event.ToolCall != nil {
					toolCalls = append(toolCalls, *event.ToolCall)
				}
				out <- event

			case types.StreamEventEnd:
				// 发送最终事件
				finalEvent := types.StreamEvent{
					Type:    types.StreamEventContent,
					Content: contentBuilder.String(),
					Done:    true,
					Metadata: map[string]any{
						"tool_calls": len(toolCalls),
					},
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
	apiMessages := make([]map[string]any, len(messages))
	for i, msg := range messages {
		apiMessages[i] = map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}

		// 添加 name（如果有）
		if msg.Name != "" {
			apiMessages[i]["name"] = msg.Name
		}

		// 添加工具调用
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]any, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
			}
			apiMessages[i]["tool_calls"] = toolCalls
		}

		// 添加工具调用 ID（用于工具响应）
		if msg.ToolCallID != "" {
			apiMessages[i]["tool_call_id"] = msg.ToolCallID
		}
	}

	// 构建请求
	reqMap := map[string]any{
		"model":    m.config.Model,
		"messages": apiMessages,
		"stream":   stream,
	}

	// 添加可选参数
	if m.config.Temperature > 0 {
		reqMap["temperature"] = m.config.Temperature
	}
	if m.config.MaxTokens > 0 {
		reqMap["max_tokens"] = m.config.MaxTokens
	}
	if m.config.TopP > 0 {
		reqMap["top_p"] = m.config.TopP
	}
	if m.config.FrequencyPenalty != 0 {
		reqMap["frequency_penalty"] = m.config.FrequencyPenalty
	}
	if m.config.PresencePenalty != 0 {
		reqMap["presence_penalty"] = m.config.PresencePenalty
	}
	if m.config.User != "" {
		reqMap["user"] = m.config.User
	}
	if m.config.Seed != nil {
		reqMap["seed"] = *m.config.Seed
	}

	// 添加绑定的工具
	if len(m.GetBoundTools()) > 0 {
		tools := make([]map[string]any, len(m.GetBoundTools()))
		for i, tool := range m.GetBoundTools() {
			tools[i] = tool.ToOpenAITool()
		}
		reqMap["tools"] = tools
	}

	// 添加结构化输出
	if schema := m.GetOutputSchema(); schema != nil {
		reqMap["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "response",
				"schema": schema.ToMap(),
				"strict": true,
			},
		}
	}

	return json.Marshal(reqMap)
}

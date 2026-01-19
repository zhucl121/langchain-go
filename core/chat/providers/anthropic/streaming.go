package anthropic

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
	reqBody, err := m.buildRequest(messages, true)
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
	var toolUses []types.ToolCall
	index := 0

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 解析 SSE 数据
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 解析 JSON
		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_start":
			// 内容块开始
			if event.ContentBlock.Type == "text" {
				// 文本块开始
			} else if event.ContentBlock.Type == "tool_use" {
				// 工具使用开始
				toolUse := types.ToolCall{
					ID:   event.ContentBlock.ID,
					Type: "function",
					Function: types.FunctionCall{
						Name:      event.ContentBlock.Name,
						Arguments: "",
					},
				}
				toolUses = append(toolUses, toolUse)
			}

		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				// 文本增量
				token := event.Delta.Text
				fullContent.WriteString(token)

				tokenEvent := types.NewTokenEvent(token)
				tokenEvent.Index = index
				tokenEvent.Metadata = map[string]any{
					"model":        m.GetModelName(),
					"total_length": fullContent.Len(),
				}

				out <- tokenEvent
				index++

			} else if event.Delta.Type == "input_json_delta" {
				// 工具参数增量
				if len(toolUses) > 0 && event.Index < len(toolUses) {
					toolUses[event.Index].Function.Arguments += event.Delta.PartialJSON
				}
			}

		case "content_block_stop":
			// 内容块结束
			if len(toolUses) > event.Index {
				// 发送工具调用事件
				toolEvent := types.NewToolCallEvent(&toolUses[event.Index])
				toolEvent.Index = index
				out <- toolEvent
				index++
			}

		case "message_delta":
			// 消息增量（通常包含停止原因）
			if event.Delta.StopReason != "" {
				contentEvent := types.StreamEvent{
					Type:    types.StreamEventContent,
					Content: fullContent.String(),
					Metadata: map[string]any{
						"stop_reason": event.Delta.StopReason,
						"tool_calls":  len(toolUses),
					},
				}
				out <- contentEvent
			}

		case "message_stop":
			// 消息结束
			return nil

		case "error":
			// 错误
			return event.Error
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
					Type:     types.StreamEventContent,
					Content:  contentBuilder.String(),
					Index:    event.Index,
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

// anthropicStreamEvent 表示 Anthropic SSE 流式事件。
type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`

	// message_start
	Message *struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Content      []any  `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`

	// content_block_start
	ContentBlock *struct {
		Type  string `json:"type"`
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Input any    `json:"input,omitempty"`
		Text  string `json:"text,omitempty"`
	} `json:"content_block,omitempty"`

	// content_block_delta
	Delta *struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
		StopReason  string `json:"stop_reason,omitempty"`
		StopSequence string `json:"stop_sequence,omitempty"`
	} `json:"delta,omitempty"`

	// error
	Error *anthropicError `json:"error,omitempty"`
}

// anthropicError 表示 Anthropic 错误。
type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *anthropicError) Error() string {
	return e.Message
}

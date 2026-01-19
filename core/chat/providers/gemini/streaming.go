package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
func (c *GeminiClient) StreamTokens(ctx context.Context, messages []types.Message, opts ...Option) (<-chan types.StreamEvent, error) {
	// 应用选项
	config := c.config
	for _, opt := range opts {
		opt(&config)
	}

	// 转换消息格式
	contents, err := c.convertMessages(messages)
	if err != nil {
		return nil, err
	}

	// 构建请求
	reqBody := GeminiRequest{
		Contents: contents,
		GenerationConfig: GenerationConfig{
			Temperature:     config.Temperature,
			TopP:            config.TopP,
			TopK:            config.TopK,
			MaxOutputTokens: config.MaxTokens,
			StopSequences:   config.StopSequences,
		},
		SafetySettings: config.SafetySettings,
	}

	// 创建输出 channel
	out := make(chan types.StreamEvent, 100)

	go func() {
		defer close(out)

		// 发送开始事件
		out <- types.StreamEvent{
			Type: types.StreamEventStart,
			Metadata: map[string]any{
				"model":    config.Model,
				"provider": "gemini",
			},
		}

		// 发送请求
		resp, err := c.doStreamRequest(ctx, reqBody)
		if err != nil {
			out <- types.NewErrorEvent(err)
			return
		}
		defer resp.Body.Close()

		// 处理流式响应
		if err := c.processTokenStream(resp.Body, out); err != nil {
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
func (c *GeminiClient) processTokenStream(reader io.Reader, out chan<- types.StreamEvent) error {
	scanner := bufio.NewScanner(reader)
	var fullContent strings.Builder
	index := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Gemini 使用 JSON 数组格式的流式响应
		// 每行是一个完整的 JSON 对象
		if line == "" || line == "[" || line == "]" {
			continue
		}

		// 去除可能的逗号
		line = strings.TrimSuffix(strings.TrimSpace(line), ",")

		// 解析 JSON
		var chunk GeminiResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		// 处理候选项
		if len(chunk.Candidates) == 0 {
			continue
		}

		candidate := chunk.Candidates[0]

		// 处理内容部分
		if len(candidate.Content.Parts) > 0 {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					fullContent.WriteString(part.Text)

					tokenEvent := types.NewTokenEvent(part.Text)
					tokenEvent.Index = index
					tokenEvent.Metadata = map[string]any{
						"model":         c.config.Model,
						"total_length":  fullContent.Len(),
						"finish_reason": candidate.FinishReason,
					}

					out <- tokenEvent
					index++
				}
			}
		}

		// 检查是否完成
		if candidate.FinishReason != "" && candidate.FinishReason != "STOP" {
			// 发送内容事件
			contentEvent := types.StreamEvent{
				Type:    types.StreamEventContent,
				Content: fullContent.String(),
				Metadata: map[string]any{
					"finish_reason": candidate.FinishReason,
				},
			}
			out <- contentEvent
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
//   - opts: 选项
//
// 返回：
//   - <-chan types.StreamEvent: 流式事件 channel
//   - error: 错误
//
func (c *GeminiClient) StreamWithAggregation(ctx context.Context, messages []types.Message, opts ...Option) (<-chan types.StreamEvent, error) {
	tokenStream, err := c.StreamTokens(ctx, messages, opts...)
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

// doStreamRequest 执行流式请求。
func (c *GeminiClient) doStreamRequest(ctx context.Context, reqBody GeminiRequest) (*http.Response, error) {
	// 构建 URL（添加 streamGenerateContent 和 alt=sse）
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse",
		c.config.BaseURL,
		c.config.Model,
		c.config.APIKey)

	// 序列化请求体
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to marshal request: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini: API error (status %d): %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

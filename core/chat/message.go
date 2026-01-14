package chat

import (
	"encoding/json"
	"fmt"

	"langchain-go/pkg/types"
)

// MessageConverter 提供消息格式转换的工具函数。
//
// 不同的 LLM 提供商有不同的消息格式要求，MessageConverter
// 提供了统一的转换接口，简化提供商实现。

// MessagesToOpenAI 将消息列表转换为 OpenAI API 格式。
//
// OpenAI 消息格式：
//   - role: "system" | "user" | "assistant" | "tool"
//   - content: 消息内容（字符串）
//   - tool_calls: 工具调用列表（仅 assistant）
//   - tool_call_id: 工具调用 ID（仅 tool）
//
// 参数：
//   - messages: 消息列表
//
// 返回：
//   - []map[string]any: OpenAI 格式的消息列表
//   - error: 转换错误
//
func MessagesToOpenAI(messages []types.Message) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(messages))

	for i, msg := range messages {
		if err := msg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid message at index %d: %w", i, err)
		}

		openaiMsg := map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}

		// 添加可选字段
		if msg.Name != "" {
			openaiMsg["name"] = msg.Name
		}

		// 处理工具调用（仅 assistant 消息）
		if msg.Role == types.RoleAssistant && len(msg.ToolCalls) > 0 {
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
			openaiMsg["tool_calls"] = toolCalls
		}

		// 处理工具结果（仅 tool 消息）
		if msg.Role == types.RoleTool {
			openaiMsg["tool_call_id"] = msg.ToolCallID
		}

		result = append(result, openaiMsg)
	}

	return result, nil
}

// MessagesToAnthropic 将消息列表转换为 Anthropic API 格式。
//
// Anthropic 消息格式特点：
//   - system 消息单独处理，不在 messages 数组中
//   - role 只有 "user" 和 "assistant"
//   - tool_use 和 tool_result 作为 content 的一部分
//
// 参数：
//   - messages: 消息列表
//
// 返回：
//   - system: 系统消息内容（可能为空）
//   - []map[string]any: Anthropic 格式的消息列表
//   - error: 转换错误
//
func MessagesToAnthropic(messages []types.Message) (string, []map[string]any, error) {
	systemMessage := ""
	result := make([]map[string]any, 0, len(messages))

	for i, msg := range messages {
		if err := msg.Validate(); err != nil {
			return "", nil, fmt.Errorf("invalid message at index %d: %w", i, err)
		}

		// 处理系统消息
		if msg.Role == types.RoleSystem {
			if systemMessage != "" {
				systemMessage += "\n\n" + msg.Content
			} else {
				systemMessage = msg.Content
			}
			continue
		}

		// 工具消息转换为 user 消息（Anthropic 要求）
		role := string(msg.Role)
		if msg.Role == types.RoleTool {
			role = "user"
		}

		anthropicMsg := map[string]any{
			"role": role,
		}

		// 处理内容
		if msg.Role == types.RoleAssistant && len(msg.ToolCalls) > 0 {
			// Assistant 消息带工具调用
			content := make([]map[string]any, 0)

			// 如果有文本内容，添加 text block
			if msg.Content != "" {
				content = append(content, map[string]any{
					"type": "text",
					"text": msg.Content,
				})
			}

			// 添加 tool_use blocks
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return "", nil, fmt.Errorf("failed to parse tool arguments: %w", err)
				}

				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": args,
				})
			}

			anthropicMsg["content"] = content
		} else if msg.Role == types.RoleTool {
			// Tool 结果消息
			content := []map[string]any{
				{
					"type":         "tool_result",
					"tool_use_id":  msg.ToolCallID,
					"content":      msg.Content,
				},
			}
			anthropicMsg["content"] = content
		} else {
			// 普通消息
			anthropicMsg["content"] = msg.Content
		}

		result = append(result, anthropicMsg)
	}

	return systemMessage, result, nil
}

// OpenAIResponseToMessage 将 OpenAI 响应转换为 Message。
//
// 参数：
//   - response: OpenAI API 响应的 message 部分
//
// 返回：
//   - types.Message: 转换后的消息
//   - error: 转换错误
//
func OpenAIResponseToMessage(response map[string]any) (types.Message, error) {
	msg := types.Message{
		Role: types.RoleAssistant,
	}

	// 提取内容
	if content, ok := response["content"].(string); ok {
		msg.Content = content
	}

	// 提取工具调用
	if toolCallsRaw, ok := response["tool_calls"]; ok {
		if toolCallsArray, ok := toolCallsRaw.([]any); ok {
			toolCalls := make([]types.ToolCall, 0, len(toolCallsArray))

			for _, tcRaw := range toolCallsArray {
				tcMap, ok := tcRaw.(map[string]any)
				if !ok {
					continue
				}

				tc := types.ToolCall{}

				if id, ok := tcMap["id"].(string); ok {
					tc.ID = id
				}

				if tcType, ok := tcMap["type"].(string); ok {
					tc.Type = tcType
				} else {
					tc.Type = "function"
				}

				if funcRaw, ok := tcMap["function"].(map[string]any); ok {
					if name, ok := funcRaw["name"].(string); ok {
						tc.Function.Name = name
					}
					if args, ok := funcRaw["arguments"].(string); ok {
						tc.Function.Arguments = args
					}
				}

				toolCalls = append(toolCalls, tc)
			}

			msg.ToolCalls = toolCalls
		}
	}

	return msg, nil
}

// AnthropicResponseToMessage 将 Anthropic 响应转换为 Message。
//
// 参数：
//   - content: Anthropic API 响应的 content 数组
//
// 返回：
//   - types.Message: 转换后的消息
//   - error: 转换错误
//
func AnthropicResponseToMessage(content []any) (types.Message, error) {
	msg := types.Message{
		Role: types.RoleAssistant,
	}

	var textParts []string
	var toolCalls []types.ToolCall

	for _, blockRaw := range content {
		block, ok := blockRaw.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)

		switch blockType {
		case "text":
			if text, ok := block["text"].(string); ok {
				textParts = append(textParts, text)
			}

		case "tool_use":
			tc := types.ToolCall{
				Type: "function",
			}

			if id, ok := block["id"].(string); ok {
				tc.ID = id
			}

			if name, ok := block["name"].(string); ok {
				tc.Function.Name = name
			}

			if input, ok := block["input"].(map[string]any); ok {
				// 将 input 转换为 JSON 字符串
				argsBytes, err := json.Marshal(input)
				if err != nil {
					return msg, fmt.Errorf("failed to marshal tool arguments: %w", err)
				}
				tc.Function.Arguments = string(argsBytes)
			}

			toolCalls = append(toolCalls, tc)
		}
	}

	// 合并文本内容
	if len(textParts) > 0 {
		msg.Content = textParts[0]
		for i := 1; i < len(textParts); i++ {
			msg.Content += "\n" + textParts[i]
		}
	}

	msg.ToolCalls = toolCalls

	return msg, nil
}

// MergeMessages 合并相邻的相同角色的消息。
//
// 某些提供商（如 Anthropic）不允许连续的相同角色消息，
// 此函数可以将它们合并。
//
// 参数：
//   - messages: 消息列表
//
// 返回：
//   - []types.Message: 合并后的消息列表
//
func MergeMessages(messages []types.Message) []types.Message {
	if len(messages) == 0 {
		return messages
	}

	result := make([]types.Message, 0, len(messages))
	current := messages[0].Clone()

	for i := 1; i < len(messages); i++ {
		msg := messages[i]

		// 如果角色相同且都没有工具调用，则合并
		if msg.Role == current.Role &&
			len(msg.ToolCalls) == 0 &&
			len(current.ToolCalls) == 0 &&
			msg.ToolCallID == "" &&
			current.ToolCallID == "" {

			current.Content += "\n" + msg.Content
		} else {
			// 否则，保存当前消息并开始新的
			result = append(result, current)
			current = msg.Clone()
		}
	}

	// 添加最后一条消息
	result = append(result, current)

	return result
}

// ExtractSystemMessage 提取系统消息。
//
// 从消息列表中提取所有系统消息，并返回剩余消息。
//
// 参数：
//   - messages: 消息列表
//
// 返回：
//   - systemContent: 合并的系统消息内容
//   - remaining: 剩余消息列表
//
func ExtractSystemMessage(messages []types.Message) (string, []types.Message) {
	var systemParts []string
	remaining := make([]types.Message, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			systemParts = append(systemParts, msg.Content)
		} else {
			remaining = append(remaining, msg)
		}
	}

	systemContent := ""
	if len(systemParts) > 0 {
		systemContent = systemParts[0]
		for i := 1; i < len(systemParts); i++ {
			systemContent += "\n\n" + systemParts[i]
		}
	}

	return systemContent, remaining
}

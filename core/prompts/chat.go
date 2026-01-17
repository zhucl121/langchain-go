package prompts

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// MessagePromptTemplate 是单个消息的模板接口。
//
// MessagePromptTemplate 表示一个特定角色（system、user、assistant）的消息模板。
type MessagePromptTemplate interface {
	// Format 格式化消息模板
	Format(values map[string]any) (types.Message, error)

	// GetInputVariables 获取输入变量列表
	GetInputVariables() []string
}

// BaseMessagePromptTemplate 是消息模板的基础实现。
type BaseMessagePromptTemplate struct {
	Prompt *PromptTemplate
	Role   types.Role
}

// Format 实现 MessagePromptTemplate 接口。
func (b *BaseMessagePromptTemplate) Format(values map[string]any) (types.Message, error) {
	content, err := b.Prompt.Format(values)
	if err != nil {
		return types.Message{}, err
	}

	return types.Message{
		Role:    b.Role,
		Content: content,
	}, nil
}

// GetInputVariables 实现 MessagePromptTemplate 接口。
func (b *BaseMessagePromptTemplate) GetInputVariables() []string {
	return b.Prompt.InputVariables
}

// SystemMessagePromptTemplate 创建系统消息模板。
//
// 参数：
//   - template: 模板字符串
//
// 返回：
//   - MessagePromptTemplate: 系统消息模板
//
// 示例：
//
//	template := SystemMessagePromptTemplate("You are a {role}.")
//	msg, _ := template.Format(map[string]any{"role": "helpful assistant"})
//
func SystemMessagePromptTemplate(template string) MessagePromptTemplate {
	prompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: template,
	})
	return &BaseMessagePromptTemplate{
		Prompt: prompt,
		Role:   types.RoleSystem,
	}
}

// HumanMessagePromptTemplate 创建用户消息模板（别名：UserMessagePromptTemplate）。
//
// 参数：
//   - template: 模板字符串
//
// 返回：
//   - MessagePromptTemplate: 用户消息模板
//
func HumanMessagePromptTemplate(template string) MessagePromptTemplate {
	prompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: template,
	})
	return &BaseMessagePromptTemplate{
		Prompt: prompt,
		Role:   types.RoleUser,
	}
}

// UserMessagePromptTemplate 是 HumanMessagePromptTemplate 的别名。
func UserMessagePromptTemplate(template string) MessagePromptTemplate {
	return HumanMessagePromptTemplate(template)
}

// AIMessagePromptTemplate 创建 AI 消息模板（别名：AssistantMessagePromptTemplate）。
//
// 参数：
//   - template: 模板字符串
//
// 返回：
//   - MessagePromptTemplate: AI 消息模板
//
func AIMessagePromptTemplate(template string) MessagePromptTemplate {
	prompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: template,
	})
	return &BaseMessagePromptTemplate{
		Prompt: prompt,
		Role:   types.RoleAssistant,
	}
}

// AssistantMessagePromptTemplate 是 AIMessagePromptTemplate 的别名。
func AssistantMessagePromptTemplate(template string) MessagePromptTemplate {
	return AIMessagePromptTemplate(template)
}

// ChatPromptTemplate 是聊天提示词模板。
//
// ChatPromptTemplate 组合多个 MessagePromptTemplate，用于创建完整的对话提示词。
// 实现了 Runnable[map[string]any, []types.Message] 接口。
//
// 示例：
//
//	template := NewChatPromptTemplate(
//	    SystemMessagePromptTemplate("You are a {role}."),
//	    HumanMessagePromptTemplate("Hello, my name is {name}."),
//	    AIMessagePromptTemplate("Nice to meet you!"),
//	)
//
//	messages, _ := template.FormatMessages(map[string]any{
//	    "role": "helpful assistant",
//	    "name": "Alice",
//	})
//
type ChatPromptTemplate struct {
	// Messages 是消息模板列表
	Messages []MessagePromptTemplate

	// InputVariables 是所有输入变量的合集
	InputVariables []string

	// PartialVariables 是部分变量
	PartialVariables map[string]any
}

// NewChatPromptTemplate 创建一个新的 ChatPromptTemplate。
//
// 参数：
//   - messages: 消息模板列表
//
// 返回：
//   - *ChatPromptTemplate: 聊天模板实例
//
func NewChatPromptTemplate(messages ...MessagePromptTemplate) *ChatPromptTemplate {
	// 收集所有输入变量
	varSet := make(map[string]bool)
	for _, msg := range messages {
		for _, v := range msg.GetInputVariables() {
			varSet[v] = true
		}
	}

	inputVars := make([]string, 0, len(varSet))
	for v := range varSet {
		inputVars = append(inputVars, v)
	}

	return &ChatPromptTemplate{
		Messages:         messages,
		InputVariables:   inputVars,
		PartialVariables: make(map[string]any),
	}
}

// FromMessages 从消息列表创建 ChatPromptTemplate。
//
// 这是一个更灵活的构造函数，支持混合使用字符串和已有消息。
//
// 参数：
//   - messages: 消息模板列表或字符串
//
// 返回：
//   - *ChatPromptTemplate: 聊天模板实例
//   - error: 创建错误
//
// 示例：
//
//	template, _ := FromMessages(
//	    []any{
//	        []any{"system", "You are a helpful assistant."},
//	        []any{"human", "Hello, {name}!"},
//	    },
//	)
//
func FromMessages(messages []any) (*ChatPromptTemplate, error) {
	templates := make([]MessagePromptTemplate, 0, len(messages))

	for i, msg := range messages {
		switch m := msg.(type) {
		case MessagePromptTemplate:
			templates = append(templates, m)

		case []any:
			if len(m) != 2 {
				return nil, fmt.Errorf("message at index %d must be [role, template]", i)
			}

			roleStr, ok := m[0].(string)
			if !ok {
				return nil, fmt.Errorf("message role at index %d must be string", i)
			}

			templateStr, ok := m[1].(string)
			if !ok {
				return nil, fmt.Errorf("message template at index %d must be string", i)
			}

			// 根据角色创建对应的模板
			var template MessagePromptTemplate
			switch roleStr {
			case "system":
				template = SystemMessagePromptTemplate(templateStr)
			case "human", "user":
				template = HumanMessagePromptTemplate(templateStr)
			case "ai", "assistant":
				template = AIMessagePromptTemplate(templateStr)
			default:
				return nil, fmt.Errorf("unknown role: %s", roleStr)
			}

			templates = append(templates, template)

		default:
			return nil, fmt.Errorf("invalid message type at index %d", i)
		}
	}

	return NewChatPromptTemplate(templates...), nil
}

// FormatMessages 格式化为消息列表。
//
// 参数：
//   - values: 变量值映射
//
// 返回：
//   - []types.Message: 格式化后的消息列表
//   - error: 格式化错误
//
func (c *ChatPromptTemplate) FormatMessages(values map[string]any) ([]types.Message, error) {
	// 合并部分变量和输入变量
	allValues := make(map[string]any)
	for k, v := range c.PartialVariables {
		allValues[k] = v
	}
	for k, v := range values {
		allValues[k] = v
	}

	// 格式化每个消息
	messages := make([]types.Message, 0, len(c.Messages))
	for i, msgTemplate := range c.Messages {
		msg, err := msgTemplate.Format(allValues)
		if err != nil {
			return nil, fmt.Errorf("failed to format message at index %d: %w", i, err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// FormatPrompt 格式化为 PromptValue。
//
// 参数：
//   - values: 变量值映射
//
// 返回：
//   - PromptValue: 提示词值
//   - error: 格式化错误
//
func (c *ChatPromptTemplate) FormatPrompt(values map[string]any) (PromptValue, error) {
	messages, err := c.FormatMessages(values)
	if err != nil {
		return nil, err
	}
	return &MessagesPromptValue{Messages: messages}, nil
}

// Partial 创建一个新的 ChatPromptTemplate，预填充部分变量。
//
// 参数：
//   - values: 要预填充的变量
//
// 返回：
//   - *ChatPromptTemplate: 新的模板实例
//
func (c *ChatPromptTemplate) Partial(values map[string]any) *ChatPromptTemplate {
	newPartial := make(map[string]any)
	for k, v := range c.PartialVariables {
		newPartial[k] = v
	}
	for k, v := range values {
		newPartial[k] = v
	}

	// 更新 InputVariables
	newInputVars := make([]string, 0)
	for _, varName := range c.InputVariables {
		if _, ok := newPartial[varName]; !ok {
			newInputVars = append(newInputVars, varName)
		}
	}

	return &ChatPromptTemplate{
		Messages:         c.Messages,
		InputVariables:   newInputVars,
		PartialVariables: newPartial,
	}
}

// Invoke 实现 Runnable 接口。
func (c *ChatPromptTemplate) Invoke(ctx context.Context, input map[string]any, opts ...runnable.Option) ([]types.Message, error) {
	return c.FormatMessages(input)
}

// Batch 实现 Runnable 接口。
func (c *ChatPromptTemplate) Batch(ctx context.Context, inputs []map[string]any, opts ...runnable.Option) ([][]types.Message, error) {
	results := make([][]types.Message, len(inputs))
	for i, input := range inputs {
		messages, err := c.FormatMessages(input)
		if err != nil {
			return nil, fmt.Errorf("batch format failed at index %d: %w", i, err)
		}
		results[i] = messages
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (c *ChatPromptTemplate) Stream(ctx context.Context, input map[string]any, opts ...runnable.Option) (<-chan runnable.StreamEvent[[]types.Message], error) {
	out := make(chan runnable.StreamEvent[[]types.Message], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[[]types.Message]{Type: runnable.EventStart}

		messages, err := c.FormatMessages(input)
		if err != nil {
			out <- runnable.StreamEvent[[]types.Message]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[[]types.Message]{Type: runnable.EventStream, Data: messages}
		out <- runnable.StreamEvent[[]types.Message]{Type: runnable.EventEnd, Data: messages}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
func (c *ChatPromptTemplate) GetName() string {
	return "ChatPromptTemplate"
}

// WithConfig 实现 Runnable 接口。
func (c *ChatPromptTemplate) WithConfig(config *types.Config) runnable.Runnable[map[string]any, []types.Message] {
	return c
}

// WithRetry 实现 Runnable 接口。
func (c *ChatPromptTemplate) WithRetry(policy types.RetryPolicy) runnable.Runnable[map[string]any, []types.Message] {
	return runnable.NewRetryRunnable[map[string]any, []types.Message](c, policy)
}

// WithFallbacks 实现 Runnable 接口。
func (c *ChatPromptTemplate) WithFallbacks(fallbacks ...runnable.Runnable[map[string]any, []types.Message]) runnable.Runnable[map[string]any, []types.Message] {
	return runnable.NewFallbackRunnable[map[string]any, []types.Message](c, fallbacks)
}

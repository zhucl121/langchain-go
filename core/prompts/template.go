package prompts

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"langchain-go/core/runnable"
	"langchain-go/pkg/types"
)

// PromptTemplate 是一个简单的字符串模板，支持变量替换。
//
// PromptTemplate 使用 {variable} 语法进行变量替换，实现了
// Runnable[map[string]any, string] 接口。
//
// 示例：
//
//	template, _ := NewPromptTemplate(PromptTemplateConfig{
//	    Template:       "Hello, {name}! You are {age} years old.",
//	    InputVariables: []string{"name", "age"},
//	})
//
//	result, _ := template.Format(map[string]any{
//	    "name": "Alice",
//	    "age":  30,
//	})
//	// result: "Hello, Alice! You are 30 years old."
//
type PromptTemplate struct {
	// Template 是模板字符串，使用 {variable} 语法
	Template string

	// InputVariables 是输入变量列表
	InputVariables []string

	// PartialVariables 是部分变量（预填充的变量）
	PartialVariables map[string]any

	// TemplateFormat 是模板格式（默认为 "f-string"）
	TemplateFormat string

	// ValidateTemplate 是否验证模板
	ValidateTemplate bool
}

// PromptTemplateConfig 是 PromptTemplate 的配置。
type PromptTemplateConfig struct {
	// Template 是模板字符串（必需）
	Template string

	// InputVariables 是输入变量列表（可选，自动检测）
	InputVariables []string

	// PartialVariables 是部分变量（可选）
	PartialVariables map[string]any

	// TemplateFormat 是模板格式（可选，默认 "f-string"）
	TemplateFormat string

	// ValidateTemplate 是否验证模板（可选，默认 true）
	ValidateTemplate bool
}

// NewPromptTemplate 创建一个新的 PromptTemplate。
//
// 参数：
//   - config: 模板配置
//
// 返回：
//   - *PromptTemplate: 模板实例
//   - error: 配置错误或验证错误
//
func NewPromptTemplate(config PromptTemplateConfig) (*PromptTemplate, error) {
	// 设置默认值
	if config.TemplateFormat == "" {
		config.TemplateFormat = "f-string"
	}

	// 如果未指定输入变量，自动检测
	if len(config.InputVariables) == 0 {
		config.InputVariables = extractVariables(config.Template)
	}

	pt := &PromptTemplate{
		Template:         config.Template,
		InputVariables:   config.InputVariables,
		PartialVariables: config.PartialVariables,
		TemplateFormat:   config.TemplateFormat,
		ValidateTemplate: config.ValidateTemplate,
	}

	// 验证模板
	if pt.ValidateTemplate {
		if err := pt.validate(); err != nil {
			return nil, err
		}
	}

	return pt, nil
}

// Format 格式化模板，替换所有变量。
//
// 参数：
//   - values: 变量值映射
//
// 返回：
//   - string: 格式化后的字符串
//   - error: 格式化错误
//
func (pt *PromptTemplate) Format(values map[string]any) (string, error) {
	// 合并部分变量和输入变量
	allValues := make(map[string]any)
	for k, v := range pt.PartialVariables {
		allValues[k] = v
	}
	for k, v := range values {
		allValues[k] = v
	}

	// 检查是否所有必需变量都已提供
	for _, varName := range pt.InputVariables {
		if _, ok := allValues[varName]; !ok {
			return "", fmt.Errorf("missing value for variable: %s", varName)
		}
	}

	// 替换变量
	result := pt.Template
	for key, value := range allValues {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}

	return result, nil
}

// FormatPrompt 格式化为 PromptValue（用于高级场景）。
//
// 参数：
//   - values: 变量值映射
//
// 返回：
//   - PromptValue: 提示词值
//   - error: 格式化错误
//
func (pt *PromptTemplate) FormatPrompt(values map[string]any) (PromptValue, error) {
	text, err := pt.Format(values)
	if err != nil {
		return nil, err
	}
	return &StringPromptValue{Text: text}, nil
}

// Partial 创建一个新的 PromptTemplate，预填充部分变量。
//
// 参数：
//   - values: 要预填充的变量
//
// 返回：
//   - *PromptTemplate: 新的模板实例
//
func (pt *PromptTemplate) Partial(values map[string]any) *PromptTemplate {
	newPartial := make(map[string]any)
	for k, v := range pt.PartialVariables {
		newPartial[k] = v
	}
	for k, v := range values {
		newPartial[k] = v
	}

	// 更新 InputVariables（移除已部分填充的变量）
	newInputVars := make([]string, 0)
	for _, varName := range pt.InputVariables {
		if _, ok := newPartial[varName]; !ok {
			newInputVars = append(newInputVars, varName)
		}
	}

	return &PromptTemplate{
		Template:         pt.Template,
		InputVariables:   newInputVars,
		PartialVariables: newPartial,
		TemplateFormat:   pt.TemplateFormat,
		ValidateTemplate: pt.ValidateTemplate,
	}
}

// Invoke 实现 Runnable 接口。
func (pt *PromptTemplate) Invoke(ctx context.Context, input map[string]any, opts ...runnable.Option) (string, error) {
	return pt.Format(input)
}

// Batch 实现 Runnable 接口。
func (pt *PromptTemplate) Batch(ctx context.Context, inputs []map[string]any, opts ...runnable.Option) ([]string, error) {
	results := make([]string, len(inputs))
	for i, input := range inputs {
		result, err := pt.Format(input)
		if err != nil {
			return nil, fmt.Errorf("batch format failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口（对于 PromptTemplate，流式输出意义不大）。
func (pt *PromptTemplate) Stream(ctx context.Context, input map[string]any, opts ...runnable.Option) (<-chan runnable.StreamEvent[string], error) {
	out := make(chan runnable.StreamEvent[string], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[string]{Type: runnable.EventStart}

		result, err := pt.Format(input)
		if err != nil {
			out <- runnable.StreamEvent[string]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[string]{Type: runnable.EventStream, Data: result}
		out <- runnable.StreamEvent[string]{Type: runnable.EventEnd, Data: result}
	}()

	return out, nil
}

// GetName 实现 Runnable 接口。
func (pt *PromptTemplate) GetName() string {
	return "PromptTemplate"
}

// WithConfig 实现 Runnable 接口。
func (pt *PromptTemplate) WithConfig(config *types.Config) runnable.Runnable[map[string]any, string] {
	// PromptTemplate 不使用配置，返回自身
	return pt
}

// WithRetry 实现 Runnable 接口。
func (pt *PromptTemplate) WithRetry(policy types.RetryPolicy) runnable.Runnable[map[string]any, string] {
	return runnable.NewRetryRunnable[map[string]any, string](pt, policy)
}

// WithFallbacks 实现 Runnable 接口。
func (pt *PromptTemplate) WithFallbacks(fallbacks ...runnable.Runnable[map[string]any, string]) runnable.Runnable[map[string]any, string] {
	return runnable.NewFallbackRunnable[map[string]any, string](pt, fallbacks)
}

// validate 验证模板的有效性。
func (pt *PromptTemplate) validate() error {
	if pt.Template == "" {
		return fmt.Errorf("template cannot be empty")
	}

	// 检测模板中的所有变量
	detectedVars := extractVariables(pt.Template)

	// 检查是否所有检测到的变量都在 InputVariables 或 PartialVariables 中
	declaredVars := make(map[string]bool)
	for _, v := range pt.InputVariables {
		declaredVars[v] = true
	}
	for k := range pt.PartialVariables {
		declaredVars[k] = true
	}

	for _, v := range detectedVars {
		if !declaredVars[v] {
			return fmt.Errorf("variable '%s' found in template but not declared", v)
		}
	}

	return nil
}

// extractVariables 从模板字符串中提取所有变量名。
func extractVariables(template string) []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	vars := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			varName := strings.TrimSpace(match[1])
			if !seen[varName] {
				vars = append(vars, varName)
				seen[varName] = true
			}
		}
	}

	return vars
}

// PromptValue 是提示词值的抽象接口。
//
// PromptValue 可以转换为字符串或消息列表，为不同的使用场景提供灵活性。
type PromptValue interface {
	// ToString 转换为字符串
	ToString() string

	// ToMessages 转换为消息列表
	ToMessages() []types.Message
}

// StringPromptValue 是字符串类型的 PromptValue。
type StringPromptValue struct {
	Text string
}

// ToString 实现 PromptValue 接口。
func (s *StringPromptValue) ToString() string {
	return s.Text
}

// ToMessages 实现 PromptValue 接口。
func (s *StringPromptValue) ToMessages() []types.Message {
	return []types.Message{
		types.NewUserMessage(s.Text),
	}
}

// MessagesPromptValue 是消息列表类型的 PromptValue。
type MessagesPromptValue struct {
	Messages []types.Message
}

// ToString 实现 PromptValue 接口。
func (m *MessagesPromptValue) ToString() string {
	var parts []string
	for _, msg := range m.Messages {
		parts = append(parts, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
	}
	return strings.Join(parts, "\n")
}

// ToMessages 实现 PromptValue 接口。
func (m *MessagesPromptValue) ToMessages() []types.Message {
	return m.Messages
}

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// FewShotPromptTemplate 是 Few-shot 学习的提示词模板。
//
// FewShotPromptTemplate 通过提供示例来引导模型学习任务模式。
// 它组合了前缀、示例和后缀，形成完整的提示词。
//
// 示例：
//
//	// 定义示例
//	examples := []map[string]string{
//	    {"input": "happy", "output": "sad"},
//	    {"input": "tall", "output": "short"},
//	    {"input": "hot", "output": "cold"},
//	}
//
//	// 定义示例格式
//	exampleTemplate := NewPromptTemplate(PromptTemplateConfig{
//	    Template: "Input: {input}\nOutput: {output}",
//	})
//
//	// 创建 Few-shot 模板
//	fewShotPrompt := NewFewShotPromptTemplate(FewShotConfig{
//	    Examples:       examples,
//	    ExamplePrompt:  exampleTemplate,
//	    Prefix:         "Give the antonym of every input.\n",
//	    Suffix:         "\nInput: {input}\nOutput:",
//	    InputVariables: []string{"input"},
//	})
//
//	// 使用
//	result, _ := fewShotPrompt.Format(map[string]any{"input": "big"})
//	// 输出包含所有示例和最终问题
//
type FewShotPromptTemplate struct {
	// Examples 是示例列表
	Examples []map[string]any

	// ExamplePrompt 是示例的格式化模板
	ExamplePrompt *PromptTemplate

	// ExampleSeparator 是示例之间的分隔符（默认 "\n\n"）
	ExampleSeparator string

	// Prefix 是提示词前缀（在所有示例之前）
	Prefix string

	// Suffix 是提示词后缀（在所有示例之后）
	Suffix string

	// InputVariables 是输入变量列表
	InputVariables []string

	// ExampleSelector 是示例选择器（可选，用于动态选择示例）
	ExampleSelector ExampleSelector
}

// FewShotConfig 是 FewShotPromptTemplate 的配置。
type FewShotConfig struct {
	// Examples 是示例列表（必需，如果没有 ExampleSelector）
	Examples []map[string]any

	// ExamplePrompt 是示例格式化模板（必需）
	ExamplePrompt *PromptTemplate

	// ExampleSeparator 是分隔符（可选，默认 "\n\n"）
	ExampleSeparator string

	// Prefix 是前缀（可选）
	Prefix string

	// Suffix 是后缀（可选）
	Suffix string

	// InputVariables 是输入变量（必需）
	InputVariables []string

	// ExampleSelector 是示例选择器（可选）
	ExampleSelector ExampleSelector
}

// NewFewShotPromptTemplate 创建一个新的 FewShotPromptTemplate。
//
// 参数：
//   - config: Few-shot 配置
//
// 返回：
//   - *FewShotPromptTemplate: Few-shot 模板实例
//   - error: 配置错误
//
func NewFewShotPromptTemplate(config FewShotConfig) (*FewShotPromptTemplate, error) {
	// 验证配置
	if config.ExamplePrompt == nil {
		return nil, fmt.Errorf("ExamplePrompt is required")
	}

	if config.Examples == nil && config.ExampleSelector == nil {
		return nil, fmt.Errorf("either Examples or ExampleSelector must be provided")
	}

	if len(config.InputVariables) == 0 {
		return nil, fmt.Errorf("InputVariables cannot be empty")
	}

	// 设置默认值
	if config.ExampleSeparator == "" {
		config.ExampleSeparator = "\n\n"
	}

	return &FewShotPromptTemplate{
		Examples:         config.Examples,
		ExamplePrompt:    config.ExamplePrompt,
		ExampleSeparator: config.ExampleSeparator,
		Prefix:           config.Prefix,
		Suffix:           config.Suffix,
		InputVariables:   config.InputVariables,
		ExampleSelector:  config.ExampleSelector,
	}, nil
}

// Format 格式化 Few-shot 提示词。
//
// 参数：
//   - values: 变量值映射
//
// 返回：
//   - string: 格式化后的提示词
//   - error: 格式化错误
//
func (f *FewShotPromptTemplate) Format(values map[string]any) (string, error) {
	// 获取要使用的示例
	examples := f.Examples
	if f.ExampleSelector != nil {
		var err error
		examples, err = f.ExampleSelector.SelectExamples(values)
		if err != nil {
			return "", fmt.Errorf("failed to select examples: %w", err)
		}
	}

	// 格式化示例
	exampleStrings := make([]string, 0, len(examples))
	for i, example := range examples {
		exampleStr, err := f.ExamplePrompt.Format(example)
		if err != nil {
			return "", fmt.Errorf("failed to format example at index %d: %w", i, err)
		}
		exampleStrings = append(exampleStrings, exampleStr)
	}

	// 组合前缀、示例和后缀
	var parts []string

	if f.Prefix != "" {
		parts = append(parts, f.Prefix)
	}

	if len(exampleStrings) > 0 {
		parts = append(parts, strings.Join(exampleStrings, f.ExampleSeparator))
	}

	if f.Suffix != "" {
		// 格式化后缀（可能包含变量）
		suffixTemplate, err := NewPromptTemplate(PromptTemplateConfig{
			Template: f.Suffix,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create suffix template: %w", err)
		}

		suffix, err := suffixTemplate.Format(values)
		if err != nil {
			return "", fmt.Errorf("failed to format suffix: %w", err)
		}

		parts = append(parts, suffix)
	}

	return strings.Join(parts, ""), nil
}

// FormatPrompt 格式化为 PromptValue。
func (f *FewShotPromptTemplate) FormatPrompt(values map[string]any) (PromptValue, error) {
	text, err := f.Format(values)
	if err != nil {
		return nil, err
	}
	return &StringPromptValue{Text: text}, nil
}

// Invoke 实现 Runnable 接口。
func (f *FewShotPromptTemplate) Invoke(ctx context.Context, input map[string]any, opts ...runnable.Option) (string, error) {
	return f.Format(input)
}

// Batch 实现 Runnable 接口。
func (f *FewShotPromptTemplate) Batch(ctx context.Context, inputs []map[string]any, opts ...runnable.Option) ([]string, error) {
	results := make([]string, len(inputs))
	for i, input := range inputs {
		result, err := f.Format(input)
		if err != nil {
			return nil, fmt.Errorf("batch format failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (f *FewShotPromptTemplate) Stream(ctx context.Context, input map[string]any, opts ...runnable.Option) (<-chan runnable.StreamEvent[string], error) {
	out := make(chan runnable.StreamEvent[string], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[string]{Type: runnable.EventStart}

		result, err := f.Format(input)
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
func (f *FewShotPromptTemplate) GetName() string {
	return "FewShotPromptTemplate"
}

// WithConfig 实现 Runnable 接口。
func (f *FewShotPromptTemplate) WithConfig(config *types.Config) runnable.Runnable[map[string]any, string] {
	return f
}

// WithRetry 实现 Runnable 接口。
func (f *FewShotPromptTemplate) WithRetry(policy types.RetryPolicy) runnable.Runnable[map[string]any, string] {
	return runnable.NewRetryRunnable[map[string]any, string](f, policy)
}

// WithFallbacks 实现 Runnable 接口。
func (f *FewShotPromptTemplate) WithFallbacks(fallbacks ...runnable.Runnable[map[string]any, string]) runnable.Runnable[map[string]any, string] {
	return runnable.NewFallbackRunnable[map[string]any, string](f, fallbacks)
}

// ExampleSelector 是示例选择器接口。
//
// ExampleSelector 根据输入动态选择最相关的示例，实现智能的 Few-shot 学习。
type ExampleSelector interface {
	// SelectExamples 根据输入选择示例
	SelectExamples(input map[string]any) ([]map[string]any, error)

	// AddExample 添加新示例
	AddExample(example map[string]any) error
}

// LengthBasedExampleSelector 基于长度选择示例。
//
// 这个选择器会选择示例，直到达到最大长度限制。
type LengthBasedExampleSelector struct {
	Examples       []map[string]any
	ExamplePrompt  *PromptTemplate
	MaxLength      int
	GetTextLength  func(string) int
}

// NewLengthBasedExampleSelector 创建一个新的长度基础选择器。
func NewLengthBasedExampleSelector(examples []map[string]any, examplePrompt *PromptTemplate, maxLength int) *LengthBasedExampleSelector {
	return &LengthBasedExampleSelector{
		Examples:      examples,
		ExamplePrompt: examplePrompt,
		MaxLength:     maxLength,
		GetTextLength: func(text string) int {
			return len(text)
		},
	}
}

// SelectExamples 实现 ExampleSelector 接口。
func (l *LengthBasedExampleSelector) SelectExamples(input map[string]any) ([]map[string]any, error) {
	selected := make([]map[string]any, 0)
	currentLength := 0

	for _, example := range l.Examples {
		// 格式化示例以计算长度
		exampleStr, err := l.ExamplePrompt.Format(example)
		if err != nil {
			continue
		}

		exampleLength := l.GetTextLength(exampleStr)

		// 检查是否超过最大长度
		if currentLength+exampleLength <= l.MaxLength {
			selected = append(selected, example)
			currentLength += exampleLength
		} else {
			break
		}
	}

	return selected, nil
}

// AddExample 实现 ExampleSelector 接口。
func (l *LengthBasedExampleSelector) AddExample(example map[string]any) error {
	l.Examples = append(l.Examples, example)
	return nil
}

// MaxMarginalRelevanceExampleSelector 基于最大边际相关性选择示例。
//
// 这个选择器选择与输入相关但彼此不同的示例，以提供多样性。
// 注意：这需要向量嵌入功能，这里提供接口定义，实际实现需要 embeddings 模块。
type MaxMarginalRelevanceExampleSelector struct {
	Examples      []map[string]any
	Embeddings    interface{} // 将在 embeddings 模块实现后替换
	K             int         // 选择的示例数量
	FetchK        int         // 候选示例数量
}

// SelectExamples 实现 ExampleSelector 接口。
func (m *MaxMarginalRelevanceExampleSelector) SelectExamples(input map[string]any) ([]map[string]any, error) {
	// 注意：需要 embeddings 模块支持
	// 这里提供接口，实际实现将在后续模块中完成
	return m.Examples[:min(m.K, len(m.Examples))], nil
}

// AddExample 实现 ExampleSelector 接口。
func (m *MaxMarginalRelevanceExampleSelector) AddExample(example map[string]any) error {
	m.Examples = append(m.Examples, example)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

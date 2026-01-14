package output

import (
	"context"
	"fmt"

	"langchain-go/core/runnable"
	"langchain-go/pkg/types"
)

// OutputParser 是输出解析器的核心接口。
//
// OutputParser 从 LLM 的文本输出中提取结构化数据。
// 实现了 Runnable[string, T] 接口，可以与其他组件链式组合。
//
// 类型参数：
//   - T: 解析后的输出类型
//
// 使用场景：
//   - 从 LLM 输出中提取 JSON
//   - 解析列表、键值对等结构化数据
//   - 验证和转换输出格式
//
type OutputParser[T any] interface {
	// 继承 Runnable 接口
	runnable.Runnable[string, T]

	// Parse 解析文本输出为结构化数据
	//
	// 参数：
	//   - text: LLM 的文本输出
	//
	// 返回：
	//   - T: 解析后的结构化数据
	//   - error: 解析错误
	//
	Parse(text string) (T, error)

	// ParseWithPrompt 使用原始提示词解析（用于错误恢复）
	//
	// 某些解析器可能需要原始提示词来修复解析错误。
	//
	// 参数：
	//   - text: LLM 的文本输出
	//   - prompt: 原始提示词
	//
	// 返回：
	//   - T: 解析后的结构化数据
	//   - error: 解析错误
	//
	ParseWithPrompt(text string, prompt string) (T, error)

	// GetFormatInstructions 获取格式指令
	//
	// 返回可以插入提示词的格式说明，告诉 LLM 如何格式化输出。
	//
	// 返回：
	//   - string: 格式指令文本
	//
	GetFormatInstructions() string

	// GetType 获取输出类型描述
	//
	// 返回：
	//   - string: 类型描述（如 "json", "list", "string"）
	//
	GetType() string
}

// BaseOutputParser 提供 OutputParser 的基础实现。
//
// BaseOutputParser 实现了 Runnable 接口的通用方法，
// 具体的解析器只需实现 Parse 方法。
type BaseOutputParser[T any] struct {
	name             string
	formatInstructions string
	parserType       string
}

// NewBaseOutputParser 创建基础解析器。
//
// 参数：
//   - name: 解析器名称
//   - formatInstructions: 格式指令
//   - parserType: 解析器类型
//
// 返回：
//   - *BaseOutputParser[T]: 基础解析器实例
//
func NewBaseOutputParser[T any](name, formatInstructions, parserType string) *BaseOutputParser[T] {
	return &BaseOutputParser[T]{
		name:               name,
		formatInstructions: formatInstructions,
		parserType:         parserType,
	}
}

// GetFormatInstructions 实现 OutputParser 接口。
func (b *BaseOutputParser[T]) GetFormatInstructions() string {
	return b.formatInstructions
}

// GetType 实现 OutputParser 接口。
func (b *BaseOutputParser[T]) GetType() string {
	return b.parserType
}

// GetName 实现 Runnable 接口。
func (b *BaseOutputParser[T]) GetName() string {
	return b.name
}

// Invoke 实现 Runnable 接口。
func (b *BaseOutputParser[T]) Invoke(ctx context.Context, input string, opts ...runnable.Option) (T, error) {
	// 注意：子类必须实现自己的 Invoke 方法，这里只是占位
	var zero T
	return zero, fmt.Errorf("Invoke not implemented in BaseOutputParser")
}

// Batch 实现 Runnable 接口的默认批量处理。
func (b *BaseOutputParser[T]) Batch(ctx context.Context, inputs []string, opts ...runnable.Option) ([]T, error) {
	// 注意：这个方法会被子类继承，但需要子类实现 Parse 方法
	var zero T
	return []T{zero}, fmt.Errorf("Batch not implemented in BaseOutputParser")
}

// Stream 实现 Runnable 接口的默认流式处理。
func (b *BaseOutputParser[T]) Stream(ctx context.Context, input string, opts ...runnable.Option) (<-chan runnable.StreamEvent[T], error) {
	out := make(chan runnable.StreamEvent[T], 1)
	close(out)
	return out, fmt.Errorf("Stream not implemented in BaseOutputParser")
}

// WithConfig 实现 Runnable 接口。
func (b *BaseOutputParser[T]) WithConfig(config *types.Config) runnable.Runnable[string, T] {
	return b
}

// WithRetry 实现 Runnable 接口。
func (b *BaseOutputParser[T]) WithRetry(policy types.RetryPolicy) runnable.Runnable[string, T] {
	return runnable.NewRetryRunnable[string, T](b, policy)
}

// WithFallbacks 实现 Runnable 接口。
func (b *BaseOutputParser[T]) WithFallbacks(fallbacks ...runnable.Runnable[string, T]) runnable.Runnable[string, T] {
	return runnable.NewFallbackRunnable[string, T](b, fallbacks)
}

// StringOutputParser 是最简单的输出解析器，直接返回字符串。
//
// StringOutputParser 不做任何转换，将 LLM 输出原样返回。
type StringOutputParser struct {
	*BaseOutputParser[string]
}

// NewStringOutputParser 创建字符串输出解析器。
//
// 返回：
//   - *StringOutputParser: 字符串解析器实例
//
func NewStringOutputParser() *StringOutputParser {
	return &StringOutputParser{
		BaseOutputParser: NewBaseOutputParser[string](
			"StringOutputParser",
			"The output should be a plain text string.",
			"string",
		),
	}
}

// Parse 实现 OutputParser 接口。
func (s *StringOutputParser) Parse(text string) (string, error) {
	return text, nil
}

// ParseWithPrompt 实现 OutputParser 接口。
func (s *StringOutputParser) ParseWithPrompt(text string, prompt string) (string, error) {
	return text, nil
}

// Invoke 实现 Runnable 接口。
func (s *StringOutputParser) Invoke(ctx context.Context, input string, opts ...runnable.Option) (string, error) {
	return s.Parse(input)
}

// Batch 实现 Runnable 接口。
func (s *StringOutputParser) Batch(ctx context.Context, inputs []string, opts ...runnable.Option) ([]string, error) {
	results := make([]string, len(inputs))
	for i, input := range inputs {
		result, err := s.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("batch parse failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (s *StringOutputParser) Stream(ctx context.Context, input string, opts ...runnable.Option) (<-chan runnable.StreamEvent[string], error) {
	out := make(chan runnable.StreamEvent[string], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[string]{Type: runnable.EventStart}

		result, err := s.Parse(input)
		if err != nil {
			out <- runnable.StreamEvent[string]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[string]{Type: runnable.EventStream, Data: result}
		out <- runnable.StreamEvent[string]{Type: runnable.EventEnd, Data: result}
	}()

	return out, nil
}

// ParseError 表示解析错误，包含原始输出和错误信息。
type ParseError struct {
	// Output 是原始 LLM 输出
	Output string

	// Err 是底层错误
	Err error

	// Message 是错误描述
	Message string
}

// Error 实现 error 接口。
func (e *ParseError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return fmt.Sprintf("parse error: %v", e.Err)
	}
	return "parse error"
}

// Unwrap 实现错误链。
func (e *ParseError) Unwrap() error {
	return e.Err
}

// NewParseError 创建解析错误。
func NewParseError(output string, err error, message string) *ParseError {
	return &ParseError{
		Output:  output,
		Err:     err,
		Message: message,
	}
}

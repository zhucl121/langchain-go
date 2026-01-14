package output

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"langchain-go/core/runnable"
)

// JSONParser 是 JSON 输出解析器。
//
// JSONParser 从 LLM 输出中提取 JSON 数据。它可以：
//   - 解析纯 JSON 字符串
//   - 从 Markdown 代码块中提取 JSON
//   - 从混合文本中提取 JSON
//
// 示例：
//
//	parser := NewJSONParser()
//
//	// 解析纯 JSON
//	result, _ := parser.Parse(`{"name": "Alice", "age": 30}`)
//
//	// 解析 Markdown 中的 JSON
//	result, _ := parser.Parse("```json\n{\"name\": \"Bob\"}\n```")
//
type JSONParser struct {
	*BaseOutputParser[map[string]any]
}

// NewJSONParser 创建 JSON 解析器。
//
// 返回：
//   - *JSONParser: JSON 解析器实例
//
func NewJSONParser() *JSONParser {
	instructions := `The output should be formatted as a JSON object.
Example format:
{"key1": "value1", "key2": "value2"}

Ensure the output is valid JSON that can be parsed.`

	return &JSONParser{
		BaseOutputParser: NewBaseOutputParser[map[string]any](
			"JSONParser",
			instructions,
			"json",
		),
	}
}

// Parse 实现 OutputParser 接口。
func (j *JSONParser) Parse(text string) (map[string]any, error) {
	// 1. 尝试直接解析
	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result, nil
	}

	// 2. 尝试从 Markdown 代码块中提取
	extracted := extractJSONFromMarkdown(text)
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &result); err == nil {
			return result, nil
		}
	}

	// 3. 尝试从文本中查找 JSON 对象
	extracted = extractJSONFromText(text)
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &result); err == nil {
			return result, nil
		}
	}

	// 4. 所有尝试都失败
	return nil, NewParseError(text, nil, "failed to parse JSON from output")
}

// ParseWithPrompt 实现 OutputParser 接口。
func (j *JSONParser) ParseWithPrompt(text string, prompt string) (map[string]any, error) {
	// 对于 JSONParser，提示词不影响解析
	return j.Parse(text)
}

// Invoke 实现 Runnable 接口。
func (j *JSONParser) Invoke(ctx context.Context, input string, opts ...runnable.Option) (map[string]any, error) {
	return j.Parse(input)
}

// Batch 实现 Runnable 接口。
func (j *JSONParser) Batch(ctx context.Context, inputs []string, opts ...runnable.Option) ([]map[string]any, error) {
	results := make([]map[string]any, len(inputs))
	for i, input := range inputs {
		result, err := j.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("batch parse failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (j *JSONParser) Stream(ctx context.Context, input string, opts ...runnable.Option) (<-chan runnable.StreamEvent[map[string]any], error) {
	out := make(chan runnable.StreamEvent[map[string]any], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[map[string]any]{Type: runnable.EventStart}

		result, err := j.Parse(input)
		if err != nil {
			out <- runnable.StreamEvent[map[string]any]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[map[string]any]{Type: runnable.EventStream, Data: result}
		out <- runnable.StreamEvent[map[string]any]{Type: runnable.EventEnd, Data: result}
	}()

	return out, nil
}

// JSONArrayParser 是 JSON 数组解析器。
//
// JSONArrayParser 专门用于解析 JSON 数组。
type JSONArrayParser struct {
	*BaseOutputParser[[]any]
}

// NewJSONArrayParser 创建 JSON 数组解析器。
//
// 返回：
//   - *JSONArrayParser: JSON 数组解析器实例
//
func NewJSONArrayParser() *JSONArrayParser {
	instructions := `The output should be formatted as a JSON array.
Example format:
["item1", "item2", "item3"]

Ensure the output is a valid JSON array that can be parsed.`

	return &JSONArrayParser{
		BaseOutputParser: NewBaseOutputParser[[]any](
			"JSONArrayParser",
			instructions,
			"json_array",
		),
	}
}

// Parse 实现 OutputParser 接口。
func (j *JSONArrayParser) Parse(text string) ([]any, error) {
	// 1. 尝试直接解析
	var result []any
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result, nil
	}

	// 2. 尝试从 Markdown 代码块中提取
	extracted := extractJSONFromMarkdown(text)
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &result); err == nil {
			return result, nil
		}
	}

	// 3. 尝试从文本中查找 JSON 数组
	extracted = extractJSONArrayFromText(text)
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &result); err == nil {
			return result, nil
		}
	}

	// 4. 所有尝试都失败
	return nil, NewParseError(text, nil, "failed to parse JSON array from output")
}

// ParseWithPrompt 实现 OutputParser 接口。
func (j *JSONArrayParser) ParseWithPrompt(text string, prompt string) ([]any, error) {
	return j.Parse(text)
}

// Invoke 实现 Runnable 接口。
func (j *JSONArrayParser) Invoke(ctx context.Context, input string, opts ...runnable.Option) ([]any, error) {
	return j.Parse(input)
}

// Batch 实现 Runnable 接口。
func (j *JSONArrayParser) Batch(ctx context.Context, inputs []string, opts ...runnable.Option) ([][]any, error) {
	results := make([][]any, len(inputs))
	for i, input := range inputs {
		result, err := j.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("batch parse failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (j *JSONArrayParser) Stream(ctx context.Context, input string, opts ...runnable.Option) (<-chan runnable.StreamEvent[[]any], error) {
	out := make(chan runnable.StreamEvent[[]any], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[[]any]{Type: runnable.EventStart}

		result, err := j.Parse(input)
		if err != nil {
			out <- runnable.StreamEvent[[]any]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[[]any]{Type: runnable.EventStream, Data: result}
		out <- runnable.StreamEvent[[]any]{Type: runnable.EventEnd, Data: result}
	}()

	return out, nil
}

// ListParser 是列表解析器，解析分隔的文本为列表。
//
// ListParser 将文本按分隔符分割为字符串列表。
//
// 示例：
//
//	parser := NewListParser(",")
//	result, _ := parser.Parse("apple, banana, orange")
//	// []string{"apple", "banana", "orange"}
//
type ListParser struct {
	*BaseOutputParser[[]string]
	separator string
	trimSpace bool
}

// NewListParser 创建列表解析器。
//
// 参数：
//   - separator: 分隔符（如 ","、"\n"）
//
// 返回：
//   - *ListParser: 列表解析器实例
//
func NewListParser(separator string) *ListParser {
	instructions := fmt.Sprintf(`The output should be a list of items separated by '%s'.
Example format:
item1%sitem2%sitem3

Each item will be trimmed of leading/trailing whitespace.`, separator, separator, separator)

	return &ListParser{
		BaseOutputParser: NewBaseOutputParser[[]string](
			"ListParser",
			instructions,
			"list",
		),
		separator: separator,
		trimSpace: true,
	}
}

// Parse 实现 OutputParser 接口。
func (l *ListParser) Parse(text string) ([]string, error) {
	if text == "" {
		return []string{}, nil
	}

	parts := strings.Split(text, l.separator)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if l.trimSpace {
			part = strings.TrimSpace(part)
		}
		if part != "" {
			result = append(result, part)
		}
	}

	return result, nil
}

// ParseWithPrompt 实现 OutputParser 接口。
func (l *ListParser) ParseWithPrompt(text string, prompt string) ([]string, error) {
	return l.Parse(text)
}

// Invoke 实现 Runnable 接口。
func (l *ListParser) Invoke(ctx context.Context, input string, opts ...runnable.Option) ([]string, error) {
	return l.Parse(input)
}

// Batch 实现 Runnable 接口。
func (l *ListParser) Batch(ctx context.Context, inputs []string, opts ...runnable.Option) ([][]string, error) {
	results := make([][]string, len(inputs))
	for i, input := range inputs {
		result, err := l.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("batch parse failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (l *ListParser) Stream(ctx context.Context, input string, opts ...runnable.Option) (<-chan runnable.StreamEvent[[]string], error) {
	out := make(chan runnable.StreamEvent[[]string], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[[]string]{Type: runnable.EventStart}

		result, err := l.Parse(input)
		if err != nil {
			out <- runnable.StreamEvent[[]string]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[[]string]{Type: runnable.EventStream, Data: result}
		out <- runnable.StreamEvent[[]string]{Type: runnable.EventEnd, Data: result}
	}()

	return out, nil
}

// 辅助函数：从 Markdown 代码块中提取 JSON
func extractJSONFromMarkdown(text string) string {
	// 匹配 ```json ... ``` 或 ``` ... ```
	patterns := []string{
		"```json\\s*\\n(.+?)\\n```",
		"```\\s*\\n(.+?)\\n```",
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile("(?s)" + pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

// 辅助函数：从文本中提取 JSON 对象
func extractJSONFromText(text string) string {
	// 查找第一个 { 和最后一个 }
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	// 从后往前找 }
	end := strings.LastIndex(text, "}")
	if end == -1 || end <= start {
		return ""
	}

	return text[start : end+1]
}

// 辅助函数：从文本中提取 JSON 数组
func extractJSONArrayFromText(text string) string {
	// 查找第一个 [ 和最后一个 ]
	start := strings.Index(text, "[")
	if start == -1 {
		return ""
	}

	// 从后往前找 ]
	end := strings.LastIndex(text, "]")
	if end == -1 || end <= start {
		return ""
	}

	return text[start : end+1]
}

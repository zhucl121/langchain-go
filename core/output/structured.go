package output

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"langchain-go/core/runnable"
	"langchain-go/pkg/types"
)

// StructuredParser 是类型安全的结构化输出解析器。
//
// StructuredParser 将 LLM 输出解析为指定的 Go 结构体类型。
// 它提供编译时类型安全，避免运行时类型断言。
//
// 类型参数：
//   - T: 目标结构体类型
//
// 示例：
//
//	type Person struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	    City string `json:"city"`
//	}
//
//	parser := NewStructuredParser[Person]()
//
//	llmOutput := `{"name": "Alice", "age": 30, "city": "NYC"}`
//	person, err := parser.Parse(llmOutput)
//	if err != nil {
//	    panic(err)
//	}
//
//	fmt.Println(person.Name) // "Alice"
//	fmt.Println(person.Age)  // 30
//
type StructuredParser[T any] struct {
	*BaseOutputParser[T]
	schema *types.Schema
}

// NewStructuredParser 创建结构化解析器。
//
// 解析器会自动从类型 T 生成 JSON Schema，并提供格式指令。
//
// 返回：
//   - *StructuredParser[T]: 结构化解析器实例
//
func NewStructuredParser[T any]() *StructuredParser[T] {
	// 生成 Schema（基于类型 T）
	schema := generateSchemaFromType[T]()

	// 生成格式指令
	instructions := generateFormatInstructions(schema)

	return &StructuredParser[T]{
		BaseOutputParser: NewBaseOutputParser[T](
			"StructuredParser",
			instructions,
			"structured",
		),
		schema: schema,
	}
}

// NewStructuredParserWithSchema 使用自定义 Schema 创建结构化解析器。
//
// 参数：
//   - schema: 自定义的 JSON Schema
//
// 返回：
//   - *StructuredParser[T]: 结构化解析器实例
//
func NewStructuredParserWithSchema[T any](schema *types.Schema) *StructuredParser[T] {
	instructions := generateFormatInstructions(schema)

	return &StructuredParser[T]{
		BaseOutputParser: NewBaseOutputParser[T](
			"StructuredParser",
			instructions,
			"structured",
		),
		schema: schema,
	}
}

// Parse 实现 OutputParser 接口。
func (s *StructuredParser[T]) Parse(text string) (T, error) {
	var zero T

	// 1. 尝试直接解析
	var result T
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
	return zero, NewParseError(text, nil, "failed to parse structured output")
}

// ParseWithPrompt 实现 OutputParser 接口。
func (s *StructuredParser[T]) ParseWithPrompt(text string, prompt string) (T, error) {
	return s.Parse(text)
}

// GetSchema 获取 JSON Schema。
//
// 返回：
//   - *types.Schema: JSON Schema
//
func (s *StructuredParser[T]) GetSchema() *types.Schema {
	return s.schema
}

// Invoke 实现 Runnable 接口。
func (s *StructuredParser[T]) Invoke(ctx context.Context, input string, opts ...runnable.Option) (T, error) {
	return s.Parse(input)
}

// Batch 实现 Runnable 接口。
func (s *StructuredParser[T]) Batch(ctx context.Context, inputs []string, opts ...runnable.Option) ([]T, error) {
	results := make([]T, len(inputs))
	for i, input := range inputs {
		result, err := s.Parse(input)
		if err != nil {
			var zero []T
			return zero, fmt.Errorf("batch parse failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (s *StructuredParser[T]) Stream(ctx context.Context, input string, opts ...runnable.Option) (<-chan runnable.StreamEvent[T], error) {
	out := make(chan runnable.StreamEvent[T], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[T]{Type: runnable.EventStart}

		result, err := s.Parse(input)
		if err != nil {
			out <- runnable.StreamEvent[T]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[T]{Type: runnable.EventStream, Data: result}
		out <- runnable.StreamEvent[T]{Type: runnable.EventEnd, Data: result}
	}()

	return out, nil
}

// generateSchemaFromType 从 Go 类型生成 JSON Schema（增强版本）
//
// 支持特性：
//   - 递归处理嵌套结构体
//   - 数组和切片的元素类型
//   - description、enum、minimum、maximum 等 tag
//   - 复杂类型映射
//
func generateSchemaFromType[T any]() *types.Schema {
	var zero T
	t := reflect.TypeOf(zero)
	return generateSchemaFromReflectType(t)
}

// generateSchemaFromReflectType 从 reflect.Type 生成 Schema（递归）
func generateSchemaFromReflectType(t reflect.Type) *types.Schema {
	// 如果是指针，获取元素类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &types.Schema{}

	switch t.Kind() {
	case reflect.Struct:
		schema.Type = "object"
		properties := make(map[string]types.Schema)
		required := make([]string, 0)

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// 跳过未导出的字段
			if !field.IsExported() {
				continue
			}

			// 获取 json tag
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			// 解析 json tag
			fieldName, omitempty := parseJSONTag(jsonTag, field.Name)

			// 递归生成字段 schema
			fieldSchema := generateSchemaFromReflectType(field.Type)

			// 从 tag 获取描述
			if desc := field.Tag.Get("description"); desc != "" {
				fieldSchema.Description = desc
			}

			// 从 tag 获取枚举值
			if enum := field.Tag.Get("enum"); enum != "" {
				fieldSchema.Enum = parseEnumTag(enum)
			}

			// 从 tag 获取验证规则
			if min := field.Tag.Get("minimum"); min != "" {
				fieldSchema.Minimum = parseFloatPtr(min)
			}
			if max := field.Tag.Get("maximum"); max != "" {
				fieldSchema.Maximum = parseFloatPtr(max)
			}
			if pattern := field.Tag.Get("pattern"); pattern != "" {
				fieldSchema.Pattern = pattern
			}

			properties[fieldName] = *fieldSchema

			// 检查是否必需（非指针、非 omitempty）
			if field.Type.Kind() != reflect.Ptr && !omitempty {
				required = append(required, fieldName)
			}
		}

		schema.Properties = properties
		if len(required) > 0 {
			schema.Required = required
		}

	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		// 递归处理元素类型
		elemType := t.Elem()
		schema.Items = generateSchemaFromReflectType(elemType)

	case reflect.Map:
		schema.Type = "object"
		// Map 的值类型作为 additionalProperties
		// 注意：types.Schema.AdditionalProperties 类型需要确认
		// 简化处理：只设置类型为 object

	default:
		schema.Type = getJSONType(t)
	}

	return schema
}

// parseJSONTag 解析 JSON tag
func parseJSONTag(tag, fieldName string) (name string, omitempty bool) {
	if tag == "" {
		return fieldName, false
	}

	parts := splitTag(tag, ',')
	name = parts[0]
	if name == "" {
		name = fieldName
	}

	// 检查 omitempty
	for i := 1; i < len(parts); i++ {
		if parts[i] == "omitempty" {
			omitempty = true
			break
		}
	}

	return name, omitempty
}

// splitTag 分割 tag 字符串
func splitTag(s string, sep rune) []string {
	var parts []string
	var current string

	for _, c := range s {
		if c == sep {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// parseEnumTag 解析枚举 tag
func parseEnumTag(tag string) []any {
	parts := splitTag(tag, '|')
	result := make([]any, len(parts))
	for i, part := range parts {
		result[i] = part
	}
	return result
}

// parseFloatPtr 解析浮点数指针
func parseFloatPtr(s string) *float64 {
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return &f
	}
	return nil
}

// getJSONType 获取 Go 类型对应的 JSON 类型
func getJSONType(t reflect.Type) string {
	// 如果是指针，获取元素类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string"
	}
}

// generateFormatInstructions 从 Schema 生成格式指令
func generateFormatInstructions(schema *types.Schema) string {
	if schema == nil {
		return "The output should be formatted as valid JSON."
	}

	// 将 Schema 序列化为 JSON
	schemaJSON, err := json.MarshalIndent(schema.ToMap(), "", "  ")
	if err != nil {
		return "The output should be formatted as valid JSON."
	}

	return fmt.Sprintf(`The output should be formatted as a JSON object that conforms to the following schema:

%s

Ensure the output is valid JSON and matches the schema exactly.`, string(schemaJSON))
}

// BooleanParser 是布尔值解析器。
//
// BooleanParser 将文本解析为布尔值。
// 它能识别多种布尔值表示：true/false, yes/no, 1/0 等。
type BooleanParser struct {
	*BaseOutputParser[bool]
}

// NewBooleanParser 创建布尔值解析器。
//
// 返回：
//   - *BooleanParser: 布尔值解析器实例
//
func NewBooleanParser() *BooleanParser {
	instructions := `The output should be a boolean value.
Acceptable values:
- true, True, TRUE, yes, Yes, YES, 1
- false, False, FALSE, no, No, NO, 0`

	return &BooleanParser{
		BaseOutputParser: NewBaseOutputParser[bool](
			"BooleanParser",
			instructions,
			"boolean",
		),
	}
}

// Parse 实现 OutputParser 接口。
func (b *BooleanParser) Parse(text string) (bool, error) {
	text = trimAndLower(text)

	switch text {
	case "true", "yes", "1", "t", "y":
		return true, nil
	case "false", "no", "0", "f", "n":
		return false, nil
	default:
		return false, NewParseError(text, nil, fmt.Sprintf("cannot parse '%s' as boolean", text))
	}
}

// ParseWithPrompt 实现 OutputParser 接口。
func (b *BooleanParser) ParseWithPrompt(text string, prompt string) (bool, error) {
	return b.Parse(text)
}

// Invoke 实现 Runnable 接口。
func (b *BooleanParser) Invoke(ctx context.Context, input string, opts ...runnable.Option) (bool, error) {
	return b.Parse(input)
}

// Batch 实现 Runnable 接口。
func (b *BooleanParser) Batch(ctx context.Context, inputs []string, opts ...runnable.Option) ([]bool, error) {
	results := make([]bool, len(inputs))
	for i, input := range inputs {
		result, err := b.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("batch parse failed at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Stream 实现 Runnable 接口。
func (b *BooleanParser) Stream(ctx context.Context, input string, opts ...runnable.Option) (<-chan runnable.StreamEvent[bool], error) {
	out := make(chan runnable.StreamEvent[bool], 1)

	go func() {
		defer close(out)

		out <- runnable.StreamEvent[bool]{Type: runnable.EventStart}

		result, err := b.Parse(input)
		if err != nil {
			out <- runnable.StreamEvent[bool]{Type: runnable.EventError, Error: err}
			return
		}

		out <- runnable.StreamEvent[bool]{Type: runnable.EventStream, Data: result}
		out <- runnable.StreamEvent[bool]{Type: runnable.EventEnd, Data: result}
	}()

	return out, nil
}

// 辅助函数：trim 和 lowercase
func trimAndLower(s string) string {
	// Trim whitespace
	trimmed := ""
	for _, c := range s {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			trimmed += string(c)
		}
	}
	
	// Convert to lowercase
	result := ""
	for _, c := range trimmed {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

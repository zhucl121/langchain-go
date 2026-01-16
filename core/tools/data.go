package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"langchain-go/pkg/types"
)

// JSONParseTool JSON 解析工具。
//
// 功能：解析 JSON 字符串为对象
type JSONParseTool struct {
	name        string
	description string
}

// NewJSONParseTool 创建 JSON 解析工具。
//
// 返回：
//   - *JSONParseTool: JSON 解析工具实例
//
// 示例：
//
//	tool := tools.NewJSONParseTool()
//	result, _ := tool.Execute(ctx, map[string]any{
//	    "json_string": `{"name": "John", "age": 30}`,
//	})
//
func NewJSONParseTool() *JSONParseTool {
	return &JSONParseTool{
		name:        "json_parse",
		description: "Parse a JSON string into an object. Returns the parsed object or an error if the JSON is invalid.",
	}
}

// GetName 实现 Tool 接口。
func (t *JSONParseTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *JSONParseTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *JSONParseTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"json_string": {
				Type:        "string",
				Description: "The JSON string to parse",
			},
		},
		Required: []string{"json_string"},
	}
}

// Execute 实现 Tool 接口。
func (t *JSONParseTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	jsonStr, ok := args["json_string"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'json_string' must be a string", ErrInvalidArguments)
	}

	var result any
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse JSON: %v", ErrExecutionFailed, err)
	}

	return result, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *JSONParseTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// JSONStringifyTool JSON 序列化工具。
//
// 功能：将对象序列化为 JSON 字符串
type JSONStringifyTool struct {
	name        string
	description string
}

// NewJSONStringifyTool 创建 JSON 序列化工具。
//
// 返回：
//   - *JSONStringifyTool: JSON 序列化工具实例
//
// 示例：
//
//	tool := tools.NewJSONStringifyTool()
//	result, _ := tool.Execute(ctx, map[string]any{
//	    "object": map[string]any{"name": "John", "age": 30},
//	    "pretty": true,
//	})
//
func NewJSONStringifyTool() *JSONStringifyTool {
	return &JSONStringifyTool{
		name:        "json_stringify",
		description: "Convert an object to a JSON string. Supports pretty printing.",
	}
}

// GetName 实现 Tool 接口。
func (t *JSONStringifyTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *JSONStringifyTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *JSONStringifyTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"object": {
				Type:        "object",
				Description: "The object to stringify",
			},
			"pretty": {
				Type:        "boolean",
				Description: "Whether to format the JSON with indentation (default: false)",
			},
		},
		Required: []string{"object"},
	}
}

// Execute 实现 Tool 接口。
func (t *JSONStringifyTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	obj, ok := args["object"]
	if !ok {
		return nil, fmt.Errorf("%w: 'object' is required", ErrInvalidArguments)
	}

	pretty := false
	if p, ok := args["pretty"].(bool); ok {
		pretty = p
	}

	var jsonBytes []byte
	var err error

	if pretty {
		jsonBytes, err = json.MarshalIndent(obj, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(obj)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: failed to stringify object: %v", ErrExecutionFailed, err)
	}

	return string(jsonBytes), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *JSONStringifyTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// JSONExtractTool JSON 提取工具。
//
// 功能：从 JSON 对象中提取指定路径的值
type JSONExtractTool struct {
	name        string
	description string
}

// NewJSONExtractTool 创建 JSON 提取工具。
//
// 返回：
//   - *JSONExtractTool: JSON 提取工具实例
//
// 示例：
//
//	tool := tools.NewJSONExtractTool()
//	result, _ := tool.Execute(ctx, map[string]any{
//	    "json_string": `{"user": {"name": "John", "age": 30}}`,
//	    "path": "user.name",
//	})
//	// result: "John"
//
func NewJSONExtractTool() *JSONExtractTool {
	return &JSONExtractTool{
		name:        "json_extract",
		description: "Extract a value from a JSON object using a dot-separated path (e.g., 'user.name').",
	}
}

// GetName 实现 Tool 接口。
func (t *JSONExtractTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *JSONExtractTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *JSONExtractTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"json_string": {
				Type:        "string",
				Description: "The JSON string to extract from",
			},
			"path": {
				Type:        "string",
				Description: "The dot-separated path to extract (e.g., 'user.name')",
			},
		},
		Required: []string{"json_string", "path"},
	}
}

// Execute 实现 Tool 接口。
func (t *JSONExtractTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	jsonStr, ok := args["json_string"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'json_string' must be a string", ErrInvalidArguments)
	}

	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'path' must be a string", ErrInvalidArguments)
	}

	// 解析 JSON
	var data any
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse JSON: %v", ErrExecutionFailed, err)
	}

	// 提取路径
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			var exists bool
			current, exists = v[part]
			if !exists {
				return nil, fmt.Errorf("%w: path not found: %s", ErrExecutionFailed, path)
			}
		default:
			return nil, fmt.Errorf("%w: cannot traverse non-object at path: %s", ErrExecutionFailed, part)
		}
	}

	return current, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *JSONExtractTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// StringLengthTool 字符串长度工具。
//
// 功能：获取字符串的长度
type StringLengthTool struct {
	name        string
	description string
}

// NewStringLengthTool 创建字符串长度工具。
//
// 返回：
//   - *StringLengthTool: 字符串长度工具实例
//
func NewStringLengthTool() *StringLengthTool {
	return &StringLengthTool{
		name:        "string_length",
		description: "Get the length of a string.",
	}
}

// GetName 实现 Tool 接口。
func (t *StringLengthTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *StringLengthTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *StringLengthTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"text": {
				Type:        "string",
				Description: "The string to get length of",
			},
		},
		Required: []string{"text"},
	}
}

// Execute 实现 Tool 接口。
func (t *StringLengthTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'text' must be a string", ErrInvalidArguments)
	}

	return len(text), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *StringLengthTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// StringSplitTool 字符串分割工具。
//
// 功能：按分隔符分割字符串
type StringSplitTool struct {
	name        string
	description string
}

// NewStringSplitTool 创建字符串分割工具。
//
// 返回：
//   - *StringSplitTool: 字符串分割工具实例
//
func NewStringSplitTool() *StringSplitTool {
	return &StringSplitTool{
		name:        "string_split",
		description: "Split a string by a delimiter.",
	}
}

// GetName 实现 Tool 接口。
func (t *StringSplitTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *StringSplitTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *StringSplitTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"text": {
				Type:        "string",
				Description: "The string to split",
			},
			"delimiter": {
				Type:        "string",
				Description: "The delimiter to split by",
			},
		},
		Required: []string{"text", "delimiter"},
	}
}

// Execute 实现 Tool 接口。
func (t *StringSplitTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'text' must be a string", ErrInvalidArguments)
	}

	delimiter, ok := args["delimiter"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'delimiter' must be a string", ErrInvalidArguments)
	}

	return strings.Split(text, delimiter), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *StringSplitTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// StringJoinTool 字符串连接工具。
//
// 功能：使用分隔符连接字符串数组
type StringJoinTool struct {
	name        string
	description string
}

// NewStringJoinTool 创建字符串连接工具。
//
// 返回：
//   - *StringJoinTool: 字符串连接工具实例
//
func NewStringJoinTool() *StringJoinTool {
	return &StringJoinTool{
		name:        "string_join",
		description: "Join an array of strings with a delimiter.",
	}
}

// GetName 实现 Tool 接口。
func (t *StringJoinTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *StringJoinTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *StringJoinTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"strings": {
				Type:        "array",
				Description: "The array of strings to join",
			},
			"delimiter": {
				Type:        "string",
				Description: "The delimiter to join with",
			},
		},
		Required: []string{"strings", "delimiter"},
	}
}

// Execute 实现 Tool 接口。
func (t *StringJoinTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	stringsAny, ok := args["strings"].([]any)
	if !ok {
		return nil, fmt.Errorf("%w: 'strings' must be an array", ErrInvalidArguments)
	}

	delimiter, ok := args["delimiter"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'delimiter' must be a string", ErrInvalidArguments)
	}

	// 转换为字符串数组
	strings := make([]string, len(stringsAny))
	for i, v := range stringsAny {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%w: all elements must be strings", ErrInvalidArguments)
		}
		strings[i] = str
	}

	return strings.Join(strings, delimiter), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *StringJoinTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

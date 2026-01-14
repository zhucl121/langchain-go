package types

import (
	"encoding/json"
	"fmt"
)

// Schema 表示 JSON Schema，用于描述数据结构。
//
// Schema 主要用于工具参数的定义和验证。
//
// 示例：
//
//	schema := types.Schema{
//	    Type: "object",
//	    Properties: map[string]types.Schema{
//	        "name": {
//	            Type:        "string",
//	            Description: "User name",
//	        },
//	        "age": {
//	            Type:        "integer",
//	            Description: "User age",
//	            Minimum:     ptr(0.0),
//	        },
//	    },
//	    Required: []string{"name"},
//	}
//
type Schema struct {
	// Type 数据类型: "string", "number", "integer", "boolean", "array", "object", "null"
	Type string `json:"type,omitempty"`

	// Description 字段描述
	Description string `json:"description,omitempty"`

	// Properties 对象属性（type 为 "object" 时使用）
	Properties map[string]Schema `json:"properties,omitempty"`

	// Required 必需字段列表（type 为 "object" 时使用）
	Required []string `json:"required,omitempty"`

	// Items 数组元素 Schema（type 为 "array" 时使用）
	Items *Schema `json:"items,omitempty"`

	// Enum 枚举值列表
	Enum []any `json:"enum,omitempty"`

	// Default 默认值
	Default any `json:"default,omitempty"`

	// Minimum 最小值（type 为 "number" 或 "integer" 时使用）
	Minimum *float64 `json:"minimum,omitempty"`

	// Maximum 最大值（type 为 "number" 或 "integer" 时使用）
	Maximum *float64 `json:"maximum,omitempty"`

	// MinLength 最小长度（type 为 "string" 时使用）
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength 最大长度（type 为 "string" 时使用）
	MaxLength *int `json:"maxLength,omitempty"`

	// Pattern 正则表达式模式（type 为 "string" 时使用）
	Pattern string `json:"pattern,omitempty"`

	// Format 格式（如 "date", "time", "email", "uri" 等）
	Format string `json:"format,omitempty"`

	// MinItems 最小元素数（type 为 "array" 时使用）
	MinItems *int `json:"minItems,omitempty"`

	// MaxItems 最大元素数（type 为 "array" 时使用）
	MaxItems *int `json:"maxItems,omitempty"`

	// AdditionalProperties 是否允许额外属性（type 为 "object" 时使用）
	AdditionalProperties *bool `json:"additionalProperties,omitempty"`
}

// Validate 验证 Schema 的有效性。
//
// 返回：
//   - error: 验证失败时返回错误
//
func (s Schema) Validate() error {
	// 检查类型
	if s.Type != "" {
		validTypes := map[string]bool{
			"string":  true,
			"number":  true,
			"integer": true,
			"boolean": true,
			"array":   true,
			"object":  true,
			"null":    true,
		}
		if !validTypes[s.Type] {
			return fmt.Errorf("invalid type: %s", s.Type)
		}
	}

	// 验证对象属性
	if s.Type == "object" && len(s.Properties) > 0 {
		for name, propSchema := range s.Properties {
			if err := propSchema.Validate(); err != nil {
				return fmt.Errorf("invalid property %s: %w", name, err)
			}
		}
	}

	// 验证数组元素
	if s.Type == "array" && s.Items != nil {
		if err := s.Items.Validate(); err != nil {
			return fmt.Errorf("invalid items schema: %w", err)
		}
	}

	// 验证数值范围
	if s.Minimum != nil && s.Maximum != nil && *s.Minimum > *s.Maximum {
		return fmt.Errorf("minimum (%f) cannot be greater than maximum (%f)", *s.Minimum, *s.Maximum)
	}

	// 验证字符串长度
	if s.MinLength != nil && s.MaxLength != nil && *s.MinLength > *s.MaxLength {
		return fmt.Errorf("minLength (%d) cannot be greater than maxLength (%d)", *s.MinLength, *s.MaxLength)
	}

	// 验证数组长度
	if s.MinItems != nil && s.MaxItems != nil && *s.MinItems > *s.MaxItems {
		return fmt.Errorf("minItems (%d) cannot be greater than maxItems (%d)", *s.MinItems, *s.MaxItems)
	}

	return nil
}

// ToMap 将 Schema 转换为 map，用于序列化。
//
// 返回：
//   - map[string]any: Schema 的 map 表示
//
func (s Schema) ToMap() map[string]any {
	data, _ := json.Marshal(s)
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
}

// Clone 创建 Schema 的深拷贝。
//
// 返回：
//   - Schema: Schema 副本
//
func (s Schema) Clone() Schema {
	clone := s

	// 深拷贝 Properties
	if s.Properties != nil {
		clone.Properties = make(map[string]Schema, len(s.Properties))
		for k, v := range s.Properties {
			clone.Properties[k] = v.Clone()
		}
	}

	// 深拷贝 Required
	if s.Required != nil {
		clone.Required = make([]string, len(s.Required))
		copy(clone.Required, s.Required)
	}

	// 深拷贝 Items
	if s.Items != nil {
		itemsClone := s.Items.Clone()
		clone.Items = &itemsClone
	}

	// 深拷贝 Enum
	if s.Enum != nil {
		clone.Enum = make([]any, len(s.Enum))
		copy(clone.Enum, s.Enum)
	}

	return clone
}

// NewStringSchema 创建字符串类型的 Schema。
//
// 参数：
//   - description: 字段描述
//
// 返回：
//   - Schema: 字符串 Schema
//
func NewStringSchema(description string) Schema {
	return Schema{
		Type:        "string",
		Description: description,
	}
}

// NewIntegerSchema 创建整数类型的 Schema。
//
// 参数：
//   - description: 字段描述
//
// 返回：
//   - Schema: 整数 Schema
//
func NewIntegerSchema(description string) Schema {
	return Schema{
		Type:        "integer",
		Description: description,
	}
}

// NewNumberSchema 创建数值类型的 Schema。
//
// 参数：
//   - description: 字段描述
//
// 返回：
//   - Schema: 数值 Schema
//
func NewNumberSchema(description string) Schema {
	return Schema{
		Type:        "number",
		Description: description,
	}
}

// NewBooleanSchema 创建布尔类型的 Schema。
//
// 参数：
//   - description: 字段描述
//
// 返回：
//   - Schema: 布尔 Schema
//
func NewBooleanSchema(description string) Schema {
	return Schema{
		Type:        "boolean",
		Description: description,
	}
}

// NewArraySchema 创建数组类型的 Schema。
//
// 参数：
//   - description: 字段描述
//   - items: 数组元素 Schema
//
// 返回：
//   - Schema: 数组 Schema
//
func NewArraySchema(description string, items Schema) Schema {
	return Schema{
		Type:        "array",
		Description: description,
		Items:       &items,
	}
}

// NewObjectSchema 创建对象类型的 Schema。
//
// 参数：
//   - description: 字段描述
//   - properties: 对象属性
//   - required: 必需字段列表
//
// 返回：
//   - Schema: 对象 Schema
//
func NewObjectSchema(description string, properties map[string]Schema, required []string) Schema {
	return Schema{
		Type:        "object",
		Description: description,
		Properties:  properties,
		Required:    required,
	}
}

// WithEnum 设置枚举值。
//
// 参数：
//   - values: 枚举值列表
//
// 返回：
//   - Schema: 新的 Schema 实例
//
func (s Schema) WithEnum(values ...any) Schema {
	s.Enum = values
	return s
}

// WithDefault 设置默认值。
//
// 参数：
//   - value: 默认值
//
// 返回：
//   - Schema: 新的 Schema 实例
//
func (s Schema) WithDefault(value any) Schema {
	s.Default = value
	return s
}

// WithMinMax 设置最小值和最大值（数值类型）。
//
// 参数：
//   - min: 最小值
//   - max: 最大值
//
// 返回：
//   - Schema: 新的 Schema 实例
//
func (s Schema) WithMinMax(min, max float64) Schema {
	s.Minimum = &min
	s.Maximum = &max
	return s
}

// WithLengthRange 设置长度范围（字符串或数组类型）。
//
// 参数：
//   - min: 最小长度
//   - max: 最大长度
//
// 返回：
//   - Schema: 新的 Schema 实例
//
func (s Schema) WithLengthRange(min, max int) Schema {
	if s.Type == "string" {
		s.MinLength = &min
		s.MaxLength = &max
	} else if s.Type == "array" {
		s.MinItems = &min
		s.MaxItems = &max
	}
	return s
}

// WithPattern 设置正则表达式模式（字符串类型）。
//
// 参数：
//   - pattern: 正则表达式
//
// 返回：
//   - Schema: 新的 Schema 实例
//
func (s Schema) WithPattern(pattern string) Schema {
	s.Pattern = pattern
	return s
}

// WithFormat 设置格式。
//
// 参数：
//   - format: 格式名称（如 "date", "email", "uri" 等）
//
// 返回：
//   - Schema: 新的 Schema 实例
//
func (s Schema) WithFormat(format string) Schema {
	s.Format = format
	return s
}

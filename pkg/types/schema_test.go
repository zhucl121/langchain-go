package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_Validate(t *testing.T) {
	t.Run("valid schemas", func(t *testing.T) {
		tests := []struct {
			name   string
			schema Schema
		}{
			{
				"string",
				Schema{Type: "string"},
			},
			{
				"number with range",
				Schema{Type: "number", Minimum: ptr(0.0), Maximum: ptr(100.0)},
			},
			{
				"object with properties",
				Schema{
					Type: "object",
					Properties: map[string]Schema{
						"name": {Type: "string"},
						"age":  {Type: "integer"},
					},
					Required: []string{"name"},
				},
			},
			{
				"array",
				Schema{
					Type:  "array",
					Items: &Schema{Type: "string"},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.schema.Validate()
				assert.NoError(t, err)
			})
		}
	})

	t.Run("invalid schemas", func(t *testing.T) {
		tests := []struct {
			name   string
			schema Schema
			errMsg string
		}{
			{
				"invalid type",
				Schema{Type: "invalid_type"},
				"invalid type",
			},
			{
				"min > max",
				Schema{Type: "number", Minimum: ptr(10.0), Maximum: ptr(5.0)},
				"cannot be greater than maximum",
			},
			{
				"minLength > maxLength",
				Schema{Type: "string", MinLength: ptr(10), MaxLength: ptr(5)},
				"cannot be greater than maxLength",
			},
			{
				"minItems > maxItems",
				Schema{Type: "array", MinItems: ptr(10), MaxItems: ptr(5)},
				"cannot be greater than maxItems",
			},
			{
				"invalid property schema",
				Schema{
					Type: "object",
					Properties: map[string]Schema{
						"bad": {Type: "invalid"},
					},
				},
				"invalid property",
			},
			{
				"invalid items schema",
				Schema{
					Type:  "array",
					Items: &Schema{Type: "invalid"},
				},
				"invalid items schema",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.schema.Validate()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})
}

func TestSchema_ToMap(t *testing.T) {
	schema := Schema{
		Type:        "object",
		Description: "Test schema",
		Properties: map[string]Schema{
			"name": {Type: "string"},
		},
		Required: []string{"name"},
	}

	m := schema.ToMap()

	assert.Equal(t, "object", m["type"])
	assert.Equal(t, "Test schema", m["description"])
	assert.NotNil(t, m["properties"])
	assert.NotNil(t, m["required"])
}

func TestSchema_Clone(t *testing.T) {
	original := Schema{
		Type: "object",
		Properties: map[string]Schema{
			"name": {Type: "string"},
		},
		Required: []string{"name"},
		Items:    &Schema{Type: "string"},
		Enum:     []any{"a", "b"},
	}

	clone := original.Clone()

	// 验证值相等
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, len(original.Properties), len(clone.Properties))
	assert.Equal(t, len(original.Required), len(clone.Required))

	// 验证是深拷贝
	clone.Properties["name"] = Schema{Type: "number"}
	assert.Equal(t, "string", original.Properties["name"].Type)

	clone.Required[0] = "modified"
	assert.Equal(t, "name", original.Required[0])

	clone.Enum[0] = "x"
	assert.Equal(t, "a", original.Enum[0])
}

func TestNewStringSchema(t *testing.T) {
	schema := NewStringSchema("A string field")

	assert.Equal(t, "string", schema.Type)
	assert.Equal(t, "A string field", schema.Description)
}

func TestNewIntegerSchema(t *testing.T) {
	schema := NewIntegerSchema("An integer field")

	assert.Equal(t, "integer", schema.Type)
	assert.Equal(t, "An integer field", schema.Description)
}

func TestNewNumberSchema(t *testing.T) {
	schema := NewNumberSchema("A number field")

	assert.Equal(t, "number", schema.Type)
	assert.Equal(t, "A number field", schema.Description)
}

func TestNewBooleanSchema(t *testing.T) {
	schema := NewBooleanSchema("A boolean field")

	assert.Equal(t, "boolean", schema.Type)
	assert.Equal(t, "A boolean field", schema.Description)
}

func TestNewArraySchema(t *testing.T) {
	itemSchema := Schema{Type: "string"}
	schema := NewArraySchema("An array field", itemSchema)

	assert.Equal(t, "array", schema.Type)
	assert.Equal(t, "An array field", schema.Description)
	assert.NotNil(t, schema.Items)
	assert.Equal(t, "string", schema.Items.Type)
}

func TestNewObjectSchema(t *testing.T) {
	properties := map[string]Schema{
		"name": {Type: "string"},
		"age":  {Type: "integer"},
	}
	required := []string{"name"}

	schema := NewObjectSchema("An object field", properties, required)

	assert.Equal(t, "object", schema.Type)
	assert.Equal(t, "An object field", schema.Description)
	assert.Equal(t, 2, len(schema.Properties))
	assert.Equal(t, []string{"name"}, schema.Required)
}

func TestSchema_WithEnum(t *testing.T) {
	schema := NewStringSchema("Color").WithEnum("red", "green", "blue")

	assert.Equal(t, []any{"red", "green", "blue"}, schema.Enum)
}

func TestSchema_WithDefault(t *testing.T) {
	schema := NewIntegerSchema("Count").WithDefault(10)

	assert.Equal(t, 10, schema.Default)
}

func TestSchema_WithMinMax(t *testing.T) {
	schema := NewNumberSchema("Score").WithMinMax(0, 100)

	require.NotNil(t, schema.Minimum)
	require.NotNil(t, schema.Maximum)
	assert.Equal(t, 0.0, *schema.Minimum)
	assert.Equal(t, 100.0, *schema.Maximum)
}

func TestSchema_WithLengthRange(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		schema := NewStringSchema("Text").WithLengthRange(1, 100)

		require.NotNil(t, schema.MinLength)
		require.NotNil(t, schema.MaxLength)
		assert.Equal(t, 1, *schema.MinLength)
		assert.Equal(t, 100, *schema.MaxLength)
	})

	t.Run("array", func(t *testing.T) {
		schema := NewArraySchema("Items", Schema{Type: "string"}).WithLengthRange(1, 10)

		require.NotNil(t, schema.MinItems)
		require.NotNil(t, schema.MaxItems)
		assert.Equal(t, 1, *schema.MinItems)
		assert.Equal(t, 10, *schema.MaxItems)
	})
}

func TestSchema_WithPattern(t *testing.T) {
	schema := NewStringSchema("Email").WithPattern(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`)

	assert.Equal(t, `^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`, schema.Pattern)
}

func TestSchema_WithFormat(t *testing.T) {
	schema := NewStringSchema("Birthday").WithFormat("date")

	assert.Equal(t, "date", schema.Format)
}

func TestSchema_ComplexExample(t *testing.T) {
	// 构建一个复杂的 Schema
	schema := NewObjectSchema(
		"User",
		map[string]Schema{
			"name": NewStringSchema("User name").
				WithLengthRange(1, 50),
			"email": NewStringSchema("Email address").
				WithFormat("email").
				WithPattern(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`),
			"age": NewIntegerSchema("User age").
				WithMinMax(0, 120),
			"role": NewStringSchema("User role").
				WithEnum("admin", "user", "guest").
				WithDefault("user"),
			"tags": NewArraySchema("User tags", NewStringSchema("Tag")).
				WithLengthRange(0, 10),
		},
		[]string{"name", "email"},
	)

	// 验证
	err := schema.Validate()
	assert.NoError(t, err)

	// 转换为 map
	m := schema.ToMap()
	assert.Equal(t, "object", m["type"])
	assert.NotNil(t, m["properties"])
	assert.Equal(t, []any{"name", "email"}, m["required"])
}

func TestSchema_JSONSerialization(t *testing.T) {
	original := Schema{
		Type:        "object",
		Description: "Test",
		Properties: map[string]Schema{
			"name": {Type: "string", MinLength: ptr(1)},
		},
		Required: []string{"name"},
	}

	// 序列化
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded Schema
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Description, decoded.Description)
	assert.Equal(t, len(original.Properties), len(decoded.Properties))
	assert.Equal(t, original.Required, decoded.Required)
}

// 基准测试
func BenchmarkSchema_Validate(b *testing.B) {
	schema := NewObjectSchema(
		"Test",
		map[string]Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		[]string{"name"},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schema.Validate()
	}
}

func BenchmarkSchema_Clone(b *testing.B) {
	schema := NewObjectSchema(
		"Test",
		map[string]Schema{
			"name":  {Type: "string"},
			"email": {Type: "string"},
			"age":   {Type: "integer"},
		},
		[]string{"name"},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schema.Clone()
	}
}

func BenchmarkSchema_ToMap(b *testing.B) {
	schema := NewObjectSchema(
		"Test",
		map[string]Schema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		[]string{"name"},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schema.ToMap()
	}
}

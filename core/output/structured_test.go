package output

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试用的结构体
type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	City string `json:"city,omitempty"`
}

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func TestNewStructuredParser(t *testing.T) {
	parser := NewStructuredParser[Person]()
	assert.NotNil(t, parser)
	assert.Equal(t, "StructuredParser", parser.GetName())
	assert.Equal(t, "structured", parser.GetType())
	assert.NotNil(t, parser.GetSchema())
}

func TestStructuredParser_Parse(t *testing.T) {
	parser := NewStructuredParser[Person]()

	tests := []struct {
		name      string
		input     string
		expectErr bool
		validate  func(t *testing.T, result Person)
	}{
		{
			name:      "simple JSON",
			input:     `{"name": "Alice", "age": 30, "city": "NYC"}`,
			expectErr: false,
			validate: func(t *testing.T, result Person) {
				assert.Equal(t, "Alice", result.Name)
				assert.Equal(t, 30, result.Age)
				assert.Equal(t, "NYC", result.City)
			},
		},
		{
			name:      "JSON without optional field",
			input:     `{"name": "Bob", "age": 25}`,
			expectErr: false,
			validate: func(t *testing.T, result Person) {
				assert.Equal(t, "Bob", result.Name)
				assert.Equal(t, 25, result.Age)
				assert.Equal(t, "", result.City)
			},
		},
		{
			name:      "JSON in markdown",
			input:     "```json\n{\"name\": \"Charlie\", \"age\": 35}\n```",
			expectErr: false,
			validate: func(t *testing.T, result Person) {
				assert.Equal(t, "Charlie", result.Name)
				assert.Equal(t, 35, result.Age)
			},
		},
		{
			name:      "JSON in mixed text",
			input:     "Here is the person: {\"name\": \"David\", \"age\": 40} done",
			expectErr: false,
			validate: func(t *testing.T, result Person) {
				assert.Equal(t, "David", result.Name)
				assert.Equal(t, 40, result.Age)
			},
		},
		{
			name:      "invalid JSON",
			input:     "not valid json",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestStructuredParser_DifferentTypes(t *testing.T) {
	parser := NewStructuredParser[Product]()

	input := `{"id": 123, "name": "Laptop", "price": 999.99}`
	result, err := parser.Parse(input)

	require.NoError(t, err)
	assert.Equal(t, 123, result.ID)
	assert.Equal(t, "Laptop", result.Name)
	assert.Equal(t, 999.99, result.Price)
}

func TestStructuredParser_Invoke(t *testing.T) {
	parser := NewStructuredParser[Person]()

	result, err := parser.Invoke(nil, `{"name": "Eve", "age": 28}`)
	require.NoError(t, err)
	assert.Equal(t, "Eve", result.Name)
	assert.Equal(t, 28, result.Age)
}

func TestStructuredParser_Batch(t *testing.T) {
	parser := NewStructuredParser[Person]()

	inputs := []string{
		`{"name": "Alice", "age": 30}`,
		`{"name": "Bob", "age": 25}`,
	}

	results, err := parser.Batch(nil, inputs)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Alice", results[0].Name)
	assert.Equal(t, "Bob", results[1].Name)
}

func TestNewBooleanParser(t *testing.T) {
	parser := NewBooleanParser()
	assert.NotNil(t, parser)
	assert.Equal(t, "BooleanParser", parser.GetName())
	assert.Equal(t, "boolean", parser.GetType())
}

func TestBooleanParser_Parse(t *testing.T) {
	parser := NewBooleanParser()

	tests := []struct {
		name      string
		input     string
		expected  bool
		expectErr bool
	}{
		// True values
		{name: "true", input: "true", expected: true, expectErr: false},
		{name: "True", input: "True", expected: true, expectErr: false},
		{name: "TRUE", input: "TRUE", expected: true, expectErr: false},
		{name: "yes", input: "yes", expected: true, expectErr: false},
		{name: "Yes", input: "Yes", expected: true, expectErr: false},
		{name: "1", input: "1", expected: true, expectErr: false},
		{name: "t", input: "t", expected: true, expectErr: false},
		{name: "y", input: "y", expected: true, expectErr: false},

		// False values
		{name: "false", input: "false", expected: false, expectErr: false},
		{name: "False", input: "False", expected: false, expectErr: false},
		{name: "FALSE", input: "FALSE", expected: false, expectErr: false},
		{name: "no", input: "no", expected: false, expectErr: false},
		{name: "No", input: "No", expected: false, expectErr: false},
		{name: "0", input: "0", expected: false, expectErr: false},
		{name: "f", input: "f", expected: false, expectErr: false},
		{name: "n", input: "n", expected: false, expectErr: false},

		// Invalid values
		{name: "invalid", input: "maybe", expected: false, expectErr: true},
		{name: "empty", input: "", expected: false, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBooleanParser_WithWhitespace(t *testing.T) {
	parser := NewBooleanParser()

	tests := []string{
		" true ",
		"\ntrue\n",
		"\ttrue\t",
		"  yes  ",
	}

	for _, input := range tests {
		result, err := parser.Parse(input)
		require.NoError(t, err, "input: %q", input)
		assert.True(t, result, "input: %q", input)
	}
}

func TestBooleanParser_Invoke(t *testing.T) {
	parser := NewBooleanParser()

	result, err := parser.Invoke(nil, "yes")
	require.NoError(t, err)
	assert.True(t, result)
}

func TestGenerateSchemaFromType(t *testing.T) {
	schema := generateSchemaFromType[Person]()

	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.NotNil(t, schema.Properties)

	// 检查字段
	nameSchema, ok := schema.Properties["name"]
	assert.True(t, ok)
	assert.Equal(t, "string", nameSchema.Type)

	ageSchema, ok := schema.Properties["age"]
	assert.True(t, ok)
	assert.Equal(t, "integer", ageSchema.Type)
}

func TestGetJSONType(t *testing.T) {
	tests := []struct {
		name     string
		goType   any
		expected string
	}{
		{name: "string", goType: "", expected: "string"},
		{name: "int", goType: 0, expected: "integer"},
		{name: "float", goType: 0.0, expected: "number"},
		{name: "bool", goType: false, expected: "boolean"},
		{name: "slice", goType: []string{}, expected: "array"},
		{name: "map", goType: map[string]any{}, expected: "object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getJSONType(reflect.TypeOf(tt.goType))
			assert.Equal(t, tt.expected, result)
		})
	}
}

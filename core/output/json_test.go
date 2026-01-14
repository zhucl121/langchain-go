package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONParser(t *testing.T) {
	parser := NewJSONParser()
	assert.NotNil(t, parser)
	assert.Equal(t, "JSONParser", parser.GetName())
	assert.Equal(t, "json", parser.GetType())
	assert.NotEmpty(t, parser.GetFormatInstructions())
}

func TestJSONParser_Parse(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name      string
		input     string
		expectErr bool
		validate  func(t *testing.T, result map[string]any)
	}{
		{
			name:      "simple JSON",
			input:     `{"name": "Alice", "age": 30}`,
			expectErr: false,
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "Alice", result["name"])
				assert.Equal(t, float64(30), result["age"])
			},
		},
		{
			name:      "JSON in markdown",
			input:     "```json\n{\"name\": \"Bob\", \"age\": 25}\n```",
			expectErr: false,
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "Bob", result["name"])
				assert.Equal(t, float64(25), result["age"])
			},
		},
		{
			name:      "JSON in plain markdown block",
			input:     "```\n{\"name\": \"Charlie\"}\n```",
			expectErr: false,
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "Charlie", result["name"])
			},
		},
		{
			name:      "JSON in mixed text",
			input:     "Here is the data: {\"name\": \"David\"} and more text",
			expectErr: false,
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "David", result["name"])
			},
		},
		{
			name:      "invalid JSON",
			input:     "not json at all",
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

func TestJSONParser_Invoke(t *testing.T) {
	parser := NewJSONParser()

	result, err := parser.Invoke(nil, `{"key": "value"}`)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

func TestJSONParser_Batch(t *testing.T) {
	parser := NewJSONParser()

	inputs := []string{
		`{"name": "Alice"}`,
		`{"name": "Bob"}`,
		`{"name": "Charlie"}`,
	}

	results, err := parser.Batch(nil, inputs)
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "Alice", results[0]["name"])
	assert.Equal(t, "Bob", results[1]["name"])
	assert.Equal(t, "Charlie", results[2]["name"])
}

func TestNewJSONArrayParser(t *testing.T) {
	parser := NewJSONArrayParser()
	assert.NotNil(t, parser)
	assert.Equal(t, "JSONArrayParser", parser.GetName())
	assert.Equal(t, "json_array", parser.GetType())
}

func TestJSONArrayParser_Parse(t *testing.T) {
	parser := NewJSONArrayParser()

	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  []any
	}{
		{
			name:      "simple array",
			input:     `["apple", "banana", "orange"]`,
			expectErr: false,
			expected:  []any{"apple", "banana", "orange"},
		},
		{
			name:      "array in markdown",
			input:     "```json\n[1, 2, 3]\n```",
			expectErr: false,
			expected:  []any{float64(1), float64(2), float64(3)},
		},
		{
			name:      "array in mixed text",
			input:     "The numbers are: [4, 5, 6] here",
			expectErr: false,
			expected:  []any{float64(4), float64(5), float64(6)},
		},
		{
			name:      "invalid array",
			input:     "not an array",
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
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNewListParser(t *testing.T) {
	parser := NewListParser(",")
	assert.NotNil(t, parser)
	assert.Equal(t, "ListParser", parser.GetName())
	assert.Equal(t, "list", parser.GetType())
}

func TestListParser_Parse(t *testing.T) {
	tests := []struct {
		name      string
		separator string
		input     string
		expected  []string
	}{
		{
			name:      "comma separated",
			separator: ",",
			input:     "apple, banana, orange",
			expected:  []string{"apple", "banana", "orange"},
		},
		{
			name:      "newline separated",
			separator: "\n",
			input:     "line1\nline2\nline3",
			expected:  []string{"line1", "line2", "line3"},
		},
		{
			name:      "empty string",
			separator: ",",
			input:     "",
			expected:  []string{},
		},
		{
			name:      "single item",
			separator: ",",
			input:     "only one",
			expected:  []string{"only one"},
		},
		{
			name:      "with extra spaces",
			separator: ",",
			input:     "  item1  ,  item2  ,  item3  ",
			expected:  []string{"item1", "item2", "item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewListParser(tt.separator)
			result, err := parser.Parse(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListParser_Invoke(t *testing.T) {
	parser := NewListParser(",")

	result, err := parser.Invoke(nil, "a, b, c")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "plain code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "no code block",
			input:    "{\"key\": \"value\"}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJSONFromText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON in text",
			input:    "Here is some data: {\"key\": \"value\"} and more",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "no JSON",
			input:    "No JSON here",
			expected: "",
		},
		{
			name:     "multiple braces",
			input:    "First { then {\"key\": \"value\"} last }",
			expected: "{ then {\"key\": \"value\"} last }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

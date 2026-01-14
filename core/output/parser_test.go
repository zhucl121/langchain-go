package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStringOutputParser(t *testing.T) {
	parser := NewStringOutputParser()
	assert.NotNil(t, parser)
	assert.Equal(t, "StringOutputParser", parser.GetName())
	assert.Equal(t, "string", parser.GetType())
}

func TestStringOutputParser_Parse(t *testing.T) {
	parser := NewStringOutputParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multi-line text",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringOutputParser_Invoke(t *testing.T) {
	parser := NewStringOutputParser()

	result, err := parser.Invoke(nil, "test input")
	require.NoError(t, err)
	assert.Equal(t, "test input", result)
}

func TestStringOutputParser_Batch(t *testing.T) {
	parser := NewStringOutputParser()

	inputs := []string{"input1", "input2", "input3"}
	results, err := parser.Batch(nil, inputs)

	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, []string{"input1", "input2", "input3"}, results)
}

func TestParseError(t *testing.T) {
	err := NewParseError("test output", nil, "test error")

	assert.Equal(t, "test error", err.Error())
	assert.Equal(t, "test output", err.Output)
}

func TestBaseOutputParser(t *testing.T) {
	base := NewBaseOutputParser[string]("TestParser", "Test instructions", "test")

	assert.Equal(t, "TestParser", base.GetName())
	assert.Equal(t, "Test instructions", base.GetFormatInstructions())
	assert.Equal(t, "test", base.GetType())
}

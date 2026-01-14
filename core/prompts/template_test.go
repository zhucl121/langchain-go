package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPromptTemplate(t *testing.T) {
	tests := []struct {
		name      string
		config    PromptTemplateConfig
		expectErr bool
	}{
		{
			name: "valid template",
			config: PromptTemplateConfig{
				Template:       "Hello, {name}!",
				InputVariables: []string{"name"},
			},
			expectErr: false,
		},
		{
			name: "auto-detect variables",
			config: PromptTemplateConfig{
				Template: "Hello, {name}! You are {age} years old.",
			},
			expectErr: false,
		},
		{
			name: "template with partial variables",
			config: PromptTemplateConfig{
				Template: "Hello, {name}! Today is {day}.",
				PartialVariables: map[string]any{
					"day": "Monday",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewPromptTemplate(tt.config)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, template)
			}
		})
	}
}

func TestPromptTemplate_Format(t *testing.T) {
	tests := []struct {
		name      string
		template  *PromptTemplate
		values    map[string]any
		expected  string
		expectErr bool
	}{
		{
			name: "simple replacement",
			template: &PromptTemplate{
				Template:       "Hello, {name}!",
				InputVariables: []string{"name"},
			},
			values: map[string]any{
				"name": "Alice",
			},
			expected:  "Hello, Alice!",
			expectErr: false,
		},
		{
			name: "multiple variables",
			template: &PromptTemplate{
				Template:       "{greeting}, {name}! You are {age} years old.",
				InputVariables: []string{"greeting", "name", "age"},
			},
			values: map[string]any{
				"greeting": "Hello",
				"name":     "Bob",
				"age":      30,
			},
			expected:  "Hello, Bob! You are 30 years old.",
			expectErr: false,
		},
		{
			name: "with partial variables",
			template: &PromptTemplate{
				Template:       "Hello, {name}! Today is {day}.",
				InputVariables: []string{"name"},
				PartialVariables: map[string]any{
					"day": "Monday",
				},
			},
			values: map[string]any{
				"name": "Charlie",
			},
			expected:  "Hello, Charlie! Today is Monday.",
			expectErr: false,
		},
		{
			name: "missing variable",
			template: &PromptTemplate{
				Template:       "Hello, {name}!",
				InputVariables: []string{"name"},
			},
			values:    map[string]any{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.template.Format(tt.values)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPromptTemplate_Partial(t *testing.T) {
	template := &PromptTemplate{
		Template:       "Hello, {name}! Today is {day} at {time}.",
		InputVariables: []string{"name", "day", "time"},
	}

	// 部分填充
	partial := template.Partial(map[string]any{
		"day": "Monday",
	})

	assert.Len(t, partial.InputVariables, 2)
	assert.Contains(t, partial.InputVariables, "name")
	assert.Contains(t, partial.InputVariables, "time")
	assert.NotContains(t, partial.InputVariables, "day")

	// 格式化
	result, err := partial.Format(map[string]any{
		"name": "Alice",
		"time": "10:00",
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice! Today is Monday at 10:00.", result)
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name:     "single variable",
			template: "Hello, {name}!",
			expected: []string{"name"},
		},
		{
			name:     "multiple variables",
			template: "{greeting}, {name}! You are {age} years old.",
			expected: []string{"greeting", "name", "age"},
		},
		{
			name:     "repeated variable",
			template: "Hello, {name}! Nice to meet you, {name}!",
			expected: []string{"name"},
		},
		{
			name:     "no variables",
			template: "Hello, world!",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVariables(tt.template)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestPromptValue(t *testing.T) {
	t.Run("StringPromptValue", func(t *testing.T) {
		pv := &StringPromptValue{Text: "Hello, world!"}
		
		assert.Equal(t, "Hello, world!", pv.ToString())
		
		messages := pv.ToMessages()
		require.Len(t, messages, 1)
		assert.Equal(t, "user", string(messages[0].Role))
		assert.Equal(t, "Hello, world!", messages[0].Content)
	})

	t.Run("MessagesPromptValue", func(t *testing.T) {
		messages := []struct {
			Role    string
			Content string
		}{
			{"system", "You are helpful."},
			{"user", "Hello!"},
		}

		var typedMessages []any
		for _, m := range messages {
			// 这里需要实际的 types.Message 类型
			typedMessages = append(typedMessages, m)
		}

		// 注意：这个测试需要实际的 types.Message
		// 实际测试将在集成测试中进行
	})
}

func TestPromptTemplate_Invoke(t *testing.T) {
	template, err := NewPromptTemplate(PromptTemplateConfig{
		Template: "Hello, {name}!",
	})
	require.NoError(t, err)

	// 测试作为 Runnable 使用
	result, err := template.Invoke(nil, map[string]any{
		"name": "Alice",
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestPromptTemplate_Batch(t *testing.T) {
	template, err := NewPromptTemplate(PromptTemplateConfig{
		Template: "Hello, {name}!",
	})
	require.NoError(t, err)

	inputs := []map[string]any{
		{"name": "Alice"},
		{"name": "Bob"},
		{"name": "Charlie"},
	}

	results, err := template.Batch(nil, inputs)
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "Hello, Alice!", results[0])
	assert.Equal(t, "Hello, Bob!", results[1])
	assert.Equal(t, "Hello, Charlie!", results[2])
}

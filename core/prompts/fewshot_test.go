package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFewShotPromptTemplate(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "Input: {input}\nOutput: {output}",
	})

	tests := []struct {
		name      string
		config    FewShotConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: FewShotConfig{
				Examples: []map[string]any{
					{"input": "happy", "output": "sad"},
				},
				ExamplePrompt:  examplePrompt,
				InputVariables: []string{"input"},
			},
			expectErr: false,
		},
		{
			name: "missing ExamplePrompt",
			config: FewShotConfig{
				Examples:       []map[string]any{},
				InputVariables: []string{"input"},
			},
			expectErr: true,
		},
		{
			name: "missing Examples and ExampleSelector",
			config: FewShotConfig{
				ExamplePrompt:  examplePrompt,
				InputVariables: []string{"input"},
			},
			expectErr: true,
		},
		{
			name: "missing InputVariables",
			config: FewShotConfig{
				Examples:      []map[string]any{},
				ExamplePrompt: examplePrompt,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewFewShotPromptTemplate(tt.config)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, template)
			}
		})
	}
}

func TestFewShotPromptTemplate_Format(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "Input: {input}\nOutput: {output}",
	})

	examples := []map[string]any{
		{"input": "happy", "output": "sad"},
		{"input": "tall", "output": "short"},
		{"input": "hot", "output": "cold"},
	}

	template, err := NewFewShotPromptTemplate(FewShotConfig{
		Examples:         examples,
		ExamplePrompt:    examplePrompt,
		ExampleSeparator: "\n\n",
		Prefix:           "Give the antonym of every input.\n\n",
		Suffix:           "\nInput: {input}\nOutput:",
		InputVariables:   []string{"input"},
	})
	require.NoError(t, err)

	result, err := template.Format(map[string]any{
		"input": "big",
	})

	require.NoError(t, err)
	assert.Contains(t, result, "Give the antonym of every input.")
	assert.Contains(t, result, "Input: happy\nOutput: sad")
	assert.Contains(t, result, "Input: tall\nOutput: short")
	assert.Contains(t, result, "Input: hot\nOutput: cold")
	assert.Contains(t, result, "Input: big\nOutput:")
}

func TestFewShotPromptTemplate_WithoutPrefix(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "{input} -> {output}",
	})

	examples := []map[string]any{
		{"input": "1", "output": "2"},
		{"input": "2", "output": "4"},
	}

	template, err := NewFewShotPromptTemplate(FewShotConfig{
		Examples:         examples,
		ExamplePrompt:    examplePrompt,
		ExampleSeparator: "\n",
		Suffix:           "\n{input} -> ",
		InputVariables:   []string{"input"},
	})
	require.NoError(t, err)

	result, err := template.Format(map[string]any{
		"input": "3",
	})

	require.NoError(t, err)
	assert.Equal(t, "1 -> 2\n2 -> 4\n3 -> ", result)
}

func TestFewShotPromptTemplate_Invoke(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "{input} = {output}",
	})

	examples := []map[string]any{
		{"input": "2+2", "output": "4"},
	}

	template, err := NewFewShotPromptTemplate(FewShotConfig{
		Examples:       examples,
		ExamplePrompt:  examplePrompt,
		Suffix:         "\n{input} = ",
		InputVariables: []string{"input"},
	})
	require.NoError(t, err)

	result, err := template.Invoke(nil, map[string]any{
		"input": "3+3",
	})

	require.NoError(t, err)
	assert.Contains(t, result, "2+2 = 4")
	assert.Contains(t, result, "3+3 = ")
}

func TestLengthBasedExampleSelector(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "Q: {question}\nA: {answer}",
	})

	examples := []map[string]any{
		{"question": "What is 2+2?", "answer": "4"},
		{"question": "What is the capital of France?", "answer": "Paris"},
		{"question": "Who wrote Romeo and Juliet?", "answer": "Shakespeare"},
	}

	// 使用足够大的长度确保能选择所有示例
	selector := NewLengthBasedExampleSelector(examples, examplePrompt, 200)

	selected, err := selector.SelectExamples(map[string]any{})
	require.NoError(t, err)

	// 应该选择所有示例，因为总长度小于 200
	assert.Len(t, selected, 3)
}

func TestLengthBasedExampleSelector_MaxLength(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "{text}",
	})

	examples := []map[string]any{
		{"text": "Short"},      // 5 chars
		{"text": "Medium text"}, // 11 chars
		{"text": "This is a much longer example text"}, // 36 chars
	}

	selector := NewLengthBasedExampleSelector(examples, examplePrompt, 20)

	selected, err := selector.SelectExamples(map[string]any{})
	require.NoError(t, err)

	// 应该只选择前两个示例（5 + 11 = 16 < 20）
	assert.Len(t, selected, 2)
	assert.Equal(t, "Short", selected[0]["text"])
	assert.Equal(t, "Medium text", selected[1]["text"])
}

func TestLengthBasedExampleSelector_AddExample(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "{text}",
	})

	examples := []map[string]any{
		{"text": "Example 1"},
	}

	selector := NewLengthBasedExampleSelector(examples, examplePrompt, 100)

	// 添加新示例
	err := selector.AddExample(map[string]any{"text": "Example 2"})
	require.NoError(t, err)

	assert.Len(t, selector.Examples, 2)
}

func TestFewShotPromptTemplate_WithExampleSelector(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "{text}",
	})

	examples := []map[string]any{
		{"text": "Example 1"},
		{"text": "Example 2"},
		{"text": "Example 3"},
	}

	selector := NewLengthBasedExampleSelector(examples, examplePrompt, 25)

	template, err := NewFewShotPromptTemplate(FewShotConfig{
		ExamplePrompt:   examplePrompt,
		ExampleSelector: selector,
		Suffix:          "\nInput: {input}",
		InputVariables:  []string{"input"},
	})
	require.NoError(t, err)

	result, err := template.Format(map[string]any{
		"input": "test",
	})

	require.NoError(t, err)
	// 应该包含部分示例（受长度限制）
	assert.Contains(t, result, "Example")
	assert.Contains(t, result, "Input: test")
}

func TestFewShotPromptTemplate_Batch(t *testing.T) {
	examplePrompt, _ := NewPromptTemplate(PromptTemplateConfig{
		Template: "{input} -> {output}",
	})

	examples := []map[string]any{
		{"input": "a", "output": "A"},
	}

	template, err := NewFewShotPromptTemplate(FewShotConfig{
		Examples:       examples,
		ExamplePrompt:  examplePrompt,
		Suffix:         "\n{input} -> ",
		InputVariables: []string{"input"},
	})
	require.NoError(t, err)

	inputs := []map[string]any{
		{"input": "b"},
		{"input": "c"},
	}

	results, err := template.Batch(nil, inputs)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Contains(t, results[0], "a -> A")
	assert.Contains(t, results[0], "b -> ")
	assert.Contains(t, results[1], "a -> A")
	assert.Contains(t, results[1], "c -> ")
}

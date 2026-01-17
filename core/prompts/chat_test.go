package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestSystemMessagePromptTemplate(t *testing.T) {
	template := SystemMessagePromptTemplate("You are a {role}.")
	
	msg, err := template.Format(map[string]any{
		"role": "helpful assistant",
	})

	require.NoError(t, err)
	assert.Equal(t, types.RoleSystem, msg.Role)
	assert.Equal(t, "You are a helpful assistant.", msg.Content)
}

func TestHumanMessagePromptTemplate(t *testing.T) {
	template := HumanMessagePromptTemplate("Hello, my name is {name}.")
	
	msg, err := template.Format(map[string]any{
		"name": "Alice",
	})

	require.NoError(t, err)
	assert.Equal(t, types.RoleUser, msg.Role)
	assert.Equal(t, "Hello, my name is Alice.", msg.Content)
}

func TestAIMessagePromptTemplate(t *testing.T) {
	template := AIMessagePromptTemplate("Nice to meet you, {name}!")
	
	msg, err := template.Format(map[string]any{
		"name": "Bob",
	})

	require.NoError(t, err)
	assert.Equal(t, types.RoleAssistant, msg.Role)
	assert.Equal(t, "Nice to meet you, Bob!", msg.Content)
}

func TestNewChatPromptTemplate(t *testing.T) {
	template := NewChatPromptTemplate(
		SystemMessagePromptTemplate("You are a {role}."),
		HumanMessagePromptTemplate("My name is {name}."),
		AIMessagePromptTemplate("Nice to meet you!"),
	)

	assert.NotNil(t, template)
	assert.Len(t, template.Messages, 3)
	assert.ElementsMatch(t, []string{"role", "name"}, template.InputVariables)
}

func TestChatPromptTemplate_FormatMessages(t *testing.T) {
	template := NewChatPromptTemplate(
		SystemMessagePromptTemplate("You are a {role}."),
		HumanMessagePromptTemplate("Hello, my name is {name}!"),
		AIMessagePromptTemplate("Nice to meet you, {name}!"),
	)

	messages, err := template.FormatMessages(map[string]any{
		"role": "helpful assistant",
		"name": "Alice",
	})

	require.NoError(t, err)
	require.Len(t, messages, 3)

	assert.Equal(t, types.RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", messages[0].Content)

	assert.Equal(t, types.RoleUser, messages[1].Role)
	assert.Equal(t, "Hello, my name is Alice!", messages[1].Content)

	assert.Equal(t, types.RoleAssistant, messages[2].Role)
	assert.Equal(t, "Nice to meet you, Alice!", messages[2].Content)
}

func TestChatPromptTemplate_FromMessages(t *testing.T) {
	template, err := FromMessages([]any{
		[]any{"system", "You are a {role}."},
		[]any{"human", "Hello, {name}!"},
		[]any{"ai", "Nice to meet you!"},
	})

	require.NoError(t, err)
	assert.NotNil(t, template)
	assert.Len(t, template.Messages, 3)

	// 测试格式化
	messages, err := template.FormatMessages(map[string]any{
		"role": "teacher",
		"name": "Bob",
	})

	require.NoError(t, err)
	require.Len(t, messages, 3)
	assert.Equal(t, "You are a teacher.", messages[0].Content)
	assert.Equal(t, "Hello, Bob!", messages[1].Content)
}

func TestChatPromptTemplate_FromMessages_InvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		messages  []any
		expectErr bool
	}{
		{
			name: "invalid message length",
			messages: []any{
				[]any{"system"}, // 缺少模板
			},
			expectErr: true,
		},
		{
			name: "invalid role type",
			messages: []any{
				[]any{123, "template"},
			},
			expectErr: true,
		},
		{
			name: "invalid template type",
			messages: []any{
				[]any{"system", 123},
			},
			expectErr: true,
		},
		{
			name: "unknown role",
			messages: []any{
				[]any{"unknown", "template"},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromMessages(tt.messages)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChatPromptTemplate_Partial(t *testing.T) {
	template := NewChatPromptTemplate(
		SystemMessagePromptTemplate("You are a {role}."),
		HumanMessagePromptTemplate("My name is {name} and I'm {age} years old."),
	)

	// 部分填充
	partial := template.Partial(map[string]any{
		"role": "helper",
	})

	assert.Len(t, partial.InputVariables, 2)
	assert.Contains(t, partial.InputVariables, "name")
	assert.Contains(t, partial.InputVariables, "age")
	assert.NotContains(t, partial.InputVariables, "role")

	// 格式化
	messages, err := partial.FormatMessages(map[string]any{
		"name": "Alice",
		"age":  25,
	})

	require.NoError(t, err)
	assert.Equal(t, "You are a helper.", messages[0].Content)
	assert.Equal(t, "My name is Alice and I'm 25 years old.", messages[1].Content)
}

func TestChatPromptTemplate_Invoke(t *testing.T) {
	template := NewChatPromptTemplate(
		SystemMessagePromptTemplate("You are helpful."),
		HumanMessagePromptTemplate("Hello, {name}!"),
	)

	messages, err := template.Invoke(nil, map[string]any{
		"name": "Alice",
	})

	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, types.RoleSystem, messages[0].Role)
	assert.Equal(t, types.RoleUser, messages[1].Role)
}

func TestChatPromptTemplate_Batch(t *testing.T) {
	template := NewChatPromptTemplate(
		HumanMessagePromptTemplate("Hello, {name}!"),
	)

	inputs := []map[string]any{
		{"name": "Alice"},
		{"name": "Bob"},
	}

	results, err := template.Batch(nil, inputs)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Hello, Alice!", results[0][0].Content)
	assert.Equal(t, "Hello, Bob!", results[1][0].Content)
}

func TestMessagesPromptValue(t *testing.T) {
	messages := []types.Message{
		types.NewSystemMessage("You are helpful."),
		types.NewUserMessage("Hello!"),
	}

	pv := &MessagesPromptValue{Messages: messages}

	// 测试 ToMessages
	result := pv.ToMessages()
	assert.Equal(t, messages, result)

	// 测试 ToString
	str := pv.ToString()
	assert.Contains(t, str, "system: You are helpful.")
	assert.Contains(t, str, "user: Hello!")
}

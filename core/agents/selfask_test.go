package agents

import (
	"context"
	"testing"
	
	"langchain-go/core/tools"
	"langchain-go/pkg/types"
)

// MockSearchTool 是用于测试的模拟搜索工具。
type MockSearchTool struct{}

func (m *MockSearchTool) GetName() string {
	return "search"
}

func (m *MockSearchTool) GetDescription() string {
	return "Search for information"
}

func (m *MockSearchTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"query": {
				Type:        "string",
				Description: "The search query",
			},
		},
		Required: []string{"query"},
	}
}

func (m *MockSearchTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	query := input["input"].(string)
	return "Search result for: " + query, nil
}

func (m *MockSearchTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        m.GetName(),
		Description: m.GetDescription(),
	}
}

// TestSelfAskAgent 测试 Self-Ask Agent。
func TestSelfAskAgent(t *testing.T) {
	t.Skip("Skipping Self-Ask Agent test - requires LLM")
	
	// 这里需要一个真实的 LLM 来测试
	// 可以使用模拟的 LLM 或跳过测试
	
	searchTool := &MockSearchTool{}
	
	agent := CreateSelfAskAgent(
		nil, // 需要提供 LLM
		searchTool,
		WithSelfAskMaxSubQuestions(3),
		WithSelfAskVerbose(true),
	)
	
	if agent == nil {
		t.Error("Expected agent to be created")
	}
}

// TestStructuredChatAgent 测试 Structured Chat Agent。
func TestStructuredChatAgent(t *testing.T) {
	t.Skip("Skipping Structured Chat Agent test - requires LLM")
	
	searchTool := &MockSearchTool{}
	
	agent := CreateStructuredChatAgent(
		nil, // 需要提供 LLM
		[]tools.Tool{searchTool},
		WithStructuredChatOutputFormat("json"),
		WithStructuredChatVerbose(true),
	)
	
	if agent == nil {
		t.Error("Expected agent to be created")
	}
}

// TestSelfAskPrompt 测试 Self-Ask 提示词解析。
func TestSelfAskPrompt(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantType AgentActionType
	}{
		{
			name:     "Follow-up question",
			output:   "Yes.\nFollow up: What is the capital of France?",
			wantType: ActionToolCall,
		},
		{
			name:     "Final answer",
			output:   "No.\nSo the final answer is: Paris is the capital of France.",
			wantType: ActionFinish,
		},
	}
	
	config := SelfAskConfig{
		AgentConfig: AgentConfig{
			Type:     AgentTypeSelfAsk,
			LLM:      nil,
			Tools:    []tools.Tool{&MockSearchTool{}},
			MaxSteps: 5,
		},
		MaxSubQuestions: 3,
	}
	
	agent := NewSelfAskAgent(config)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := agent.parseOutput(tt.output)
			if err != nil {
				t.Errorf("parseOutput() error = %v", err)
				return
			}
			
			if action.Type != tt.wantType {
				t.Errorf("parseOutput() action type = %v, want %v", action.Type, tt.wantType)
			}
		})
	}
}

// TestExtractFollowUpQuestion 测试提取后续问题。
func TestExtractFollowUpQuestion(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "Standard format",
			text: "Yes.\nFollow up: What is the capital?",
			want: "What is the capital?",
		},
		{
			name: "Follow-up question format",
			text: "Follow up question: Who is the president?",
			want: "Who is the president?",
		},
		{
			name: "Sub-question format",
			text: "Sub-question: How old is he?",
			want: "How old is he?",
		},
		{
			name: "No question",
			text: "No follow up needed.",
			want: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFollowUpQuestion(tt.text)
			if got != tt.want {
				t.Errorf("extractFollowUpQuestion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStructuredChatOutputFormat 测试结构化输出格式。
func TestStructuredChatOutputFormat(t *testing.T) {
	config := StructuredChatConfig{
		AgentConfig: AgentConfig{
			Type:  AgentTypeStructuredChat,
			LLM:   nil,
			Tools: []tools.Tool{},
		},
		OutputFormat:   "json",
		ConversationID: "test-123",
	}
	
	agent := NewStructuredChatAgent(config)
	
	content := "Hello, how can I help you?"
	formatted := agent.formatOutput(content)
	
	// JSON 格式应该包含响应内容
	if len(formatted) == 0 {
		t.Error("Expected formatted output")
	}
}

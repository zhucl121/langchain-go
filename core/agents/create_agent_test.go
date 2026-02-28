package agents

import (
	"context"
	"testing"

	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/core/tools"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// ─── CreateUnifiedAgent 测试 ──────────────────────────────────

func TestCreateUnifiedAgent_NoModel(t *testing.T) {
	_, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Preset: PresetReAct,
	})
	if err == nil {
		t.Fatal("expected error when model is nil")
	}
	t.Logf("expected error: %v", err)
}

func TestCreateUnifiedAgent_ReActPreset(t *testing.T) {
	model := NewMockChatModel()

	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Preset:   PresetReAct,
		Model:    model,
		MaxSteps: 5,
	})
	if err != nil {
		t.Fatalf("CreateUnifiedAgent failed: %v", err)
	}
	if ca.Agent == nil {
		t.Fatal("agent should not be nil")
	}
	if ca.Executor == nil {
		t.Fatal("executor should not be nil")
	}
	if ca.Agent.GetType() != AgentTypeReAct {
		t.Errorf("expected react agent, got %s", ca.Agent.GetType())
	}
}

func TestCreateUnifiedAgent_ToolCallingPreset(t *testing.T) {
	model := NewMockChatModel()

	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Preset: PresetToolCalling,
		Model:  model,
	})
	if err != nil {
		t.Fatalf("CreateUnifiedAgent failed: %v", err)
	}
	// ToolCalling 预设使用 OpenAIFunctionsAgent 实现（返回 openai_functions 类型）
	agentType := ca.Agent.GetType()
	if agentType != AgentTypeToolCalling && agentType != "openai_functions" {
		t.Errorf("expected tool_calling or openai_functions agent, got %s", agentType)
	}
}

func TestCreateUnifiedAgent_DefaultToToolCalling_WhenToolsProvided(t *testing.T) {
	model := NewMockChatModel()

	// 不指定 Preset，有工具时默认为 ToolCalling（底层用 OpenAIFunctionsAgent）
	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Model: model,
		Tools: []tools.Tool{mockTool("calc")},
	})
	if err != nil {
		t.Fatalf("CreateUnifiedAgent failed: %v", err)
	}
	agentType := ca.Agent.GetType()
	if agentType != AgentTypeToolCalling && agentType != "openai_functions" {
		t.Errorf("expected tool_calling or openai_functions when tools provided, got %s", agentType)
	}
}

func TestCreateUnifiedAgent_DefaultToReAct_WhenNoTools(t *testing.T) {
	model := NewMockChatModel()

	// 不指定 Preset，无工具时默认为 ReAct
	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Model: model,
	})
	if err != nil {
		t.Fatalf("CreateUnifiedAgent failed: %v", err)
	}
	if ca.Agent.GetType() != AgentTypeReAct {
		t.Errorf("expected react when no tools, got %s", ca.Agent.GetType())
	}
}

func TestCreateUnifiedAgent_DefaultMaxSteps(t *testing.T) {
	model := NewMockChatModel()

	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Model: model,
	})
	if err != nil {
		t.Fatalf("CreateUnifiedAgent failed: %v", err)
	}
	// MaxSteps 默认为 10
	if ca.Config.MaxSteps != 10 {
		t.Errorf("expected default MaxSteps=10, got %d", ca.Config.MaxSteps)
	}
}

// ─── Run 方法测试 ─────────────────────────────────────────────

func TestManagedAgent_Run(t *testing.T) {
	model := NewMockChatModel()
	model.InvokeFunc = func(_ context.Context, _ []types.Message, _ ...runnable.Option) (types.Message, error) {
		return types.NewAssistantMessage("Final Answer: 42"), nil
	}

	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Preset:   PresetReAct,
		Model:    model,
		MaxSteps: 5,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	result, err := ca.Run(context.Background(), "What is 6*7?")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

// ─── 内存记忆测试 ─────────────────────────────────────────────

func TestInMemoryConversationMemory_BasicOps(t *testing.T) {
	mem := NewInMemoryConversationMemory()

	// 初始为空
	msgs, err := mem.LoadMessages(context.Background())
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty messages, got %d", len(msgs))
	}

	// 保存消息
	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
		{Role: types.RoleAssistant, Content: "Hi there!"},
	}
	if err := mem.SaveMessages(context.Background(), messages); err != nil {
		t.Fatalf("SaveMessages failed: %v", err)
	}

	// 读取消息
	loaded, err := mem.LoadMessages(context.Background())
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 messages, got %d", len(loaded))
	}
	if loaded[0].Content != "Hello" {
		t.Errorf("expected Hello, got %s", loaded[0].Content)
	}

	// 计数
	if mem.MessageCount() != 2 {
		t.Errorf("expected count 2, got %d", mem.MessageCount())
	}

	// 清除
	if err := mem.Clear(context.Background()); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if mem.MessageCount() != 0 {
		t.Error("expected empty after clear")
	}
}

func TestManagedAgent_RunWithMemory_NoMemoryConfigured(t *testing.T) {
	model := NewMockChatModel()

	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Preset: PresetReAct,
		Model:  model,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 没有配置 memory，应该正常运行
	result, err := ca.RunWithMemory(context.Background(), "test")
	if err != nil {
		t.Fatalf("RunWithMemory without memory failed: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestManagedAgent_RunWithMemory_WithMemory(t *testing.T) {
	model := NewMockChatModel()

	mem := NewInMemoryConversationMemory()
	ca, err := CreateUnifiedAgent(UnifiedAgentConfig{
		Preset: PresetReAct,
		Model:  model,
		Memory: mem,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 第一次调用
	result1, err := ca.RunWithMemory(context.Background(), "What is Go?")
	if err != nil {
		t.Fatalf("first RunWithMemory failed: %v", err)
	}
	if result1 == nil {
		t.Fatal("result1 should not be nil")
	}

	// 第一次调用后应该有记忆
	if mem.MessageCount() == 0 {
		t.Error("memory should not be empty after first run")
	}

	// 第二次调用（有历史上下文）
	result2, err := ca.RunWithMemory(context.Background(), "Tell me more")
	if err != nil {
		t.Fatalf("second RunWithMemory failed: %v", err)
	}
	if result2 == nil {
		t.Fatal("result2 should not be nil")
	}

	t.Logf("memory count after 2 runs: %d", mem.MessageCount())
}

// ─── 选项函数测试 ─────────────────────────────────────────────

func TestUnifiedAgentOptions(t *testing.T) {
	model := NewMockChatModel()
	mem := NewInMemoryConversationMemory()

	ca, err := NewReActAgentV2(model, nil,
		WithUnifiedSystemPrompt("You are expert"),
		WithUnifiedMaxSteps(15),
		WithUnifiedVerbose(true),
		WithUnifiedMemory(mem),
	)
	if err != nil {
		t.Fatalf("NewReActAgentV2 failed: %v", err)
	}

	if ca.Config.SystemPrompt != "You are expert" {
		t.Errorf("system prompt not set correctly: %s", ca.Config.SystemPrompt)
	}
	if ca.Config.MaxSteps != 15 {
		t.Errorf("max steps not set: %d", ca.Config.MaxSteps)
	}
	if !ca.Config.Verbose {
		t.Error("verbose should be true")
	}
	if ca.Config.Memory == nil {
		t.Error("memory should be set")
	}
}

func TestNewToolCallingAgentV2(t *testing.T) {
	model := NewMockChatModel()

	ca, err := NewToolCallingAgentV2(model, nil)
	if err != nil {
		t.Fatalf("NewToolCallingAgentV2 failed: %v", err)
	}
	agentType := ca.Agent.GetType()
	if agentType != AgentTypeToolCalling && agentType != "openai_functions" {
		t.Errorf("expected tool_calling or openai_functions, got %s", agentType)
	}
}

func TestNewMemoryAgentV2(t *testing.T) {
	model := NewMockChatModel()
	mem := NewInMemoryConversationMemory()

	ca, err := NewMemoryAgentV2(model, nil,
		WithUnifiedMemory(mem),
	)
	if err != nil {
		t.Fatalf("NewMemoryAgentV2 failed: %v", err)
	}
	if ca.Config.Memory == nil {
		t.Error("memory should be set")
	}
}

// ─── Preset 解析测试 ──────────────────────────────────────────

func TestResolveUnifiedAgentType_AllPresets(t *testing.T) {
	tests := []struct {
		preset   AgentPreset
		wantType AgentType
	}{
		{PresetReAct, AgentTypeReAct},
		{PresetToolCalling, AgentTypeToolCalling},
		{PresetConversationalV2, AgentTypeConversational},
		{PresetPlanExecute, AgentTypePlanAndExecute},
		{PresetSelfAsk, AgentTypeSelfAsk},
	}

	for _, tt := range tests {
		t.Run(string(tt.preset), func(t *testing.T) {
			config := UnifiedAgentConfig{Preset: tt.preset}
			got := resolveUnifiedAgentType(config)
			if got != tt.wantType {
				t.Errorf("preset %s: want %s, got %s", tt.preset, tt.wantType, got)
			}
		})
	}
}

// ─── 测试辅助 ─────────────────────────────────────────────────

// mockToolImpl 模拟工具实现（实现 tools.Tool 接口）。
type mockToolImpl struct {
	name string
}

func (t *mockToolImpl) GetName() string        { return t.name }
func (t *mockToolImpl) GetDescription() string  { return "mock tool" }
func (t *mockToolImpl) GetParameters() types.Schema { return types.Schema{} }
func (t *mockToolImpl) Execute(_ context.Context, _ map[string]any) (any, error) {
	return "mock result", nil
}
func (t *mockToolImpl) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: "mock tool",
	}
}

func mockTool(name string) tools.Tool {
	return &mockToolImpl{name: name}
}

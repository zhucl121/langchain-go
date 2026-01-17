package agents

import (
	"context"
	"testing"
	"time"
	
	"github.com/zhucl121/langchain-go/core/tools"
)

// MockMultiAgent 模拟的 MultiAgent
type MockMultiAgent struct {
	BaseMultiAgent
}

func NewMockMultiAgent(id string) *MockMultiAgent {
	return &MockMultiAgent{
		BaseMultiAgent: BaseMultiAgent{
			id:           id,
			capabilities: []string{"test"},
		},
	}
}

func (m *MockMultiAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	// 简单地返回一个结果
	if msg.Type == MessageTypeTask {
		return m.SendResult(ctx, msg, "task completed")
	}
	return nil
}

func (m *MockMultiAgent) CanHandle(task string) (bool, float64) {
	return true, 0.8
}

func (m *MockMultiAgent) GetTools() []tools.Tool {
	return nil
}

func (m *MockMultiAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error) {
	return &AgentAction{
		Tool:      "test",
		ToolInput: map[string]any{"input": input},
	}, nil
}

func (m *MockMultiAgent) GetType() AgentType {
	return AgentTypeReAct
}

func TestMultiAgentSystem_AddAgent(t *testing.T) {
	llm := &MockChatModel{}
	strategy := NewSequentialStrategy(llm)
	coordinator := NewCoordinatorAgent("coordinator", llm, strategy)
	
	config := DefaultMultiAgentConfig()
	system := NewMultiAgentSystem(coordinator, config)
	
	// 测试添加 Agent
	agent := NewMockMultiAgent("agent1")
	err := system.AddAgent("agent1", agent)
	if err != nil {
		t.Fatalf("Failed to add agent: %v", err)
	}
	
	// 测试重复添加
	err = system.AddAgent("agent1", agent)
	if err == nil {
		t.Fatal("Expected error when adding duplicate agent")
	}
	
	// 测试获取 Agent
	retrieved, exists := system.GetAgent("agent1")
	if !exists {
		t.Fatal("Agent not found after adding")
	}
	if retrieved.ID() != "agent1" {
		t.Fatalf("Expected agent1, got %s", retrieved.ID())
	}
}

func TestMultiAgentSystem_RemoveAgent(t *testing.T) {
	llm := &MockChatModel{}
	strategy := NewSequentialStrategy(llm)
	coordinator := NewCoordinatorAgent("coordinator", llm, strategy)
	
	config := DefaultMultiAgentConfig()
	system := NewMultiAgentSystem(coordinator, config)
	
	agent := NewMockMultiAgent("agent1")
	system.AddAgent("agent1", agent)
	
	// 测试移除 Agent
	err := system.RemoveAgent("agent1")
	if err != nil {
		t.Fatalf("Failed to remove agent: %v", err)
	}
	
	// 确认已移除
	_, exists := system.GetAgent("agent1")
	if exists {
		t.Fatal("Agent still exists after removal")
	}
	
	// 测试移除不存在的 Agent
	err = system.RemoveAgent("nonexistent")
	if err == nil {
		t.Fatal("Expected error when removing nonexistent agent")
	}
}

func TestMultiAgentSystem_ListAgents(t *testing.T) {
	llm := &MockChatModel{}
	strategy := NewSequentialStrategy(llm)
	coordinator := NewCoordinatorAgent("coordinator", llm, strategy)
	
	config := DefaultMultiAgentConfig()
	system := NewMultiAgentSystem(coordinator, config)
	
	// 添加多个 Agent
	system.AddAgent("agent1", NewMockMultiAgent("agent1"))
	system.AddAgent("agent2", NewMockMultiAgent("agent2"))
	system.AddAgent("agent3", NewMockMultiAgent("agent3"))
	
	agents := system.ListAgents()
	if len(agents) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(agents))
	}
}

func TestMessageBus_RegisterAndSend(t *testing.T) {
	bus := NewMessageBus(10)
	agent := NewMockMultiAgent("agent1")
	
	bus.RegisterAgent("agent1", agent)
	agent.SetMessageBus(bus)
	
	ctx := context.Background()
	msg := &AgentMessage{
		ID:      "msg1",
		From:    "system",
		To:      "agent1",
		Type:    MessageTypeTask,
		Content: "test task",
	}
	
	err := bus.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
	
	// 验证消息计数
	count := bus.GetMessageCount()
	if count != 1 {
		t.Fatalf("Expected 1 message, got %d", count)
	}
}

func TestMessageBus_Route(t *testing.T) {
	bus := NewMessageBus(10)
	agent := NewMockMultiAgent("agent1")
	bus.RegisterAgent("agent1", agent)
	agent.SetMessageBus(bus)
	
	ctx := context.Background()
	msg := &AgentMessage{
		ID:      "msg1",
		From:    "system",
		To:      "agent1",
		Type:    MessageTypeTask,
		Content: "test task",
	}
	
	// 测试点对点路由
	err := bus.Route(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to route message: %v", err)
	}
	
	// 测试路由到不存在的 Agent
	msg.To = "nonexistent"
	err = bus.Route(ctx, msg)
	if err == nil {
		t.Fatal("Expected error when routing to nonexistent agent")
	}
}

func TestSharedState(t *testing.T) {
	state := NewSharedState()
	
	// 测试 Set 和 Get
	state.Set("key1", "value1")
	value, exists := state.Get("key1")
	if !exists {
		t.Fatal("Key not found after setting")
	}
	if value != "value1" {
		t.Fatalf("Expected value1, got %v", value)
	}
	
	// 测试 Delete
	state.Delete("key1")
	_, exists = state.Get("key1")
	if exists {
		t.Fatal("Key still exists after deletion")
	}
	
	// 测试 GetAll
	state.Set("key1", "value1")
	state.Set("key2", "value2")
	all := state.GetAll()
	if len(all) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(all))
	}
	
	// 测试 Clear
	state.Clear()
	all = state.GetAll()
	if len(all) != 0 {
		t.Fatal("State not cleared")
	}
}

func TestExecutionHistory(t *testing.T) {
	history := NewExecutionHistory()
	
	msg := &AgentMessage{
		ID:      "msg1",
		From:    "system",
		To:      "agent1",
		Type:    MessageTypeTask,
		Content: "test task",
	}
	
	// 测试记录开始
	history.RecordStart(msg)
	
	record, exists := history.GetRecord("msg1")
	if !exists {
		t.Fatal("Record not found after recording start")
	}
	if record.Status != "running" {
		t.Fatalf("Expected status 'running', got '%s'", record.Status)
	}
	
	// 测试记录完成
	result := &MultiAgentResult{
		RootMessageID: "msg1",
		FinalResult:   "completed",
		MessageCount:  5,
	}
	history.RecordComplete("msg1", result)
	
	record, _ = history.GetRecord("msg1")
	if record.Status != "completed" {
		t.Fatalf("Expected status 'completed', got '%s'", record.Status)
	}
}

func TestMultiAgentMetrics(t *testing.T) {
	metrics := NewMultiAgentMetrics()
	
	// 测试计数
	metrics.IncrementTotalRuns()
	metrics.IncrementSuccessfulRuns()
	metrics.IncrementTotalMessages()
	
	if metrics.TotalRuns != 1 {
		t.Fatalf("Expected 1 total run, got %d", metrics.TotalRuns)
	}
	if metrics.SuccessfulRuns != 1 {
		t.Fatalf("Expected 1 successful run, got %d", metrics.SuccessfulRuns)
	}
	
	// 测试执行时间记录
	metrics.RecordExecutionTime(100 * time.Millisecond)
	if metrics.AverageTime != 100*time.Millisecond {
		t.Fatalf("Expected 100ms, got %v", metrics.AverageTime)
	}
	
	// 测试 Agent 使用率
	metrics.RecordAgentUtilization("agent1")
	metrics.RecordAgentUtilization("agent1")
	metrics.RecordAgentUtilization("agent2")
	
	if metrics.AgentUtilization["agent1"] != 2 {
		t.Fatalf("Expected 2 for agent1, got %d", metrics.AgentUtilization["agent1"])
	}
	if metrics.AgentUtilization["agent2"] != 1 {
		t.Fatalf("Expected 1 for agent2, got %d", metrics.AgentUtilization["agent2"])
	}
	
	// 测试统计信息
	stats := metrics.GetStats()
	if stats["total_runs"].(int64) != 1 {
		t.Fatal("Stats total_runs mismatch")
	}
}

func TestResearcherAgent_CanHandle(t *testing.T) {
	llm := &MockChatModel{}
	researcher := NewResearcherAgent("researcher", llm, nil)
	
	testCases := []struct {
		task     string
		expected bool
	}{
		{"research the topic", true},
		{"find information about AI", true},
		{"investigate the issue", true},
		{"write a report", false},
		{"analyze data", false},
	}
	
	for _, tc := range testCases {
		canHandle, score := researcher.CanHandle(tc.task)
		if canHandle != tc.expected {
			t.Errorf("Task '%s': expected %v, got %v (score: %.2f)", 
				tc.task, tc.expected, canHandle, score)
		}
	}
}

func TestWriterAgent_CanHandle(t *testing.T) {
	llm := &MockChatModel{}
	writer := NewWriterAgent("writer", llm, "technical")
	
	testCases := []struct {
		task     string
		expected bool
	}{
		{"write an article", true},
		{"compose a summary", true},
		{"create content", true},
		{"research the topic", false},
		{"analyze data", false},
	}
	
	for _, tc := range testCases {
		canHandle, score := writer.CanHandle(tc.task)
		if canHandle != tc.expected {
			t.Errorf("Task '%s': expected %v, got %v (score: %.2f)", 
				tc.task, tc.expected, canHandle, score)
		}
	}
}

func TestReviewerAgent_CanHandle(t *testing.T) {
	llm := &MockChatModel{}
	reviewer := NewReviewerAgent("reviewer", llm, nil)
	
	testCases := []struct {
		task     string
		expected bool
	}{
		{"review the document", true},
		{"evaluate the content", true},
		{"check for errors", true},
		{"write a report", false},
		{"research the topic", false},
	}
	
	for _, tc := range testCases {
		canHandle, score := reviewer.CanHandle(tc.task)
		if canHandle != tc.expected {
			t.Errorf("Task '%s': expected %v, got %v (score: %.2f)", 
				tc.task, tc.expected, canHandle, score)
		}
	}
}

func TestAnalystAgent_CanHandle(t *testing.T) {
	llm := &MockChatModel{}
	analyst := NewAnalystAgent("analyst", llm)
	
	testCases := []struct {
		task     string
		expected bool
	}{
		{"analyze the data", true},
		{"find patterns", true},
		{"provide insights", true},
		{"write a report", false},
		{"review document", false},
	}
	
	for _, tc := range testCases {
		canHandle, score := analyst.CanHandle(tc.task)
		if canHandle != tc.expected {
			t.Errorf("Task '%s': expected %v, got %v (score: %.2f)", 
				tc.task, tc.expected, canHandle, score)
		}
	}
}

func TestSequentialStrategy_SelectAgent(t *testing.T) {
	llm := &MockChatModel{}
	strategy := NewSequentialStrategy(llm)
	
	agents := make(map[string]MultiAgent)
	agents["researcher"] = NewResearcherAgent("researcher", llm, nil)
	agents["writer"] = NewWriterAgent("writer", llm, "technical")
	
	ctx := context.Background()
	
	// 测试选择研究 Agent
	agentID, err := strategy.SelectAgent(ctx, "research this topic", agents)
	if err != nil {
		t.Fatalf("Failed to select agent: %v", err)
	}
	if agentID != "researcher" {
		t.Fatalf("Expected researcher, got %s", agentID)
	}
	
	// 测试选择写作 Agent
	agentID, err = strategy.SelectAgent(ctx, "write an article", agents)
	if err != nil {
		t.Fatalf("Failed to select agent: %v", err)
	}
	if agentID != "writer" {
		t.Fatalf("Expected writer, got %s", agentID)
	}
}

func BenchmarkMessageBus_Send(b *testing.B) {
	bus := NewMessageBus(1000)
	ctx := context.Background()
	
	msg := &AgentMessage{
		ID:      "msg1",
		From:    "system",
		To:      "agent1",
		Type:    MessageTypeTask,
		Content: "test task",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Send(ctx, msg)
	}
}

func BenchmarkSharedState_SetGet(b *testing.B) {
	state := NewSharedState()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.Set("key", "value")
		state.Get("key")
	}
}

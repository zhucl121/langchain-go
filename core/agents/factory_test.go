package agents_test

import (
	"context"
	"testing"
	
	"langchain-go/core/agents"
	"langchain-go/core/chat/ollama"
	"langchain-go/core/tools"
)

// TestCreateReActAgent 测试创建 ReAct Agent。
func TestCreateReActAgent(t *testing.T) {
	// 创建 LLM
	llm := ollama.NewChatOllama("qwen2.5:7b")
	
	// 创建工具
	agentTools := []tools.Tool{
		tools.NewCalculator(),
		tools.NewGetTimeTool(nil),
		tools.NewGetDateTool(nil),
	}
	
	// 创建 Agent
	agent := agents.CreateReActAgent(llm, agentTools,
		agents.WithMaxSteps(10),
		agents.WithVerbose(false),
	)
	
	if agent == nil {
		t.Fatal("agent should not be nil")
	}
	
	if agent.GetType() != agents.AgentTypeReAct {
		t.Errorf("expected agent type %s, got %s", agents.AgentTypeReAct, agent.GetType())
	}
	
	agentToolsFromAgent := agent.GetTools()
	if len(agentToolsFromAgent) != 3 {
		t.Errorf("expected 3 tools, got %d", len(agentToolsFromAgent))
	}
}

// TestCreateToolCallingAgent 测试创建 Tool Calling Agent。
func TestCreateToolCallingAgent(t *testing.T) {
	// 创建 LLM
	llm := ollama.NewChatOllama("qwen2.5:7b")
	
	// 创建工具
	agentTools := []tools.Tool{
		tools.NewCalculator(),
	}
	
	// 创建 Agent
	agent := agents.CreateToolCallingAgent(llm, agentTools,
		agents.WithSystemPrompt("You are a helpful assistant"),
	)
	
	if agent == nil {
		t.Fatal("agent should not be nil")
	}
	
	if agent.GetType() != agents.AgentTypeToolCalling {
		t.Errorf("expected agent type %s, got %s", agents.AgentTypeToolCalling, agent.GetType())
	}
}

// TestSimplifiedAgentExecutor 测试简化的 Agent 执行器。
func TestSimplifiedAgentExecutor(t *testing.T) {
	// 跳过需要实际 LLM 调用的测试
	t.Skip("Requires actual LLM")
	
	ctx := context.Background()
	
	// 创建 LLM
	llm := ollama.NewChatOllama("qwen2.5:7b")
	
	// 创建工具
	agentTools := []tools.Tool{
		tools.NewCalculator(),
		tools.NewGetTimeTool(nil),
	}
	
	// 创建 Agent
	agent := agents.CreateReActAgent(llm, agentTools)
	
	// 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools,
		agents.WithMaxSteps(5),
		agents.WithVerbose(true),
	)
	
	// 执行
	result, err := executor.Run(ctx, "What is 25 * 4?")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	
	if !result.Success {
		t.Error("expected successful execution")
	}
	
	if result.Output == "" {
		t.Error("expected non-empty output")
	}
}

// ExampleCreateReActAgent 示例：创建 ReAct Agent。
func ExampleCreateReActAgent() {
	// 创建 LLM
	llm := ollama.NewChatOllama("qwen2.5:7b")
	
	// 获取内置工具
	agentTools := tools.GetBasicTools()
	
	// 创建 ReAct Agent（只需 1 行！）
	agent := agents.CreateReActAgent(llm, agentTools)
	
	// 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools)
	
	// 执行任务
	result, _ := executor.Run(context.Background(), "What time is it?")
	println("Result:", result.Output)
}

// ExampleCreateToolCallingAgent 示例：创建 Tool Calling Agent。
func ExampleCreateToolCallingAgent() {
	// 创建支持工具调用的 LLM
	llm := ollama.NewChatOllama("qwen2.5:7b")
	
	// 获取所有内置工具
	agentTools := tools.GetBuiltinTools()
	
	// 创建 Tool Calling Agent
	agent := agents.CreateToolCallingAgent(llm, agentTools,
		agents.WithMaxSteps(10),
		agents.WithSystemPrompt("You are a helpful assistant that can use tools"),
	)
	
	// 创建执行器
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools)
	
	// 执行任务
	result, _ := executor.Run(context.Background(), "Calculate 123 * 456")
	println("Result:", result.Output)
}

// ExampleAgentExecutor_Stream 示例：流式执行 Agent。
func ExampleAgentExecutor_Stream() {
	llm := ollama.NewChatOllama("qwen2.5:7b")
	agentTools := tools.GetBasicTools()
	
	agent := agents.CreateReActAgent(llm, agentTools)
	executor := agents.NewSimplifiedAgentExecutor(agent, agentTools)
	
	ctx := context.Background()
	eventChan := executor.Stream(ctx, "What is 10 + 20?")
	
	for event := range eventChan {
		switch event.Type {
		case agents.EventTypeStart:
			println("Agent started")
		case agents.EventTypeStep:
			println("Step:", event.Step)
		case agents.EventTypeToolCall:
			println("Tool call:", event.Action.Tool)
		case agents.EventTypeToolResult:
			println("Tool result:", event.Observation)
		case agents.EventTypeFinish:
			println("Agent finished:", event.Observation)
		case agents.EventTypeError:
			println("Error:", event.Error)
		}
	}
}

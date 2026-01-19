package main

import (
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/core/chat/stream"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	fmt.Println("=== LangChain-Go Streaming 示例 ===\n")

	// 示例 1: 基础 Token 流式处理
	example1BasicStreaming()

	// 示例 2: 使用 StreamAggregator
	example2Aggregator()

	// 示例 3: 工具调用流式
	example3ToolCalls()

	// 示例 4: 错误处理
	example4ErrorHandling()
}

// 示例 1: 基础 Token 流式处理
func example1BasicStreaming() {
	fmt.Println("## 示例 1: 基础 Token 流式处理")

	// 模拟流式事件
	streamCh := make(chan types.StreamEvent, 10)

	go func() {
		defer close(streamCh)

		// 开始事件
		streamCh <- types.StreamEvent{Type: types.StreamEventStart}

		// 发送 token
		tokens := []string{"Hello", " ", "from", " ", "LangChain", "-", "Go", "!"}
		for i, token := range tokens {
			time.Sleep(50 * time.Millisecond)
			streamCh <- types.NewTokenEvent(token).WithIndex(i)
		}

		// 结束事件
		streamCh <- types.StreamEvent{Type: types.StreamEventEnd, Done: true}
	}()

	// 消费流式事件
	fmt.Print("输出: ")
	for event := range streamCh {
		switch event.Type {
		case types.StreamEventToken:
			fmt.Print(event.Token)
		case types.StreamEventEnd:
			fmt.Println()
		}
	}

	fmt.Println()
}

// 示例 2: 使用 StreamAggregator
func example2Aggregator() {
	fmt.Println("## 示例 2: 使用 StreamAggregator 聚合流")

	// 创建聚合器
	aggregator := stream.NewStreamAggregator()

	// 模拟流式事件
	streamCh := make(chan types.StreamEvent, 10)

	go func() {
		defer close(streamCh)

		words := []string{"Go", " ", "is", " ", "awesome", "!"}
		for _, word := range words {
			time.Sleep(30 * time.Millisecond)
			streamCh <- types.NewTokenEvent(word)
		}
	}()

	// 聚合事件并实时显示
	fmt.Print("流式输出: ")
	for event := range streamCh {
		aggregator.Add(event)
		fmt.Print(event.Token)
	}
	fmt.Println()

	// 获取最终结果
	message, err := aggregator.GetResult()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Printf("最终消息: %s\n", message.Content)
	fmt.Printf("事件数量: %d\n", aggregator.GetEventCount())
	fmt.Println()
}

// 示例 3: 工具调用流式
func example3ToolCalls() {
	fmt.Println("## 示例 3: 工具调用流式处理")

	aggregator := stream.NewStreamAggregator()

	// 模拟包含工具调用的流
	streamCh := make(chan types.StreamEvent, 10)

	go func() {
		defer close(streamCh)

		// 发送思考过程
		streamCh <- types.NewTokenEvent("我需要查询天气信息...")

		time.Sleep(100 * time.Millisecond)

		// 发送工具调用
		toolCall := &types.ToolCall{
			ID:   "call_weather_001",
			Type: "function",
			Function: types.FunctionCall{
				Name:      "get_weather",
				Arguments: `{"city":"北京","unit":"celsius"}`,
			},
		}
		streamCh <- types.NewToolCallEvent(toolCall)

		time.Sleep(100 * time.Millisecond)

		// 工具结果
		streamCh <- types.StreamEvent{
			Type:       types.StreamEventToolResult,
			ToolResult: `{"temperature":22,"condition":"晴朗"}`,
		}

		// 最终回复
		streamCh <- types.NewTokenEvent("北京今天天气晴朗，温度22°C。")
	}()

	// 处理流
	for event := range streamCh {
		aggregator.Add(event)

		switch event.Type {
		case types.StreamEventToken:
			fmt.Printf("Token: %s\n", event.Token)
		case types.StreamEventToolCall:
			fmt.Printf("工具调用: %s(%s)\n", 
				event.ToolCall.Function.Name,
				event.ToolCall.Function.Arguments)
		case types.StreamEventToolResult:
			fmt.Printf("工具结果: %s\n", event.ToolResult)
		}
	}

	// 显示最终结果
	fmt.Printf("\n工具调用数量: %d\n", len(aggregator.GetToolCalls()))
	fmt.Println()
}

// 示例 4: 错误处理
func example4ErrorHandling() {
	fmt.Println("## 示例 4: 错误处理")

	aggregator := stream.NewStreamAggregator()

	// 模拟包含错误的流
	streamCh := make(chan types.StreamEvent, 10)

	go func() {
		defer close(streamCh)

		// 发送一些正常 token
		streamCh <- types.NewTokenEvent("Processing...")

		time.Sleep(50 * time.Millisecond)

		// 发送错误
		streamCh <- types.NewErrorEvent(fmt.Errorf("API rate limit exceeded"))
	}()

	// 处理流
	for event := range streamCh {
		if err := aggregator.Add(event); err != nil {
			fmt.Printf("❌ 错误: %v\n", err)
			break
		}

		if event.IsToken() {
			fmt.Printf("✓ Token: %s\n", event.Token)
		}
	}

	// 检查聚合器状态
	if aggregator.HasError() {
		fmt.Printf("聚合器检测到错误: %v\n", aggregator.GetError())
	}

	fmt.Println()
}

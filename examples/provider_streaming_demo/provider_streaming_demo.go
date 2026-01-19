package main

import (
	"fmt"
	"os"

	"github.com/zhucl121/langchain-go/core/chat/stream"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	fmt.Println("=== LangChain-Go Provider Streaming 示例 ===\n")

	// 示例 1: 模拟 Provider Streaming
	example1ProviderStreaming()

	// 示例 2: SSE 格式输出
	example2SSEOutput()

	// 示例 3: 实时聚合和最终结果
	example3AggregationWithFinal()
}

// 示例 1: 模拟不同 Provider 的流式输出
func example1ProviderStreaming() {
	fmt.Println("## 示例 1: 模拟多 Provider Streaming")

	providers := []struct {
		name   string
		tokens []string
	}{
		{
			name:   "OpenAI",
			tokens: []string{"Hello", " from", " OpenAI", " GPT-4"},
		},
		{
			name:   "Anthropic",
			tokens: []string{"Greetings", " from", " Claude", " by", " Anthropic"},
		},
		{
			name:   "Gemini",
			tokens: []string{"Hi", " from", " Google", " Gemini"},
		},
		{
			name:   "Ollama",
			tokens: []string{"Welcome", " from", " Ollama", " (local)"},
		},
	}

	for _, provider := range providers {
		fmt.Printf("\n### %s Streaming:\n", provider.name)

		// 模拟流式输出
		streamCh := simulateProviderStream(provider.name, provider.tokens)

		// 使用 StreamAggregator 聚合
		aggregator := stream.NewStreamAggregator()

		fmt.Print("输出: ")
		for event := range streamCh {
			aggregator.Add(event)

			if event.IsToken() {
				fmt.Print(event.Token)
			} else if event.IsError() {
				fmt.Printf("\n❌ 错误: %v\n", event.Error)
				break
			}
		}

		if !aggregator.HasError() {
			message, _ := aggregator.GetResult()
			fmt.Printf("\n完整消息: %s\n", message.Content)
			fmt.Printf("事件数量: %d\n", aggregator.GetEventCount())
		}
	}

	fmt.Println()
}

// 示例 2: SSE 格式输出
func example2SSEOutput() {
	fmt.Println("## 示例 2: SSE 格式输出")

	// 创建 SSE Writer（输出到 stdout）
	sse := stream.NewSSEWriter(os.Stdout)

	// 模拟流式事件
	streamCh := simulateProviderStream("Demo", []string{"SSE", " format", " demo"})

	fmt.Println("\nSSE 输出格式:")
	fmt.Println("---")

	for event := range streamCh {
		if err := sse.WriteEvent(event); err != nil {
			fmt.Printf("SSE 写入错误: %v\n", err)
			break
		}
	}

	fmt.Println("---")
	fmt.Printf("总事件数: %d\n\n", sse.GetEventCount())
}

// 示例 3: 实时聚合和最终结果
func example3AggregationWithFinal() {
	fmt.Println("## 示例 3: 实时聚合 + 最终结果")

	// 模拟长文本流式输出
	tokens := []string{
		"LangChain-Go", " 是", " 一个", " 强大的", " Go", " 语言",
		" LLM", " 应用", " 开发", " 框架", "。",
		" 它", " 支持", " 多种", " LLM", " Provider", ",",
		" 包括", " OpenAI", "、", "Anthropic", "、", "Gemini", " 和", " Ollama", "。",
	}

	streamCh := simulateProviderStream("LangChain-Go", tokens)
	aggregator := stream.NewStreamAggregator()

	fmt.Println("\n### 实时流式输出:")
	fmt.Print("> ")

	for event := range streamCh {
		aggregator.Add(event)

		switch event.Type {
		case types.StreamEventToken:
			fmt.Print(event.Token)

			// 每 5 个 token 显示一次统计
			if aggregator.GetEventCount()%5 == 0 {
				fmt.Printf(" [%d tokens]", aggregator.GetEventCount())
			}

		case types.StreamEventEnd:
			fmt.Println()
		}
	}

	// 获取最终结果
	message, err := aggregator.GetResult()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Printf("\n### 最终统计:\n")
	fmt.Printf("- 总内容长度: %d 字符\n", len(message.Content))
	fmt.Printf("- 总事件数: %d\n", aggregator.GetEventCount())
	fmt.Printf("- 完整内容: %s\n", message.Content)
	fmt.Println()
}

// simulateProviderStream 模拟 Provider 的流式输出
func simulateProviderStream(provider string, tokens []string) <-chan types.StreamEvent {
	out := make(chan types.StreamEvent, 100)

	go func() {
		defer close(out)

		// 开始事件
		out <- types.StreamEvent{
			Type: types.StreamEventStart,
			Metadata: map[string]any{
				"provider": provider,
			},
		}

		// Token 事件
		for i, token := range tokens {
			event := types.NewTokenEvent(token)
			event.Index = i
			event.Metadata = map[string]any{
				"provider": provider,
			}
			out <- event
		}

		// 结束事件
		out <- types.StreamEvent{
			Type: types.StreamEventEnd,
			Done: true,
		}
	}()

	return out
}

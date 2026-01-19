package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zhucl121/langchain-go/core/agents"
	"github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
	fmt.Println("=== Agent Middleware 系统示例 ===\n")

	// 示例 1: 基础 Middleware
	basicMiddlewareExample()

	// 示例 2: 重试 Middleware
	retryMiddlewareExample()

	// 示例 3: 限流 Middleware
	rateLimitMiddlewareExample()

	// 示例 4: 内容审核 Middleware
	contentModerationExample()

	// 示例 5: 缓存 Middleware
	cachingExample()

	// 示例 6: Middleware 链
	middlewareChainExample()
}

// basicMiddlewareExample 基础 Middleware 示例
func basicMiddlewareExample() {
	fmt.Println("1. 基础 Middleware 示例")
	fmt.Println("---")

	// 创建基础 middleware
	middleware := agents.NewBaseAgentMiddleware("TestMiddleware")

	fmt.Printf("Middleware 名称: %s\n", middleware.Name())

	// 测试 BeforeModel
	ctx := context.Background()
	state := &agents.AgentState{
		Input: "测试问题",
		Steps: []agents.AgentStep{},
	}

	newState, err := middleware.BeforeModel(ctx, state)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("BeforeModel 执行成功, 状态: %+v\n", newState.Input)
	}

	fmt.Println()
}

// retryMiddlewareExample 重试 Middleware 示例
func retryMiddlewareExample() {
	fmt.Println("2. 重试 Middleware 示例")
	fmt.Println("---")

	// 创建重试 middleware（最多重试 3 次）
	retryMw := agents.NewRetryMiddleware(3).
		WithDelay(100 * time.Millisecond).
		WithBackoff(2.0)

	fmt.Printf("Middleware: %s\n", retryMw.Name())
	fmt.Println("配置: 最多重试 3 次, 延迟 100ms, 退避系数 2.0")

	ctx := context.Background()
	state := &agents.AgentState{
		Input: "测试问题",
		Steps: []agents.AgentStep{},
	}

	// 模拟错误
	testErr := fmt.Errorf("模拟的错误")

	// 第 1 次重试
	shouldRetry, err := retryMw.OnError(ctx, state, testErr)
	fmt.Printf("第 1 次错误 - 是否重试: %v, 错误: %v\n", shouldRetry, err)

	// 第 2 次重试
	shouldRetry, err = retryMw.OnError(ctx, state, testErr)
	fmt.Printf("第 2 次错误 - 是否重试: %v\n", shouldRetry)

	// 第 3 次重试
	shouldRetry, err = retryMw.OnError(ctx, state, testErr)
	fmt.Printf("第 3 次错误 - 是否重试: %v\n", shouldRetry)

	// 第 4 次（超过最大重试）
	shouldRetry, err = retryMw.OnError(ctx, state, testErr)
	fmt.Printf("第 4 次错误 - 是否重试: %v (已达到最大重试次数)\n", shouldRetry)

	fmt.Println()
}

// rateLimitMiddlewareExample 限流 Middleware 示例
func rateLimitMiddlewareExample() {
	fmt.Println("3. 限流 Middleware 示例")
	fmt.Println("---")

	// 创建限流 middleware（每秒最多 2 次请求）
	rateLimitMw := agents.NewRateLimitMiddleware(2, time.Second)

	fmt.Printf("Middleware: %s\n", rateLimitMw.Name())
	fmt.Println("配置: 每秒最多 2 次请求")

	ctx := context.Background()
	state := &agents.AgentState{Input: "测试"}

	start := time.Now()

	// 前两次请求应该立即通过
	for i := 1; i <= 2; i++ {
		_, err := rateLimitMw.BeforeModel(ctx, state)
		if err != nil {
			fmt.Printf("请求 %d 失败: %v\n", i, err)
		} else {
			elapsed := time.Since(start)
			fmt.Printf("请求 %d 通过 (耗时: %v)\n", i, elapsed)
		}
	}

	// 第三次请求应该被限流
	fmt.Println("第三次请求（应该被限流，需要等待）...")
	_, err := rateLimitMw.BeforeModel(ctx, state)
	if err != nil {
		fmt.Printf("请求 3 失败: %v\n", err)
	} else {
		elapsed := time.Since(start)
		fmt.Printf("请求 3 通过 (耗时: %v)\n", elapsed)
	}

	fmt.Println()
}

// contentModerationExample 内容审核 Middleware 示例
func contentModerationExample() {
	fmt.Println("4. 内容审核 Middleware 示例")
	fmt.Println("---")

	// 创建内容审核 middleware
	moderationMw := agents.NewContentModerationMiddleware([]string{
		"敏感词",
		"禁用词",
		"不当内容",
	}).WithCaseSensitive(false).
		OnViolation(func(ctx context.Context, violationType string, content string) error {
		fmt.Printf("[警告] 检测到违规内容 (%s)\n", violationType)
		return nil
	})

	fmt.Printf("Middleware: %s\n", moderationMw.Name())
	fmt.Println("配置: 禁用词 ['敏感词', '禁用词', '不当内容']")

	ctx := context.Background()

	// 测试正常输入
	fmt.Println("\n测试 1: 正常输入")
	normalState := &agents.AgentState{Input: "这是正常的问题"}
	_, err := moderationMw.BeforeModel(ctx, normalState)
	if err != nil {
		fmt.Printf("❌ 输入被拒绝: %v\n", err)
	} else {
		fmt.Println("✅ 输入通过审核")
	}

	// 测试包含敏感词的输入
	fmt.Println("\n测试 2: 包含敏感词的输入")
	badState := &agents.AgentState{Input: "这包含敏感词的内容"}
	_, err = moderationMw.BeforeModel(ctx, badState)
	if err != nil {
		fmt.Printf("❌ 输入被拒绝: %v\n", err)
	} else {
		fmt.Println("✅ 输入通过审核")
	}

	// 测试输出审核
	fmt.Println("\n测试 3: 输出审核")
	state := &agents.AgentState{Input: "正常输入"}
	badResponse := types.NewAssistantMessage("这个回复包含禁用词")
	_, err = moderationMw.AfterModel(ctx, state, &badResponse)
	if err != nil {
		fmt.Printf("❌ 输出被拒绝: %v\n", err)
	} else {
		fmt.Println("✅ 输出通过审核")
	}

	fmt.Println()
}

// cachingExample 缓存 Middleware 示例
func cachingExample() {
	fmt.Println("5. 缓存 Middleware 示例")
	fmt.Println("---")

	// 创建缓存 middleware
	cacheMw := agents.NewCachingMiddleware().
		WithTTL(5 * time.Minute).
		WithMaxSize(100)

	fmt.Printf("Middleware: %s\n", cacheMw.Name())
	fmt.Println("配置: TTL=5分钟, 最大缓存数=100")

	ctx := context.Background()
	state := &agents.AgentState{
		Input: "什么是机器学习？",
		Steps: []agents.AgentStep{},
	}

	// 第一次调用（缓存未命中）
	fmt.Println("\n第一次调用（缓存未命中）:")
	newState, _ := cacheMw.BeforeModel(ctx, state)
	if newState.Extra == nil || newState.Extra["cache_hit"] != true {
		fmt.Println("✅ 缓存未命中，将调用 LLM")
	}

	// 模拟 LLM 响应
	response := types.NewAssistantMessage("机器学习是...")
	cacheMw.AfterModel(ctx, state, &response)

	// 第二次调用（缓存命中）
	fmt.Println("\n第二次调用（缓存命中）:")
	newState2, _ := cacheMw.BeforeModel(ctx, state)
	if newState2.Extra != nil && newState2.Extra["cache_hit"] == true {
		fmt.Println("✅ 缓存命中！跳过 LLM 调用")
		if cachedResp, ok := newState2.Extra["cached_response"].(*types.Message); ok {
			fmt.Printf("缓存的响应: %s\n", cachedResp.Content)
		}
	}

	// 显示统计
	hits, misses, hitRate := cacheMw.GetStats()
	fmt.Printf("\n缓存统计: 命中=%d, 未命中=%d, 命中率=%.2f%%\n", hits, misses, hitRate)

	fmt.Println()
}

// middlewareChainExample Middleware 链示例
func middlewareChainExample() {
	fmt.Println("6. Middleware 链示例")
	fmt.Println("---")

	// 创建多个 middleware
	loggingMw := agents.NewLoggingAgentMiddleware().
		WithLogger(func(level, message string, fields map[string]any) {
		fmt.Printf("[%s] %s\n", level, message)
	})

	retryMw := agents.NewRetryMiddleware(2)

	moderationMw := agents.NewContentModerationMiddleware([]string{"敏感词"})

	// 创建 middleware 链
	chain := agents.NewAgentMiddlewareChain(
		loggingMw,
		moderationMw,
		retryMw,
	)

	fmt.Printf("Middleware 链包含 %d 个 middleware:\n", len(chain.GetMiddlewares()))
	for i, mw := range chain.GetMiddlewares() {
		fmt.Printf("%d. %s\n", i+1, mw.Name())
	}

	// 测试链式调用
	fmt.Println("\n执行 Middleware 链:")
	ctx := context.Background()
	state := &agents.AgentState{Input: "测试问题"}

	newState, err := chain.BeforeModel(ctx, state)
	if err != nil {
		fmt.Printf("Middleware 链执行失败: %v\n", err)
	} else {
		fmt.Printf("Middleware 链执行成功\n")
		fmt.Printf("最终状态: %+v\n", newState.Input)
	}

	fmt.Println()
}

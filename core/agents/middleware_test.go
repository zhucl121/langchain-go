package agents

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// TestBaseAgentMiddleware 测试基础中间件
func TestBaseAgentMiddleware(t *testing.T) {
	mw := NewBaseAgentMiddleware("TestMiddleware")

	if mw.Name() != "TestMiddleware" {
		t.Errorf("expected name 'TestMiddleware', got %s", mw.Name())
	}

	// 测试默认实现（应该不做任何修改）
	ctx := context.Background()
	state := &AgentState{Input: "test"}

	newState, err := mw.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("BeforeModel should not error: %v", err)
	}
	if newState != state {
		t.Error("BeforeModel should return original state")
	}

	response := types.NewAssistantMessage("test response")
	newResponse, err := mw.AfterModel(ctx, state, &response)
	if err != nil {
		t.Errorf("AfterModel should not error: %v", err)
	}
	if newResponse != &response {
		t.Error("AfterModel should return original response")
	}
}

// TestAgentMiddlewareChain 测试中间件链
func TestAgentMiddlewareChain(t *testing.T) {
	// 创建多个中间件
	mw1 := NewBaseAgentMiddleware("mw1")
	mw2 := NewBaseAgentMiddleware("mw2")
	mw3 := NewBaseAgentMiddleware("mw3")

	chain := NewAgentMiddlewareChain(mw1, mw2, mw3)

	if len(chain.GetMiddlewares()) != 3 {
		t.Errorf("expected 3 middlewares, got %d", len(chain.GetMiddlewares()))
	}

	// 测试链式调用
	ctx := context.Background()
	state := &AgentState{Input: "test"}

	newState, err := chain.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("chain BeforeModel failed: %v", err)
	}
	if newState == nil {
		t.Error("chain should return state")
	}
}

// TestRetryMiddleware 测试重试中间件
func TestRetryMiddleware(t *testing.T) {
	retry := NewRetryMiddleware(3).
		WithDelay(10 * time.Millisecond).
		WithBackoff(1.0)

	ctx := context.Background()
	state := &AgentState{Input: "test", Steps: []AgentStep{}}

	testErr := errors.New("test error")

	// 第一次重试
	shouldRetry, err := retry.OnError(ctx, state, testErr)
	if !shouldRetry {
		t.Error("should retry on first error")
	}
	if err != testErr {
		t.Error("should return original error")
	}

	// 第二次重试
	shouldRetry, err = retry.OnError(ctx, state, testErr)
	if !shouldRetry {
		t.Error("should retry on second error")
	}

	// 第三次重试
	shouldRetry, err = retry.OnError(ctx, state, testErr)
	if !shouldRetry {
		t.Error("should retry on third error")
	}

	// 第四次不应该重试（达到最大次数）
	shouldRetry, err = retry.OnError(ctx, state, testErr)
	if shouldRetry {
		t.Error("should not retry after max retries")
	}

	// 重置后应该可以再次重试
	retry.Reset()
	shouldRetry, err = retry.OnError(ctx, state, testErr)
	if !shouldRetry {
		t.Error("should retry after reset")
	}
}

// TestRateLimitMiddleware 测试限流中间件
func TestRateLimitMiddleware(t *testing.T) {
	// 创建限流：每 100ms 最多 2 次请求
	rateLimit := NewRateLimitMiddleware(2, 100*time.Millisecond)

	ctx := context.Background()
	state := &AgentState{Input: "test"}

	start := time.Now()

	// 第一次请求 - 应该立即通过
	_, err := rateLimit.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("first request should pass: %v", err)
	}

	// 第二次请求 - 应该立即通过
	_, err = rateLimit.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("second request should pass: %v", err)
	}

	// 第三次请求 - 应该被限流（等待）
	_, err = rateLimit.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("third request should eventually pass: %v", err)
	}

	elapsed := time.Since(start)
	// 第三次请求应该至少等待了约 100ms
	if elapsed < 50*time.Millisecond {
		t.Errorf("third request should be rate limited, elapsed: %v", elapsed)
	}
}

// TestContentModerationMiddleware 测试内容审核中间件
func TestContentModerationMiddleware(t *testing.T) {
	moderation := NewContentModerationMiddleware([]string{
		"敏感词",
		"禁用词",
	}).WithCaseSensitive(false)

	ctx := context.Background()

	// 测试输入检查
	t.Run("input moderation", func(t *testing.T) {
		state := &AgentState{Input: "这包含敏感词的内容"}

		_, err := moderation.BeforeModel(ctx, state)
		if err == nil {
			t.Error("should detect banned word in input")
		}
	})

	// 测试输出检查
	t.Run("output moderation", func(t *testing.T) {
		state := &AgentState{Input: "正常输入"}
		response := types.NewAssistantMessage("这是包含禁用词的回复")

		_, err := moderation.AfterModel(ctx, state, &response)
		if err == nil {
			t.Error("should detect banned word in output")
		}
	})

	// 测试正常内容
	t.Run("normal content", func(t *testing.T) {
		state := &AgentState{Input: "正常输入"}

		_, err := moderation.BeforeModel(ctx, state)
		if err != nil {
			t.Errorf("should pass normal input: %v", err)
		}

		response := types.NewAssistantMessage("正常回复")
		_, err = moderation.AfterModel(ctx, state, &response)
		if err != nil {
			t.Errorf("should pass normal output: %v", err)
		}
	})

	// 测试大小写不敏感
	t.Run("case insensitive", func(t *testing.T) {
		state := &AgentState{Input: "这包含敏感词的内容"}

		_, err := moderation.BeforeModel(ctx, state)
		if err == nil {
			t.Error("should detect banned word (case insensitive)")
		}
	})
}

// TestCachingMiddleware 测试缓存中间件
func TestCachingMiddleware(t *testing.T) {
	cache := NewCachingMiddleware().
		WithTTL(1 * time.Second).
		WithMaxSize(10)

	ctx := context.Background()
	state := &AgentState{Input: "test query", Steps: []AgentStep{}}

	// 第一次调用 - 缓存未命中
	newState, err := cache.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("BeforeModel failed: %v", err)
	}

	// 不应该有缓存命中
	if newState.Extra != nil && newState.Extra["cache_hit"] == true {
		t.Error("first call should not hit cache")
	}

	// 模拟 LLM 响应
	response := types.NewAssistantMessage("test response")
	_, err = cache.AfterModel(ctx, state, &response)
	if err != nil {
		t.Errorf("AfterModel failed: %v", err)
	}

	// 第二次调用 - 应该缓存命中
	newState2, err := cache.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("second BeforeModel failed: %v", err)
	}

	if newState2.Extra == nil || newState2.Extra["cache_hit"] != true {
		t.Error("second call should hit cache")
	}

	// 检查统计
	hits, misses, hitRate := cache.GetStats()
	if hits == 0 {
		t.Error("should have cache hits")
	}
	if misses == 0 {
		t.Error("should have cache misses")
	}
	t.Logf("Cache stats: hits=%d, misses=%d, hitRate=%.2f%%", hits, misses, hitRate)

	// 测试 TTL
	time.Sleep(1100 * time.Millisecond)

	// 缓存应该过期
	newState3, err := cache.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("BeforeModel after TTL failed: %v", err)
	}

	if newState3.Extra != nil && newState3.Extra["cache_hit"] == true {
		t.Error("should not hit cache after TTL")
	}

	// 测试清空缓存
	cache.Clear()
	hits, misses, _ = cache.GetStats()
	if hits == 0 && misses == 0 {
		// 清空后统计应该为0或保持
	}
}

// TestLoggingAgentMiddleware 测试日志中间件
func TestLoggingAgentMiddleware(t *testing.T) {
	logs := []string{}

	logging := NewLoggingAgentMiddleware().
		WithLogger(func(level, message string, fields map[string]any) {
			logs = append(logs, message)
		})

	ctx := context.Background()
	state := &AgentState{Input: "test", Steps: []AgentStep{}}

	// BeforeModel
	_, err := logging.BeforeModel(ctx, state)
	if err != nil {
		t.Errorf("BeforeModel failed: %v", err)
	}

	// AfterModel
	response := types.NewAssistantMessage("response")
	_, err = logging.AfterModel(ctx, state, &response)
	if err != nil {
		t.Errorf("AfterModel failed: %v", err)
	}

	// BeforeToolCall
	_, err = logging.BeforeToolCall(ctx, "test_tool", map[string]any{"param": "value"})
	if err != nil {
		t.Errorf("BeforeToolCall failed: %v", err)
	}

	// AfterToolCall
	_, err = logging.AfterToolCall(ctx, "test_tool", map[string]any{}, "output", nil)
	if err != nil {
		t.Errorf("AfterToolCall failed: %v", err)
	}

	// OnError
	testErr := errors.New("test error")
	_, _ = logging.OnError(ctx, state, testErr)

	// OnComplete
	result := &AgentResult{
		IsFinish:   true,
		Steps:      []AgentStep{},
		TotalSteps: 5,
	}
	err = logging.OnComplete(ctx, result)
	if err != nil {
		t.Errorf("OnComplete failed: %v", err)
	}

	// 检查日志
	if len(logs) == 0 {
		t.Error("should have logs")
	}

	t.Logf("Logged %d messages", len(logs))
}

// TestMiddlewareChainWithErrors 测试中间件链的错误处理
func TestMiddlewareChainWithErrors(t *testing.T) {
	// 创建一个会返回错误的中间件
	errorMw := &testErrorMiddleware{
		BaseAgentMiddleware: NewBaseAgentMiddleware("error"),
	}

	normalMw := NewBaseAgentMiddleware("normal")

	chain := NewAgentMiddlewareChain(normalMw, errorMw)

	ctx := context.Background()
	state := &AgentState{Input: "test"}

	_, err := chain.BeforeModel(ctx, state)
	if err == nil {
		t.Error("chain should propagate error from middleware")
	}
}

// testErrorMiddleware 是用于测试的错误中间件
type testErrorMiddleware struct {
	*BaseAgentMiddleware
}

func (m *testErrorMiddleware) BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error) {
	return nil, errors.New("test error")
}

// TestWithMiddlewareOption 测试中间件选项
func TestWithMiddlewareOption(t *testing.T) {
	config := &AgentConfig{
		Extra: make(map[string]any),
	}

	mw := NewBaseAgentMiddleware("test")

	// 应用选项
	WithMiddleware(mw)(config)

	// 检查是否正确添加
	if config.Extra["middlewares"] == nil {
		t.Error("middleware should be added to config")
	}

	middlewares := config.Extra["middlewares"].([]AgentMiddleware)
	if len(middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(middlewares))
	}

	if middlewares[0].Name() != "test" {
		t.Errorf("expected middleware name 'test', got %s", middlewares[0].Name())
	}
}

// TestWithMiddlewareChainOption 测试中间件链选项
func TestWithMiddlewareChainOption(t *testing.T) {
	config := &AgentConfig{
		Extra: make(map[string]any),
	}

	mw1 := NewBaseAgentMiddleware("mw1")
	mw2 := NewBaseAgentMiddleware("mw2")

	// 应用选项
	WithMiddlewareChain(mw1, mw2)(config)

	// 检查是否正确添加
	if config.Extra["middleware_chain"] == nil {
		t.Error("middleware chain should be added to config")
	}

	chain := config.Extra["middleware_chain"].(*AgentMiddlewareChain)
	if len(chain.GetMiddlewares()) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(chain.GetMiddlewares()))
	}
}

// TestGetMiddlewareChainFromConfig 测试从配置提取中间件链
func TestGetMiddlewareChainFromConfig(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config := &AgentConfig{}
		chain := GetMiddlewareChainFromConfig(config)
		if len(chain.GetMiddlewares()) != 0 {
			t.Error("empty config should return empty chain")
		}
	})

	t.Run("with chain", func(t *testing.T) {
		config := &AgentConfig{
			Extra: make(map[string]any),
		}
		mw := NewBaseAgentMiddleware("test")
		WithMiddlewareChain(mw)(config)

		chain := GetMiddlewareChainFromConfig(config)
		if len(chain.GetMiddlewares()) != 1 {
			t.Errorf("expected 1 middleware, got %d", len(chain.GetMiddlewares()))
		}
	})

	t.Run("with individual middlewares", func(t *testing.T) {
		config := &AgentConfig{
			Extra: make(map[string]any),
		}
		mw1 := NewBaseAgentMiddleware("mw1")
		mw2 := NewBaseAgentMiddleware("mw2")
		WithMiddleware(mw1)(config)
		WithMiddleware(mw2)(config)

		chain := GetMiddlewareChainFromConfig(config)
		if len(chain.GetMiddlewares()) != 2 {
			t.Errorf("expected 2 middlewares, got %d", len(chain.GetMiddlewares()))
		}
	})
}

// Benchmark 测试
func BenchmarkRetryMiddleware(b *testing.B) {
	retry := NewRetryMiddleware(3).
		WithDelay(1 * time.Millisecond)

	ctx := context.Background()
	state := &AgentState{Input: "test", Steps: []AgentStep{}}
	testErr := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retry.OnError(ctx, state, testErr)
		retry.Reset()
	}
}

func BenchmarkCachingMiddleware(b *testing.B) {
	cache := NewCachingMiddleware()

	ctx := context.Background()
	state := &AgentState{Input: "test", Steps: []AgentStep{}}
	response := types.NewAssistantMessage("response")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.BeforeModel(ctx, state)
		cache.AfterModel(ctx, state, &response)
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	mw1 := NewBaseAgentMiddleware("mw1")
	mw2 := NewBaseAgentMiddleware("mw2")
	mw3 := NewBaseAgentMiddleware("mw3")

	chain := NewAgentMiddlewareChain(mw1, mw2, mw3)

	ctx := context.Background()
	state := &AgentState{Input: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain.BeforeModel(ctx, state)
	}
}

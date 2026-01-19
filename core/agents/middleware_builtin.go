package agents

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// RetryMiddleware 是重试中间件。
//
// RetryMiddleware 在发生错误时自动重试。
//
// 示例：
//
//	middleware := NewRetryMiddleware(3).
//	    WithDelay(time.Second).
//	    WithBackoff(2.0)
//
type RetryMiddleware struct {
	*BaseAgentMiddleware
	maxRetries int
	delay      time.Duration
	backoff    float64
	retryCount map[string]int
	mu         sync.RWMutex
}

// NewRetryMiddleware 创建重试中间件。
//
// 参数：
//   - maxRetries: 最大重试次数
//
// 返回：
//   - *RetryMiddleware: 重试中间件实例
//
func NewRetryMiddleware(maxRetries int) *RetryMiddleware {
	return &RetryMiddleware{
		BaseAgentMiddleware: NewBaseAgentMiddleware("RetryMiddleware"),
		maxRetries:          maxRetries,
		delay:               time.Second,
		backoff:             2.0,
		retryCount:          make(map[string]int),
	}
}

// WithDelay 设置重试延迟。
func (r *RetryMiddleware) WithDelay(delay time.Duration) *RetryMiddleware {
	r.delay = delay
	return r
}

// WithBackoff 设置退避系数。
func (r *RetryMiddleware) WithBackoff(backoff float64) *RetryMiddleware {
	r.backoff = backoff
	return r
}

// OnError 实现 AgentMiddleware 接口。
func (r *RetryMiddleware) OnError(ctx context.Context, state *AgentState, err error) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 获取当前步骤的重试次数
	stepKey := fmt.Sprintf("step_%d", len(state.Steps))
	currentRetries := r.retryCount[stepKey]

	if currentRetries >= r.maxRetries {
		// 达到最大重试次数
		delete(r.retryCount, stepKey) // 清理
		return false, fmt.Errorf("max retries exceeded (%d): %w", r.maxRetries, err)
	}

	// 增加重试次数
	r.retryCount[stepKey] = currentRetries + 1

	// 计算延迟时间（指数退避）
	multiplier := 1.0
	for i := 0; i < currentRetries; i++ {
		multiplier *= r.backoff
	}
	delay := time.Duration(float64(r.delay) * multiplier)

	// 等待
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(delay):
		// 继续重试
	}

	return true, err
}

// Reset 重置重试计数。
func (r *RetryMiddleware) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retryCount = make(map[string]int)
}

// RateLimitMiddleware 是限流中间件。
//
// RateLimitMiddleware 限制 LLM 调用频率。
//
// 示例：
//
//	middleware := NewRateLimitMiddleware(10, time.Second)  // 每秒最多 10 次
//
type RateLimitMiddleware struct {
	*BaseAgentMiddleware
	maxRequests int
	window      time.Duration
	requests    []time.Time
	mu          sync.Mutex
}

// NewRateLimitMiddleware 创建限流中间件。
//
// 参数：
//   - maxRequests: 时间窗口内的最大请求数
//   - window: 时间窗口
//
// 返回：
//   - *RateLimitMiddleware: 限流中间件实例
//
func NewRateLimitMiddleware(maxRequests int, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		BaseAgentMiddleware: NewBaseAgentMiddleware("RateLimitMiddleware"),
		maxRequests:         maxRequests,
		window:              window,
		requests:            make([]time.Time, 0),
	}
}

// BeforeModel 实现 AgentMiddleware 接口。
func (rl *RateLimitMiddleware) BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// 清理过期请求
	cutoff := now.Add(-rl.window)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range rl.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests = validRequests

	// 检查是否超过限流
	if len(rl.requests) >= rl.maxRequests {
		// 计算需要等待的时间
		oldestRequest := rl.requests[0]
		waitTime := rl.window - now.Sub(oldestRequest)

		// 等待
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(waitTime):
			// 等待完成，清理一个请求
			rl.requests = rl.requests[1:]
		}
	}

	// 记录本次请求
	rl.requests = append(rl.requests, now)

	return state, nil
}

// ContentModerationMiddleware 是内容审核中间件。
//
// ContentModerationMiddleware 检查输入和输出是否包含敏感内容。
//
// 示例：
//
//	middleware := NewContentModerationMiddleware([]string{
//	    "敏感词1", "敏感词2",
//	})
//
type ContentModerationMiddleware struct {
	*BaseAgentMiddleware
	bannedWords    []string
	checkInput     bool
	checkOutput    bool
	caseSensitive  bool
	onViolation    func(ctx context.Context, violationType string, content string) error
}

// NewContentModerationMiddleware 创建内容审核中间件。
//
// 参数：
//   - bannedWords: 禁用词列表
//
// 返回：
//   - *ContentModerationMiddleware: 内容审核中间件实例
//
func NewContentModerationMiddleware(bannedWords []string) *ContentModerationMiddleware {
	return &ContentModerationMiddleware{
		BaseAgentMiddleware: NewBaseAgentMiddleware("ContentModerationMiddleware"),
		bannedWords:         bannedWords,
		checkInput:          true,
		checkOutput:         true,
		caseSensitive:       false,
	}
}

// WithCaseSensitive 设置是否区分大小写。
func (cm *ContentModerationMiddleware) WithCaseSensitive(sensitive bool) *ContentModerationMiddleware {
	cm.caseSensitive = sensitive
	return cm
}

// WithCheckInput 设置是否检查输入。
func (cm *ContentModerationMiddleware) WithCheckInput(check bool) *ContentModerationMiddleware {
	cm.checkInput = check
	return cm
}

// WithCheckOutput 设置是否检查输出。
func (cm *ContentModerationMiddleware) WithCheckOutput(check bool) *ContentModerationMiddleware {
	cm.checkOutput = check
	return cm
}

// OnViolation 设置违规回调。
func (cm *ContentModerationMiddleware) OnViolation(callback func(ctx context.Context, violationType string, content string) error) *ContentModerationMiddleware {
	cm.onViolation = callback
	return cm
}

// BeforeModel 实现 AgentMiddleware 接口。
func (cm *ContentModerationMiddleware) BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error) {
	if !cm.checkInput {
		return state, nil
	}

	// 检查输入
	if violation := cm.checkContent(state.Input); violation != "" {
		if cm.onViolation != nil {
			if err := cm.onViolation(ctx, "input", violation); err != nil {
				return nil, err
			}
		}
		return nil, fmt.Errorf("content moderation: input contains banned word: %s", violation)
	}

	return state, nil
}

// AfterModel 实现 AgentMiddleware 接口。
func (cm *ContentModerationMiddleware) AfterModel(ctx context.Context, state *AgentState, response *types.Message) (*types.Message, error) {
	if !cm.checkOutput {
		return response, nil
	}

	// 检查输出
	if violation := cm.checkContent(response.Content); violation != "" {
		if cm.onViolation != nil {
			if err := cm.onViolation(ctx, "output", violation); err != nil {
				return nil, err
			}
		}
		return nil, fmt.Errorf("content moderation: output contains banned word: %s", violation)
	}

	return response, nil
}

// checkContent 检查内容是否包含禁用词。
func (cm *ContentModerationMiddleware) checkContent(content string) string {
	checkStr := content
	if !cm.caseSensitive {
		checkStr = strings.ToLower(content)
	}

	for _, word := range cm.bannedWords {
		searchWord := word
		if !cm.caseSensitive {
			searchWord = strings.ToLower(word)
		}

		if strings.Contains(checkStr, searchWord) {
			return word
		}
	}

	return ""
}

// CachingMiddleware 是缓存中间件。
//
// CachingMiddleware 缓存 LLM 响应以减少重复调用。
//
// 示例：
//
//	middleware := NewCachingMiddleware().
//	    WithTTL(5 * time.Minute).
//	    WithMaxSize(100)
//
type CachingMiddleware struct {
	*BaseAgentMiddleware
	cache    map[string]*cacheEntry
	maxSize  int
	ttl      time.Duration
	mu       sync.RWMutex
	hits     int64
	misses   int64
}

type cacheEntry struct {
	response  *types.Message
	timestamp time.Time
}

// NewCachingMiddleware 创建缓存中间件。
//
// 返回：
//   - *CachingMiddleware: 缓存中间件实例
//
func NewCachingMiddleware() *CachingMiddleware {
	return &CachingMiddleware{
		BaseAgentMiddleware: NewBaseAgentMiddleware("CachingMiddleware"),
		cache:               make(map[string]*cacheEntry),
		maxSize:             1000,
		ttl:                 5 * time.Minute,
	}
}

// WithTTL 设置缓存过期时间。
func (c *CachingMiddleware) WithTTL(ttl time.Duration) *CachingMiddleware {
	c.ttl = ttl
	return c
}

// WithMaxSize 设置最大缓存数量。
func (c *CachingMiddleware) WithMaxSize(size int) *CachingMiddleware {
	c.maxSize = size
	return c
}

// BeforeModel 实现 AgentMiddleware 接口。
func (c *CachingMiddleware) BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error) {
	// 生成缓存键
	key := c.generateKey(state)

	c.mu.RLock()
	entry, exists := c.cache[key]
	c.mu.RUnlock()

	if exists {
		// 检查是否过期
		if time.Since(entry.timestamp) < c.ttl {
			c.mu.Lock()
			c.hits++
			c.mu.Unlock()

			// 缓存命中，直接返回缓存的响应
			// 注意：这里我们需要一种机制来跳过实际的 LLM 调用
			// 我们将缓存的响应存储在 state.Extra 中
			if state.Extra == nil {
				state.Extra = make(map[string]any)
			}
			state.Extra["cached_response"] = entry.response
			state.Extra["cache_hit"] = true
		} else {
			// 过期，删除
			c.mu.Lock()
			delete(c.cache, key)
			c.mu.Unlock()
		}
	}

	c.mu.Lock()
	c.misses++
	c.mu.Unlock()

	return state, nil
}

// AfterModel 实现 AgentMiddleware 接口。
func (c *CachingMiddleware) AfterModel(ctx context.Context, state *AgentState, response *types.Message) (*types.Message, error) {
	// 如果是缓存命中，不需要再次缓存
	if state.Extra != nil && state.Extra["cache_hit"] == true {
		return response, nil
	}

	// 生成缓存键
	key := c.generateKey(state)

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查缓存大小
	if len(c.cache) >= c.maxSize {
		// 简单的 FIFO 策略：删除一个旧条目
		// 在生产环境中，应该使用 LRU 等更好的策略
		for k := range c.cache {
			delete(c.cache, k)
			break
		}
	}

	// 缓存响应
	c.cache[key] = &cacheEntry{
		response:  response,
		timestamp: time.Now(),
	}

	return response, nil
}

// generateKey 生成缓存键。
func (c *CachingMiddleware) generateKey(state *AgentState) string {
	// 简单的键生成策略：基于输入和步骤数
	return fmt.Sprintf("%s_%d", state.Input, len(state.Steps))
}

// GetStats 获取缓存统计。
func (c *CachingMiddleware) GetStats() (hits int64, misses int64, hitRate float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hits = c.hits
	misses = c.misses
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return hits, misses, hitRate
}

// Clear 清空缓存。
func (c *CachingMiddleware) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cacheEntry)
}

// LoggingAgentMiddleware 是 Agent 专用日志中间件。
//
// LoggingAgentMiddleware 记录 Agent 执行的详细日志。
//
type LoggingAgentMiddleware struct {
	*BaseAgentMiddleware
	verbose       bool
	logModelCalls bool
	logToolCalls  bool
	logErrors     bool
	logger        func(level, message string, fields map[string]any)
}

// NewLoggingAgentMiddleware 创建 Agent 日志中间件。
func NewLoggingAgentMiddleware() *LoggingAgentMiddleware {
	return &LoggingAgentMiddleware{
		BaseAgentMiddleware: NewBaseAgentMiddleware("LoggingAgentMiddleware"),
		verbose:             true,
		logModelCalls:       true,
		logToolCalls:        true,
		logErrors:           true,
		logger:              defaultLogger,
	}
}

// defaultLogger 默认日志函数。
func defaultLogger(level, message string, fields map[string]any) {
	fmt.Printf("[%s] %s %v\n", level, message, fields)
}

// WithLogger 设置自定义日志函数。
func (l *LoggingAgentMiddleware) WithLogger(logger func(level, message string, fields map[string]any)) *LoggingAgentMiddleware {
	l.logger = logger
	return l
}

// WithVerbose 设置是否详细输出。
func (l *LoggingAgentMiddleware) WithVerbose(verbose bool) *LoggingAgentMiddleware {
	l.verbose = verbose
	return l
}

// BeforeModel 实现 AgentMiddleware 接口。
func (l *LoggingAgentMiddleware) BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error) {
	if l.logModelCalls {
		l.logger("INFO", "Calling LLM", map[string]any{
			"step":  len(state.Steps),
			"input": state.Input,
		})
	}
	return state, nil
}

// AfterModel 实现 AgentMiddleware 接口。
func (l *LoggingAgentMiddleware) AfterModel(ctx context.Context, state *AgentState, response *types.Message) (*types.Message, error) {
	if l.logModelCalls {
		l.logger("INFO", "LLM response received", map[string]any{
			"step":    len(state.Steps),
			"content": response.Content,
		})
	}
	return response, nil
}

// BeforeToolCall 实现 AgentMiddleware 接口。
func (l *LoggingAgentMiddleware) BeforeToolCall(ctx context.Context, toolName string, toolInput map[string]any) (map[string]any, error) {
	if l.logToolCalls {
		l.logger("INFO", "Calling tool", map[string]any{
			"tool":  toolName,
			"input": toolInput,
		})
	}
	return toolInput, nil
}

// AfterToolCall 实现 AgentMiddleware 接口。
func (l *LoggingAgentMiddleware) AfterToolCall(ctx context.Context, toolName string, toolInput map[string]any, toolOutput string, err error) (string, error) {
	if l.logToolCalls {
		fields := map[string]any{
			"tool":   toolName,
			"output": toolOutput,
		}
		if err != nil {
			fields["error"] = err.Error()
		}
		l.logger("INFO", "Tool execution completed", fields)
	}
	return toolOutput, err
}

// OnError 实现 AgentMiddleware 接口。
func (l *LoggingAgentMiddleware) OnError(ctx context.Context, state *AgentState, err error) (bool, error) {
	if l.logErrors {
		l.logger("ERROR", "Agent error", map[string]any{
			"step":  len(state.Steps),
			"error": err.Error(),
		})
	}
	return false, err
}

// OnComplete 实现 AgentMiddleware 接口。
func (l *LoggingAgentMiddleware) OnComplete(ctx context.Context, result *AgentResult) error {
	l.logger("INFO", "Agent completed", map[string]any{
		"steps":     result.Steps,
		"is_finish": result.IsFinish,
	})
	return nil
}

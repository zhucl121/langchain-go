package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	
	"github.com/zhucl121/langchain-go/core/tools"
)

// RetryConfig 重试配置。
type RetryConfig struct {
	// MaxRetries 最大重试次数 (0 表示不重试)
	MaxRetries int

	// InitialDelay 初始延迟时间
	InitialDelay time.Duration

	// MaxDelay 最大延迟时间
	MaxDelay time.Duration

	// BackoffFactor 退避因子 (exponential backoff)
	BackoffFactor float64

	// RetryableErrors 可重试的错误类型
	RetryableErrors []error

	// OnRetry 重试回调函数
	OnRetry func(attempt int, err error)
}

// DefaultRetryConfig 返回默认重试配置。
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: nil, // nil 表示重试所有错误
	}
}

// WithRetry 配置 Agent 重试机制。
//
// 参数：
//   - config: 重试配置
//
// 返回：
//   - AgentOption: 配置选项
//
// 示例：
//
//	agent := agents.CreateReActAgent(llm, tools,
//	    agents.WithRetry(agents.RetryConfig{
//	        MaxRetries:    3,
//	        InitialDelay:  time.Second,
//	        BackoffFactor: 2.0,
//	    }),
//	)
//
func WithRetry(config RetryConfig) AgentOption {
	return func(agentConfig *AgentConfig) {
		if agentConfig.Extra == nil {
			agentConfig.Extra = make(map[string]any)
		}
		agentConfig.Extra["retry_config"] = config
	}
}

// RetryExecutor 带重试功能的执行器包装器。
type RetryExecutor struct {
	executor *AgentExecutor
	config   RetryConfig
}

// NewRetryExecutor 创建带重试功能的执行器。
//
// 参数：
//   - executor: Agent 执行器
//   - config: 重试配置
//
// 返回：
//   - *RetryExecutor: 重试执行器
//
func NewRetryExecutor(executor *AgentExecutor, config RetryConfig) *RetryExecutor {
	return &RetryExecutor{
		executor: executor,
		config:   config,
	}
}

// Run 带重试的执行。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
//
func (re *RetryExecutor) Run(ctx context.Context, input string) (*AgentResult, error) {
	var lastErr error
	delay := re.config.InitialDelay

	for attempt := 0; attempt <= re.config.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 执行
		result, err := re.executor.Run(ctx, input)
		
		// 成功则返回
		if err == nil {
			return result, nil
		}

		// 保存错误
		lastErr = err

		// 检查是否可重试
		if !re.isRetryable(err) {
			return result, fmt.Errorf("non-retryable error: %w", err)
		}

		// 达到最大重试次数
		if attempt >= re.config.MaxRetries {
			break
		}

		// 调用重试回调
		if re.config.OnRetry != nil {
			re.config.OnRetry(attempt+1, err)
		}

		// 等待后重试
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// 计算下一次延迟 (exponential backoff)
			delay = time.Duration(float64(delay) * re.config.BackoffFactor)
			if delay > re.config.MaxDelay {
				delay = re.config.MaxDelay
			}
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", re.config.MaxRetries, lastErr)
}

// isRetryable 检查错误是否可重试。
func (re *RetryExecutor) isRetryable(err error) bool {
	// 如果没有配置可重试错误，则所有错误都可重试
	if re.config.RetryableErrors == nil || len(re.config.RetryableErrors) == 0 {
		return true
	}

	// 检查错误是否在可重试列表中
	for _, retryableErr := range re.config.RetryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	return false
}

// Stream 带重试的流式执行。
//
// 注意: 流式执行的重试机制比较复杂，暂不支持
//
func (re *RetryExecutor) Stream(ctx context.Context, input string) <-chan AgentStreamEvent {
	// 流式执行暂不支持重试
	return re.executor.Stream(ctx, input)
}

// RetryableToolExecutor 带重试功能的工具执行器。
type RetryableToolExecutor struct {
	executor *SimplifiedAgentExecutor
	config   RetryConfig
}

// NewRetryableAgentExecutor 创建带重试功能的简化执行器。
//
// 这是一个便捷函数，结合了 SimplifiedAgentExecutor 和重试功能。
//
// 参数：
//   - agent: Agent 实例
//   - agentTools: 工具列表
//   - retryConfig: 重试配置
//   - opts: Agent 选项
//
// 返回：
//   - *RetryExecutor: 带重试的执行器
//
// 示例：
//
//	executor := agents.NewRetryableAgentExecutor(
//	    agent, tools,
//	    agents.RetryConfig{MaxRetries: 3},
//	    agents.WithVerbose(true),
//	)
//	result, _ := executor.Run(ctx, "question")
//
func NewRetryableAgentExecutor(
	agent Agent,
	agentTools []tools.Tool,
	retryConfig RetryConfig,
	opts ...AgentOption,
) *RetryExecutor {
	// 创建简化执行器
	simplifiedExecutor := NewSimplifiedAgentExecutor(agent, agentTools, opts...)
	
	// 获取内部的 AgentExecutor
	executor := simplifiedExecutor.executor
	
	// 创建重试执行器
	return NewRetryExecutor(executor, retryConfig)
}

// ExponentialBackoff 计算指数退避延迟。
//
// 参数：
//   - attempt: 当前重试次数 (从 0 开始)
//   - initialDelay: 初始延迟
//   - maxDelay: 最大延迟
//   - factor: 退避因子
//
// 返回：
//   - time.Duration: 延迟时间
//
func ExponentialBackoff(attempt int, initialDelay, maxDelay time.Duration, factor float64) time.Duration {
	delay := initialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * factor)
		if delay > maxDelay {
			return maxDelay
		}
	}
	return delay
}

// IsTemporaryError 判断是否是临时错误（可重试）。
//
// 参数：
//   - err: 错误
//
// 返回：
//   - bool: 是否可重试
//
func IsTemporaryError(err error) bool {
	// 检查是否实现了 Temporary 接口
	type temporary interface {
		Temporary() bool
	}

	if te, ok := err.(temporary); ok {
		return te.Temporary()
	}

	// 检查常见的临时错误
	errorStrings := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
	}

	errStr := err.Error()
	errStrLower := strings.ToLower(errStr)
	for _, s := range errorStrings {
		if strings.Contains(errStrLower, s) {
			return true
		}
	}

	return false
}

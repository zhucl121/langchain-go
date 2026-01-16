package agents

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// AgentMetrics Agent 指标。
//
// 用于收集 Agent 执行的统计信息。
type AgentMetrics struct {
	// TotalCalls 总调用次数
	TotalCalls int64

	// SuccessfulCalls 成功调用次数
	SuccessfulCalls int64

	// FailedCalls 失败调用次数
	FailedCalls int64

	// TotalDuration 总耗时
	TotalDuration time.Duration

	// AvgDuration 平均耗时
	AvgDuration time.Duration

	// MinDuration 最小耗时
	MinDuration time.Duration

	// MaxDuration 最大耗时
	MaxDuration time.Duration

	// TotalSteps 总步数
	TotalSteps int64

	// AvgSteps 平均步数
	AvgSteps float64

	// ToolUsage 工具使用统计
	ToolUsage map[string]int64

	// ErrorCounts 错误统计
	ErrorCounts map[string]int64

	mu sync.RWMutex
}

// NewAgentMetrics 创建 Agent 指标。
func NewAgentMetrics() *AgentMetrics {
	return &AgentMetrics{
		ToolUsage:   make(map[string]int64),
		ErrorCounts: make(map[string]int64),
	}
}

// RecordCall 记录调用。
//
// 参数：
//   - duration: 耗时
//   - steps: 步数
//   - success: 是否成功
//   - err: 错误 (如果有)
//
func (m *AgentMetrics) RecordCall(duration time.Duration, steps int, success bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 增加调用次数
	atomic.AddInt64(&m.TotalCalls, 1)

	if success {
		atomic.AddInt64(&m.SuccessfulCalls, 1)
	} else {
		atomic.AddInt64(&m.FailedCalls, 1)
		if err != nil {
			m.ErrorCounts[err.Error()]++
		}
	}

	// 记录耗时
	m.TotalDuration += duration
	if m.MinDuration == 0 || duration < m.MinDuration {
		m.MinDuration = duration
	}
	if duration > m.MaxDuration {
		m.MaxDuration = duration
	}

	// 计算平均耗时
	if m.TotalCalls > 0 {
		m.AvgDuration = m.TotalDuration / time.Duration(m.TotalCalls)
	}

	// 记录步数
	m.TotalSteps += int64(steps)
	if m.TotalCalls > 0 {
		m.AvgSteps = float64(m.TotalSteps) / float64(m.TotalCalls)
	}
}

// RecordToolUse 记录工具使用。
//
// 参数：
//   - toolName: 工具名称
//
func (m *AgentMetrics) RecordToolUse(toolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ToolUsage[toolName]++
}

// GetSuccessRate 获取成功率。
//
// 返回：
//   - float64: 成功率 (0-1)
//
func (m *AgentMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalCalls == 0 {
		return 0
	}
	return float64(m.SuccessfulCalls) / float64(m.TotalCalls)
}

// GetMetricsSummary 获取指标摘要。
//
// 返回：
//   - string: 指标摘要
//
func (m *AgentMetrics) GetMetricsSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return fmt.Sprintf(`Agent Metrics:
  Total Calls: %d
  Successful: %d
  Failed: %d
  Success Rate: %.2f%%
  Avg Duration: %v
  Min Duration: %v
  Max Duration: %v
  Avg Steps: %.2f
  Total Steps: %d
`,
		m.TotalCalls,
		m.SuccessfulCalls,
		m.FailedCalls,
		m.GetSuccessRate()*100,
		m.AvgDuration,
		m.MinDuration,
		m.MaxDuration,
		m.AvgSteps,
		m.TotalSteps,
	)
}

// AgentLogger Agent 日志接口。
//
// 用于记录 Agent 执行过程。
type AgentLogger interface {
	// LogStart 记录开始
	LogStart(input string)

	// LogStep 记录步骤
	LogStep(step int, action *AgentAction)

	// LogToolCall 记录工具调用
	LogToolCall(tool string, input map[string]any)

	// LogToolResult 记录工具结果
	LogToolResult(tool string, result any, err error)

	// LogFinish 记录完成
	LogFinish(result *AgentResult)

	// LogError 记录错误
	LogError(err error)
}

// ConsoleLogger 控制台日志器。
type ConsoleLogger struct {
	verbose bool
}

// NewConsoleLogger 创建控制台日志器。
//
// 参数：
//   - verbose: 是否详细输出
//
func NewConsoleLogger(verbose bool) *ConsoleLogger {
	return &ConsoleLogger{
		verbose: verbose,
	}
}

// LogStart 实现 AgentLogger 接口。
func (l *ConsoleLogger) LogStart(input string) {
	fmt.Printf("\n🚀 Agent Started\n")
	fmt.Printf("Input: %s\n", input)
}

// LogStep 实现 AgentLogger 接口。
func (l *ConsoleLogger) LogStep(step int, action *AgentAction) {
	fmt.Printf("\n📍 Step %d\n", step)
	if l.verbose && action != nil {
		fmt.Printf("Action Type: %s\n", action.Type)
		if action.Log != "" {
			fmt.Printf("Thought: %s\n", action.Log)
		}
	}
}

// LogToolCall 实现 AgentLogger 接口。
func (l *ConsoleLogger) LogToolCall(tool string, input map[string]any) {
	fmt.Printf("🔧 Tool Call: %s\n", tool)
	if l.verbose {
		fmt.Printf("   Input: %v\n", input)
	}
}

// LogToolResult 实现 AgentLogger 接口。
func (l *ConsoleLogger) LogToolResult(tool string, result any, err error) {
	if err != nil {
		fmt.Printf("❌ Tool Error: %v\n", err)
	} else {
		fmt.Printf("✅ Tool Result: %v\n", result)
	}
}

// LogFinish 实现 AgentLogger 接口。
func (l *ConsoleLogger) LogFinish(result *AgentResult) {
	fmt.Printf("\n🎉 Agent Finished\n")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Total Steps: %d\n", result.TotalSteps)
	if result.Output != "" {
		fmt.Printf("Output: %s\n", result.Output)
	}
}

// LogError 实现 AgentLogger 接口。
func (l *ConsoleLogger) LogError(err error) {
	fmt.Printf("❌ Error: %v\n", err)
}

// ObservableExecutor 可观测的执行器。
type ObservableExecutor struct {
	executor *AgentExecutor
	metrics  *AgentMetrics
	logger   AgentLogger
}

// NewObservableExecutor 创建可观测的执行器。
//
// 参数：
//   - executor: Agent 执行器
//   - metrics: 指标收集器 (可选)
//   - logger: 日志器 (可选)
//
// 返回：
//   - *ObservableExecutor: 可观测执行器
//
func NewObservableExecutor(executor *AgentExecutor, metrics *AgentMetrics, logger AgentLogger) *ObservableExecutor {
	if metrics == nil {
		metrics = NewAgentMetrics()
	}
	if logger == nil {
		logger = NewConsoleLogger(false)
	}

	return &ObservableExecutor{
		executor: executor,
		metrics:  metrics,
		logger:   logger,
	}
}

// Run 带可观测性的执行。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
//
func (oe *ObservableExecutor) Run(ctx context.Context, input string) (*AgentResult, error) {
	// 记录开始
	oe.logger.LogStart(input)
	startTime := time.Now()

	// 执行
	result, err := oe.executor.Run(ctx, input)

	// 记录指标
	duration := time.Since(startTime)
	steps := 0
	if result != nil {
		steps = result.TotalSteps

		// 记录工具使用
		for _, step := range result.Steps {
			if step.Action != nil && step.Action.Type == ActionToolCall {
				oe.metrics.RecordToolUse(step.Action.Tool)
			}
		}
	}

	success := (err == nil && result != nil && result.Success)
	oe.metrics.RecordCall(duration, steps, success, err)

	// 记录完成或错误
	if err != nil {
		oe.logger.LogError(err)
	} else if result != nil {
		oe.logger.LogFinish(result)
	}

	return result, err
}

// GetMetrics 获取指标。
//
// 返回：
//   - *AgentMetrics: 指标
//
func (oe *ObservableExecutor) GetMetrics() *AgentMetrics {
	return oe.metrics
}

// PrintMetrics 打印指标。
func (oe *ObservableExecutor) PrintMetrics() {
	fmt.Println(oe.metrics.GetMetricsSummary())
}

// WithMetrics 配置指标收集。
//
// 参数：
//   - metrics: 指标收集器
//
// 返回：
//   - AgentOption: 配置选项
//
func WithMetrics(metrics *AgentMetrics) AgentOption {
	return func(config *AgentConfig) {
		if config.Extra == nil {
			config.Extra = make(map[string]any)
		}
		config.Extra["metrics"] = metrics
	}
}

// WithLogger 配置日志器。
//
// 参数：
//   - logger: 日志器
//
// 返回：
//   - AgentOption: 配置选项
//
func WithLogger(logger AgentLogger) AgentOption {
	return func(config *AgentConfig) {
		if config.Extra == nil {
			config.Extra = make(map[string]any)
		}
		config.Extra["logger"] = logger
	}
}

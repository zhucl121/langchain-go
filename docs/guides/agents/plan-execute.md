# Plan-and-Execute Agent 使用指南

## 概述

Plan-and-Execute Agent 是一种高级 Agent 模式，它将复杂任务的处理分为两个阶段：

1. **Plan（规划）**: 使用 LLM 将复杂任务分解为多个可执行的步骤
2. **Execute（执行）**: 按顺序执行每个步骤，每步都可以使用工具和之前步骤的结果

这种模式特别适合需要多步骤推理和执行的复杂任务。

## 核心优势

### 1. 结构化思考
- ✅ 将复杂任务分解为清晰的步骤
- ✅ 提前规划整体执行路径
- ✅ 避免盲目的试错

### 2. 可追踪性
- ✅ 每个步骤都有明确的目标
- ✅ 可以查看完整的执行计划
- ✅ 便于调试和优化

### 3. 动态调整
- ✅ 支持根据执行结果重新规划
- ✅ 自动处理执行失败的情况
- ✅ 灵活适应变化

### 4. 步骤依赖
- ✅ 后续步骤可以使用前面步骤的结果
- ✅ 自动管理步骤间的数据流
- ✅ 构建复杂的执行流程

## 快速开始

### 基础用法

```go
package main

import (
	"context"
	"fmt"
	
	"langchain-go/core/agents"
	"langchain-go/core/chat"
	"langchain-go/core/tools"
)

func main() {
	// 1. 创建 LLM
	llm := chat.NewOpenAI(chat.OpenAIConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4",
	})
	
	// 2. 定义工具
	searchTool := tools.NewFunctionTool(
		"search",
		"Search the internet for information",
		func(ctx context.Context, input map[string]any) (any, error) {
			query := input["input"].(string)
			// 实现搜索逻辑
			return fmt.Sprintf("Search results for: %s", query), nil
		},
	)
	
	calculatorTool := tools.NewFunctionTool(
		"calculator",
		"Perform mathematical calculations",
		func(ctx context.Context, input map[string]any) (any, error) {
			expression := input["input"].(string)
			// 实现计算逻辑
			return fmt.Sprintf("Result: %s", expression), nil
		},
	)
	
	// 3. 创建 Plan-and-Execute Agent
	config := agents.PlanAndExecuteConfig{
		LLM:          llm,
		Tools:        []tools.Tool{searchTool, calculatorTool},
		EnableReplan: false,
		MaxSteps:     10,
		Verbose:      true,
	}
	
	agent := agents.NewPlanAndExecuteAgent(config)
	
	// 4. 创建执行器
	executor := agents.NewExecutor(agent).
		WithMaxSteps(10).
		WithVerbose(true)
	
	// 5. 执行任务
	ctx := context.Background()
	result, err := executor.Execute(ctx, 
		"Search for the population of Tokyo and calculate what 10% of it is")
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// 6. 输出结果
	fmt.Printf("Final Answer: %s\n", result.Output)
	fmt.Printf("Total Steps: %d\n", result.TotalSteps)
	fmt.Printf("Success: %v\n", result.Success)
}
```

## 配置选项

### PlanAndExecuteConfig

```go
type PlanAndExecuteConfig struct {
	// LLM 语言模型（必需）
	LLM chat.ChatModel
	
	// Tools 工具列表（必需）
	Tools []tools.Tool
	
	// PlannerPrompt 自定义规划器提示词（可选）
	PlannerPrompt string
	
	// ExecutorPrompt 自定义执行器提示词（可选）
	ExecutorPrompt string
	
	// EnableReplan 是否启用动态重新规划（默认: false）
	EnableReplan bool
	
	// MaxSteps 最大步骤数（默认: 10）
	MaxSteps int
	
	// Verbose 是否输出详细日志（默认: false）
	Verbose bool
}
```

### 配置说明

#### 1. **基础配置**

```go
config := agents.PlanAndExecuteConfig{
	LLM:      llm,
	Tools:    tools,
	MaxSteps: 10,
}
```

#### 2. **启用详细日志**

```go
config := agents.PlanAndExecuteConfig{
	LLM:     llm,
	Tools:   tools,
	Verbose: true,  // 输出每个步骤的详细信息
}
```

#### 3. **启用动态重新规划**

```go
config := agents.PlanAndExecuteConfig{
	LLM:          llm,
	Tools:        tools,
	EnableReplan: true,  // 当步骤失败时自动重新规划
}
```

#### 4. **自定义提示词**

```go
customPlannerPrompt := `You are an expert planner for data analysis tasks.
Break down the task into clear steps that involve:
1. Data collection
2. Data processing
3. Analysis
4. Reporting`

config := agents.PlanAndExecuteConfig{
	LLM:           llm,
	Tools:         tools,
	PlannerPrompt: customPlannerPrompt,
}
```

## 工作流程详解

### 1. 规划阶段

Agent 首次调用时会创建执行计划：

```
Input: "分析最近一周的销售数据并生成报告"

Plan:
1. 获取最近一周的销售数据
2. 清洗和处理数据
3. 计算关键指标（总销售额、平均值、增长率）
4. 生成可视化图表
5. 编写分析报告
```

### 2. 执行阶段

逐步执行计划中的每个步骤：

```
Step 1: 获取最近一周的销售数据
Tool: database_query
Result: 获取到 1,234 条销售记录

Step 2: 清洗和处理数据
Tool: data_processor
Result: 清洗后有效数据 1,180 条

Step 3: 计算关键指标
Tool: calculator
Result: 总销售额: $125,000, 平均值: $106, 增长率: +12%

... (继续执行)
```

### 3. 重新规划（可选）

当 `EnableReplan` 为 true 且某步失败时：

```
Step 1: 连接数据库
Error: 数据库连接失败

[Replan Triggered]

Updated Plan:
1. 使用备用数据源（CSV 文件）
2. 加载并验证数据
3. 继续分析流程
```

## 使用场景

### 1. 数据分析任务

```go
result, err := executor.Execute(ctx, 
	"分析 Q3 销售数据，找出表现最好的产品类别，并给出改进建议")
```

**自动生成的计划可能包括**:
1. 获取 Q3 销售数据
2. 按产品类别分组统计
3. 计算各类别的增长率
4. 识别 Top 3 类别
5. 分析成功因素
6. 生成改进建议

### 2. 研究和调查

```go
result, err := executor.Execute(ctx, 
	"研究人工智能在医疗领域的最新应用，并总结主要趋势")
```

**自动生成的计划可能包括**:
1. 搜索 AI 医疗应用的最新论文
2. 搜索行业新闻和报告
3. 整理应用案例
4. 分类和归纳
5. 识别主要趋势
6. 编写总结报告

### 3. 问题诊断和解决

```go
result, err := executor.Execute(ctx, 
	"我的网站访问速度很慢，帮我诊断问题并提供解决方案")
```

**自动生成的计划可能包括**:
1. 检查服务器性能指标
2. 分析网络延迟
3. 检查数据库查询性能
4. 审查前端资源加载
5. 识别瓶颈
6. 提供优化建议

### 4. 多步骤工作流

```go
result, err := executor.Execute(ctx, 
	"从 API 获取数据，转换格式，保存到数据库，然后发送确认邮件")
```

**自动生成的计划可能包括**:
1. 调用 API 获取数据
2. 验证数据完整性
3. 转换为目标格式
4. 保存到数据库
5. 生成确认消息
6. 发送邮件通知

## 高级用法

### 1. 带状态追踪的执行

```go
executor := agents.NewExecutor(agent).
	WithMaxSteps(15).
	WithVerbose(true)

result, err := executor.Stream(ctx, input, func(step agents.AgentStep) error {
	// 每完成一步就调用此回调
	fmt.Printf("✓ Step %d completed\n", len(result.Steps)+1)
	fmt.Printf("  Action: %s\n", step.Action.Log)
	fmt.Printf("  Result: %s\n", step.Observation)
	
	// 可以在这里记录日志、更新 UI 等
	return nil
})
```

### 2. 批量任务处理

```go
tasks := []string{
	"分析产品 A 的用户反馈",
	"分析产品 B 的用户反馈",
	"分析产品 C 的用户反馈",
}

results, err := executor.Batch(ctx, tasks)

for i, result := range results {
	if result.Success {
		fmt.Printf("Task %d: %s\n", i+1, result.Output)
	} else {
		fmt.Printf("Task %d failed: %v\n", i+1, result.Error)
	}
}
```

### 3. 自定义规划逻辑

```go
// 创建自定义 Planner
plannerConfig := agents.PlannerConfig{
	LLM: llm,
	Prompt: `You are a project management expert.
Break down the project into phases:
1. Planning & Requirements
2. Design & Architecture  
3. Implementation
4. Testing & QA
5. Deployment & Monitoring

For each phase, list specific tasks.`,
	MaxSteps: 20,
}

planner := agents.NewPlanner(plannerConfig)

// 在 PlanAndExecuteAgent 中使用
// (需要修改代码以支持注入自定义 Planner)
```

### 4. 错误处理和重试

```go
executor := agents.NewExecutor(agent).
	WithMaxSteps(10).
	WithVerbose(true)

// 添加错误处理中间件
executor.WithMiddleware(middleware.NewErrorHandler(
	middleware.ErrorHandlerConfig{
		MaxRetries: 3,
		RetryDelay: time.Second * 2,
		OnError: func(err error) {
			fmt.Printf("Step failed: %v, retrying...\n", err)
		},
	},
))

result, err := executor.Execute(ctx, input)
```

## 执行结果

### AgentResult 结构

```go
type AgentResult struct {
	// Output 最终输出
	Output string
	
	// Steps 执行步骤历史
	Steps []AgentStep
	
	// TotalSteps 总步数
	TotalSteps int
	
	// Success 是否成功
	Success bool
	
	// Error 错误（如果有）
	Error error
}
```

### 查看执行历史

```go
result, err := executor.Execute(ctx, input)

// 查看所有步骤
for i, step := range result.Steps {
	fmt.Printf("\n=== Step %d ===\n", i+1)
	fmt.Printf("Action: %s\n", step.Action.Log)
	
	if step.Action.Type == agents.ActionToolCall {
		fmt.Printf("Tool: %s\n", step.Action.Tool)
		fmt.Printf("Input: %v\n", step.Action.ToolInput)
	}
	
	fmt.Printf("Observation: %s\n", step.Observation)
	
	if step.Error != nil {
		fmt.Printf("Error: %v\n", step.Error)
	}
}
```

## 最佳实践

### 1. 任务描述要清晰

✅ **好的描述**:
```go
"分析2024年Q1的销售数据，计算同比增长率，并生成包含趋势图的报告"
```

❌ **不好的描述**:
```go
"看一下销售"
```

### 2. 提供充足的工具

确保 Agent 有足够的工具来完成任务：

```go
tools := []tools.Tool{
	databaseTool,      // 数据访问
	calculatorTool,    // 计算
	chartGeneratorTool, // 可视化
	emailTool,         // 通知
}
```

### 3. 合理设置 MaxSteps

- 简单任务: `MaxSteps: 5-7`
- 中等复杂度: `MaxSteps: 10-15`
- 复杂任务: `MaxSteps: 20-30`

### 4. 在开发时启用 Verbose

```go
config := agents.PlanAndExecuteConfig{
	LLM:     llm,
	Tools:   tools,
	Verbose: true,  // 开发时启用
}
```

### 5. 为生产环境启用 Replan

```go
config := agents.PlanAndExecuteConfig{
	LLM:          llm,
	Tools:        tools,
	EnableReplan: true,  // 生产环境建议启用
}
```

### 6. 监控执行时间

```go
start := time.Now()
result, err := executor.Execute(ctx, input)
duration := time.Since(start)

fmt.Printf("Execution time: %v\n", duration)
fmt.Printf("Steps executed: %d\n", result.TotalSteps)
fmt.Printf("Average time per step: %v\n", duration/time.Duration(result.TotalSteps))
```

## 性能优化

### 1. 减少步骤数量

通过更好的提示词引导 Planner 生成更精简的计划：

```go
plannerPrompt := `Create a concise plan with 3-5 steps maximum.
Combine related operations into single steps when possible.`
```

### 2. 使用更快的 LLM

```go
// 规划阶段使用强大模型
plannerLLM := chat.NewOpenAI(chat.OpenAIConfig{
	Model: "gpt-4",
})

// 执行阶段使用更快的模型
executorLLM := chat.NewOpenAI(chat.OpenAIConfig{
	Model: "gpt-3.5-turbo",
})
```

### 3. 缓存工具结果

为频繁使用的工具添加缓存：

```go
cachedTool := tools.NewCachedTool(originalTool, cache.NewLRUCache(100))
```

### 4. 并行执行独立步骤

(未来版本可能支持)

## 故障排查

### 问题 1: Agent 无法生成有效计划

**症状**: Plan 为空或只有一个步骤

**解决方案**:
1. 检查任务描述是否清晰
2. 自定义 PlannerPrompt
3. 尝试使用更强大的 LLM

### 问题 2: 执行超时

**症状**: 达到 MaxSteps 限制

**解决方案**:
1. 增加 MaxSteps
2. 简化任务描述
3. 检查工具是否正常工作

### 问题 3: 步骤执行失败

**症状**: 某个步骤返回错误

**解决方案**:
1. 启用 EnableReplan 自动恢复
2. 检查工具实现
3. 添加错误处理中间件

### 问题 4: 计划不合理

**症状**: 生成的步骤顺序不对或缺少关键步骤

**解决方案**:
1. 自定义 PlannerPrompt 提供更多上下文
2. 在任务描述中明确关键步骤
3. 使用更好的 LLM

## 与其他 Agent 类型的比较

| 特性 | Plan-and-Execute | ReAct | Tool Calling |
|------|-----------------|-------|--------------|
| 任务分解 | ✅ 提前规划 | ❌ 逐步探索 | ❌ 即时响应 |
| 可追踪性 | ✅ 完整计划 | ⚠️ 历史记录 | ❌ 单次调用 |
| 复杂任务 | ✅ 优秀 | ⚠️ 一般 | ❌ 不适合 |
| 执行效率 | ⚠️ 较慢（多次LLM调用） | ⚠️ 一般 | ✅ 快速 |
| 动态调整 | ✅ 支持Replan | ✅ 自然支持 | ❌ 不支持 |
| 使用场景 | 多步骤工作流 | 探索式任务 | 简单工具调用 |

## API 参考

### 核心类型

#### PlanAndExecuteAgent

```go
type PlanAndExecuteAgent struct {
	// 内部字段...
}

func NewPlanAndExecuteAgent(config PlanAndExecuteConfig) *PlanAndExecuteAgent
func (pea *PlanAndExecuteAgent) Plan(ctx context.Context, input string, history []AgentStep) (*AgentAction, error)
func (pea *PlanAndExecuteAgent) GetType() AgentType
func (pea *PlanAndExecuteAgent) GetTools() []tools.Tool
```

#### Planner

```go
type Planner struct {
	// 内部字段...
}

func NewPlanner(config PlannerConfig) *Planner
func (p *Planner) CreatePlan(ctx context.Context, input string) (*Plan, error)
func (p *Planner) Replan(ctx context.Context, input string, currentPlan *Plan, history []AgentStep) (*Plan, error)
```

#### StepExecutor

```go
type StepExecutor struct {
	// 内部字段...
}

func NewStepExecutor(config StepExecutorConfig) *StepExecutor
func (se *StepExecutor) ExecuteStep(ctx context.Context, step PlanStep, originalInput string, previousResults map[string]string) (*AgentAction, error)
```

#### Plan & PlanStep

```go
type Plan struct {
	Steps         []PlanStep
	OriginalInput string
}

type PlanStep struct {
	ID           string
	Description  string
	Dependencies []string
	ToolName     string  // 建议使用的工具（可选）
}
```

## 总结

Plan-and-Execute Agent 是处理复杂多步骤任务的强大工具。它通过：

1. ✅ **结构化分解** - 将复杂任务分解为清晰步骤
2. ✅ **有序执行** - 按计划逐步执行
3. ✅ **动态调整** - 根据结果重新规划
4. ✅ **结果聚合** - 综合所有步骤生成最终答案

特别适合数据分析、研究调查、问题诊断等需要多步骤推理的场景。

---

**相关文档**:
- [Agent 系统概述](./Phase3-Agent-System-Summary.md)
- [ReAct Agent 指南](./Phase3-Complete-Summary.md#react-agent)
- [Tool 开发指南](./M17-M18-Tools-Summary.md)

**版本**: v1.0  
**最后更新**: 2026-01-15

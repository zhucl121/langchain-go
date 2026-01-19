package agents

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// AgentMiddleware 是 Agent 专用中间件接口。
//
// AgentMiddleware 提供细粒度的钩子，在 Agent 执行的不同阶段插入自定义逻辑。
//
// 示例：
//
//	middleware := NewRetryMiddleware(3)
//	agent := CreateAgentWithMiddleware(
//	    llm,
//	    tools,
//	    middleware,
//	)
//
type AgentMiddleware interface {
	// BeforeModel 在调用 LLM 之前执行。
	//
	// 参数：
	//   - ctx: 上下文
	//   - state: Agent 状态
	//
	// 返回：
	//   - *AgentState: 修改后的状态（如果需要修改）
	//   - error: 错误（如果返回错误，将中止执行）
	//
	BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error)

	// AfterModel 在 LLM 响应后执行。
	//
	// 参数：
	//   - ctx: 上下文
	//   - state: Agent 状态
	//   - response: LLM 响应
	//
	// 返回：
	//   - *types.Message: 修改后的响应（如果需要修改）
	//   - error: 错误（如果返回错误，将中止执行）
	//
	AfterModel(ctx context.Context, state *AgentState, response *types.Message) (*types.Message, error)

	// OnError 当发生错误时执行。
	//
	// 参数：
	//   - ctx: 上下文
	//   - state: Agent 状态
	//   - err: 错误
	//
	// 返回：
	//   - bool: 是否应该重试
	//   - error: 新的错误（如果需要修改错误）
	//
	OnError(ctx context.Context, state *AgentState, err error) (shouldRetry bool, newErr error)

	// BeforeToolCall 在调用工具之前执行。
	//
	// 参数：
	//   - ctx: 上下文
	//   - toolName: 工具名称
	//   - toolInput: 工具输入
	//
	// 返回：
	//   - map[string]any: 修改后的输入（如果需要修改）
	//   - error: 错误（如果返回错误，将中止工具调用）
	//
	BeforeToolCall(ctx context.Context, toolName string, toolInput map[string]any) (map[string]any, error)

	// AfterToolCall 在工具调用后执行。
	//
	// 参数：
	//   - ctx: 上下文
	//   - toolName: 工具名称
	//   - toolInput: 工具输入
	//   - toolOutput: 工具输出
	//   - err: 工具执行错误
	//
	// 返回：
	//   - string: 修改后的输出（如果需要修改）
	//   - error: 新的错误（如果需要修改）
	//
	AfterToolCall(ctx context.Context, toolName string, toolInput map[string]any, toolOutput string, err error) (string, error)

	// OnComplete 当 Agent 完成执行时调用。
	//
	// 参数：
	//   - ctx: 上下文
	//   - result: Agent 结果
	//
	// 返回：
	//   - error: 错误
	//
	OnComplete(ctx context.Context, result *AgentResult) error

	// Name 返回中间件名称。
	Name() string
}

// BaseAgentMiddleware 是基础 Agent 中间件。
//
// BaseAgentMiddleware 提供默认的空实现，子类只需覆盖需要的方法。
//
type BaseAgentMiddleware struct {
	name string
}

// NewBaseAgentMiddleware 创建基础 Agent 中间件。
func NewBaseAgentMiddleware(name string) *BaseAgentMiddleware {
	return &BaseAgentMiddleware{
		name: name,
	}
}

// BeforeModel 实现 AgentMiddleware 接口（默认：不做任何修改）。
func (b *BaseAgentMiddleware) BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error) {
	return state, nil
}

// AfterModel 实现 AgentMiddleware 接口（默认：不做任何修改）。
func (b *BaseAgentMiddleware) AfterModel(ctx context.Context, state *AgentState, response *types.Message) (*types.Message, error) {
	return response, nil
}

// OnError 实现 AgentMiddleware 接口（默认：不重试，原样返回错误）。
func (b *BaseAgentMiddleware) OnError(ctx context.Context, state *AgentState, err error) (bool, error) {
	return false, err
}

// BeforeToolCall 实现 AgentMiddleware 接口（默认：不做任何修改）。
func (b *BaseAgentMiddleware) BeforeToolCall(ctx context.Context, toolName string, toolInput map[string]any) (map[string]any, error) {
	return toolInput, nil
}

// AfterToolCall 实现 AgentMiddleware 接口（默认：不做任何修改）。
func (b *BaseAgentMiddleware) AfterToolCall(ctx context.Context, toolName string, toolInput map[string]any, toolOutput string, err error) (string, error) {
	return toolOutput, err
}

// OnComplete 实现 AgentMiddleware 接口（默认：什么也不做）。
func (b *BaseAgentMiddleware) OnComplete(ctx context.Context, result *AgentResult) error {
	return nil
}

// Name 实现 AgentMiddleware 接口。
func (b *BaseAgentMiddleware) Name() string {
	return b.name
}

// AgentMiddlewareChain 是中间件链。
//
// AgentMiddlewareChain 按顺序执行多个中间件。
//
type AgentMiddlewareChain struct {
	middlewares []AgentMiddleware
}

// NewAgentMiddlewareChain 创建中间件链。
func NewAgentMiddlewareChain(middlewares ...AgentMiddleware) *AgentMiddlewareChain {
	return &AgentMiddlewareChain{
		middlewares: middlewares,
	}
}

// Add 添加中间件到链中。
func (c *AgentMiddlewareChain) Add(middleware AgentMiddleware) *AgentMiddlewareChain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

// BeforeModel 依次执行所有中间件的 BeforeModel。
func (c *AgentMiddlewareChain) BeforeModel(ctx context.Context, state *AgentState) (*AgentState, error) {
	currentState := state
	for _, mw := range c.middlewares {
		newState, err := mw.BeforeModel(ctx, currentState)
		if err != nil {
			return nil, fmt.Errorf("middleware %s: %w", mw.Name(), err)
		}
		currentState = newState
	}
	return currentState, nil
}

// AfterModel 依次执行所有中间件的 AfterModel。
func (c *AgentMiddlewareChain) AfterModel(ctx context.Context, state *AgentState, response *types.Message) (*types.Message, error) {
	currentResponse := response
	for _, mw := range c.middlewares {
		newResponse, err := mw.AfterModel(ctx, state, currentResponse)
		if err != nil {
			return nil, fmt.Errorf("middleware %s: %w", mw.Name(), err)
		}
		currentResponse = newResponse
	}
	return currentResponse, nil
}

// OnError 依次执行所有中间件的 OnError。
func (c *AgentMiddlewareChain) OnError(ctx context.Context, state *AgentState, err error) (bool, error) {
	shouldRetry := false
	currentErr := err

	for _, mw := range c.middlewares {
		retry, newErr := mw.OnError(ctx, state, currentErr)
		if retry {
			shouldRetry = true
		}
		if newErr != nil {
			currentErr = newErr
		}
	}

	return shouldRetry, currentErr
}

// BeforeToolCall 依次执行所有中间件的 BeforeToolCall。
func (c *AgentMiddlewareChain) BeforeToolCall(ctx context.Context, toolName string, toolInput map[string]any) (map[string]any, error) {
	currentInput := toolInput
	for _, mw := range c.middlewares {
		newInput, err := mw.BeforeToolCall(ctx, toolName, currentInput)
		if err != nil {
			return nil, fmt.Errorf("middleware %s: %w", mw.Name(), err)
		}
		currentInput = newInput
	}
	return currentInput, nil
}

// AfterToolCall 依次执行所有中间件的 AfterToolCall。
func (c *AgentMiddlewareChain) AfterToolCall(ctx context.Context, toolName string, toolInput map[string]any, toolOutput string, err error) (string, error) {
	currentOutput := toolOutput
	currentErr := err

	for _, mw := range c.middlewares {
		newOutput, newErr := mw.AfterToolCall(ctx, toolName, toolInput, currentOutput, currentErr)
		currentOutput = newOutput
		if newErr != nil {
			currentErr = newErr
		}
	}

	return currentOutput, currentErr
}

// OnComplete 依次执行所有中间件的 OnComplete。
func (c *AgentMiddlewareChain) OnComplete(ctx context.Context, result *AgentResult) error {
	for _, mw := range c.middlewares {
		if err := mw.OnComplete(ctx, result); err != nil {
			return fmt.Errorf("middleware %s: %w", mw.Name(), err)
		}
	}
	return nil
}

// Name 返回链的名称。
func (c *AgentMiddlewareChain) Name() string {
	return "AgentMiddlewareChain"
}

// GetMiddlewares 返回所有中间件。
func (c *AgentMiddlewareChain) GetMiddlewares() []AgentMiddleware {
	return c.middlewares
}

// AgentExecutorConfig 扩展以支持 Middleware
// 注意：这个扩展会在 executor.go 中使用

// WithMiddleware 是 AgentOption，用于添加中间件。
func WithMiddleware(middleware AgentMiddleware) AgentOption {
	return func(config *AgentConfig) {
		if config.Extra == nil {
			config.Extra = make(map[string]any)
		}

		// 将中间件添加到配置中
		if existingMiddlewares, ok := config.Extra["middlewares"].([]AgentMiddleware); ok {
			config.Extra["middlewares"] = append(existingMiddlewares, middleware)
		} else {
			config.Extra["middlewares"] = []AgentMiddleware{middleware}
		}
	}
}

// WithMiddlewareChain 是 AgentOption，用于添加中间件链。
func WithMiddlewareChain(middlewares ...AgentMiddleware) AgentOption {
	return func(config *AgentConfig) {
		if config.Extra == nil {
			config.Extra = make(map[string]any)
		}

		chain := NewAgentMiddlewareChain(middlewares...)
		config.Extra["middleware_chain"] = chain
	}
}

// GetMiddlewareChainFromConfig 从配置中提取中间件链。
func GetMiddlewareChainFromConfig(config *AgentConfig) *AgentMiddlewareChain {
	if config.Extra == nil {
		return NewAgentMiddlewareChain()
	}

	// 检查是否有完整的链
	if chain, ok := config.Extra["middleware_chain"].(*AgentMiddlewareChain); ok {
		return chain
	}

	// 否则从独立的中间件构建链
	if middlewares, ok := config.Extra["middlewares"].([]AgentMiddleware); ok {
		return NewAgentMiddlewareChain(middlewares...)
	}

	return NewAgentMiddlewareChain()
}

package middleware

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Chain 是中间件链。
//
// Chain 管理多个中间件，并按顺序执行它们。
//
type Chain struct {
	middlewares []*MiddlewareWithMeta
	mu          sync.RWMutex
}

// NewChain 创建中间件链。
//
// 返回：
//   - *Chain: 中间件链实例
//
func NewChain() *Chain {
	return &Chain{
		middlewares: make([]*MiddlewareWithMeta, 0),
	}
}

// Use 添加中间件。
//
// 参数：
//   - middleware: 中间件
//
// 返回：
//   - *Chain: 返回自身，支持链式调用
//
func (c *Chain) Use(middleware Middleware) *Chain {
	c.mu.Lock()
	defer c.mu.Unlock()

	if middleware == nil {
		return c
	}

	// 如果已经是 MiddlewareWithMeta，直接使用
	if mwm, ok := middleware.(*MiddlewareWithMeta); ok {
		c.middlewares = append(c.middlewares, mwm)
	} else {
		// 创建默认元数据
		meta := NewMetadata(fmt.Sprintf("middleware-%d", len(c.middlewares)))
		c.middlewares = append(c.middlewares, NewMiddlewareWithMeta(middleware, meta))
	}

	return c
}

// UseWithMeta 添加带元数据的中间件。
//
// 参数：
//   - middleware: 中间件
//   - metadata: 元数据
//
// 返回：
//   - *Chain: 返回自身，支持链式调用
//
func (c *Chain) UseWithMeta(middleware Middleware, metadata *Metadata) *Chain {
	c.mu.Lock()
	defer c.mu.Unlock()

	if middleware == nil {
		return c
	}

	c.middlewares = append(c.middlewares, NewMiddlewareWithMeta(middleware, metadata))
	return c
}

// Remove 移除中间件（按名称）。
//
// 参数：
//   - name: 中间件名称
//
// 返回：
//   - bool: 是否成功移除
//
func (c *Chain) Remove(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, mw := range c.middlewares {
		if mw.metadata.Name == name {
			c.middlewares = append(c.middlewares[:i], c.middlewares[i+1:]...)
			return true
		}
	}

	return false
}

// Clear 清空中间件链。
func (c *Chain) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.middlewares = make([]*MiddlewareWithMeta, 0)
}

// Len 返回中间件数量。
func (c *Chain) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.middlewares)
}

// Execute 执行中间件链。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//   - handler: 最终处理函数
//
// 返回：
//   - any: 输出数据
//   - error: 错误
//
func (c *Chain) Execute(ctx context.Context, input any, handler HandlerFunc) (any, error) {
	if handler == nil {
		return nil, ErrHandlerNil
	}

	c.mu.RLock()
	middlewares := make([]*MiddlewareWithMeta, len(c.middlewares))
	copy(middlewares, c.middlewares)
	c.mu.RUnlock()

	// 构建执行链
	next := c.buildChain(middlewares, handler)

	// 执行
	return next(ctx, input)
}

// buildChain 构建中间件执行链。
func (c *Chain) buildChain(middlewares []*MiddlewareWithMeta, handler HandlerFunc) NextFunc {
	// 从后向前构建链
	next := NextFunc(handler)

	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		currentNext := next

		// 闭包捕获当前中间件和下一个处理函数
		next = func(ctx context.Context, input any) (any, error) {
			// 设置当前中间件名称到上下文
			ctx = WithMiddlewareName(ctx, mw.metadata.Name)

			// 执行中间件
			return mw.Process(ctx, input, currentNext)
		}
	}

	return next
}

// SortByPriority 按优先级排序中间件。
//
// 优先级数字越小，执行顺序越靠前。
//
func (c *Chain) SortByPriority() {
	c.mu.Lock()
	defer c.mu.Unlock()

	sort.Slice(c.middlewares, func(i, j int) bool {
		return c.middlewares[i].metadata.Priority < c.middlewares[j].metadata.Priority
	})
}

// GetMiddlewares 获取所有中间件（副本）。
func (c *Chain) GetMiddlewares() []Middleware {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]Middleware, len(c.middlewares))
	for i, mw := range c.middlewares {
		result[i] = mw.middleware
	}

	return result
}

// GetMiddlewaresWithMeta 获取所有带元数据的中间件（副本）。
func (c *Chain) GetMiddlewaresWithMeta() []*MiddlewareWithMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*MiddlewareWithMeta, len(c.middlewares))
	copy(result, c.middlewares)

	return result
}

// Clone 克隆中间件链。
func (c *Chain) Clone() *Chain {
	c.mu.RLock()
	defer c.mu.RUnlock()

	newChain := NewChain()
	newChain.middlewares = make([]*MiddlewareWithMeta, len(c.middlewares))
	copy(newChain.middlewares, c.middlewares)

	return newChain
}

// ExecuteWithRecovery 带恢复的执行（捕获 panic）。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//   - handler: 最终处理函数
//
// 返回：
//   - any: 输出数据
//   - error: 错误
//
func (c *Chain) ExecuteWithRecovery(ctx context.Context, input any, handler HandlerFunc) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrMiddlewarePanic, r)
		}
	}()

	return c.Execute(ctx, input, handler)
}

// Wrap 包装 HandlerFunc 为 Middleware。
//
// 这允许将普通的处理函数转换为中间件。
//
func Wrap(handler HandlerFunc) Middleware {
	return MiddlewareFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return handler(ctx, input)
	})
}

// Compose 组合多个中间件为一个。
//
// 参数：
//   - middlewares: 中间件列表
//
// 返回：
//   - Middleware: 组合后的中间件
//
func Compose(middlewares ...Middleware) Middleware {
	chain := NewChain()
	for _, mw := range middlewares {
		chain.Use(mw)
	}

	return MiddlewareFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
		return chain.Execute(ctx, input, HandlerFunc(next))
	})
}

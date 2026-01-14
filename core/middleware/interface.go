package middleware

import (
	"context"
	"fmt"
)

// NextFunc 是下一个处理函数的类型。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入数据
//
// 返回：
//   - any: 输出数据
//   - error: 错误
//
type NextFunc func(ctx context.Context, input any) (any, error)

// Middleware 是中间件接口。
//
// 中间件可以在请求处理前后执行自定义逻辑。
//
type Middleware interface {
	// Process 处理请求
	//
	// 参数：
	//   - ctx: 上下文
	//   - input: 输入数据
	//   - next: 下一个处理函数
	//
	// 返回：
	//   - any: 输出数据
	//   - error: 错误
	//
	Process(ctx context.Context, input any, next NextFunc) (any, error)
}

// MiddlewareFunc 是中间件函数类型。
//
// 允许使用函数作为中间件。
//
type MiddlewareFunc func(ctx context.Context, input any, next NextFunc) (any, error)

// Process 实现 Middleware 接口。
func (f MiddlewareFunc) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	return f(ctx, input, next)
}

// NewFunc 创建函数中间件。
//
// 参数：
//   - fn: 中间件函数
//
// 返回：
//   - Middleware: 中间件实例
//
func NewFunc(fn MiddlewareFunc) Middleware {
	return fn
}

// HandlerFunc 是最终处理函数类型。
//
// 这是中间件链的最后一个环节。
//
type HandlerFunc func(ctx context.Context, input any) (any, error)

// Metadata 是中间件元数据。
type Metadata struct {
	// Name 中间件名称
	Name string

	// Description 描述
	Description string

	// Priority 优先级（数字越小优先级越高）
	Priority int

	// Extra 额外信息
	Extra map[string]any
}

// NewMetadata 创建元数据。
func NewMetadata(name string) *Metadata {
	return &Metadata{
		Name:  name,
		Extra: make(map[string]any),
	}
}

// WithDescription 设置描述。
func (m *Metadata) WithDescription(desc string) *Metadata {
	m.Description = desc
	return m
}

// WithPriority 设置优先级。
func (m *Metadata) WithPriority(priority int) *Metadata {
	m.Priority = priority
	return m
}

// MiddlewareWithMeta 是带元数据的中间件。
type MiddlewareWithMeta struct {
	middleware Middleware
	metadata   *Metadata
}

// NewMiddlewareWithMeta 创建带元数据的中间件。
func NewMiddlewareWithMeta(middleware Middleware, metadata *Metadata) *MiddlewareWithMeta {
	return &MiddlewareWithMeta{
		middleware: middleware,
		metadata:   metadata,
	}
}

// Process 实现 Middleware 接口。
func (m *MiddlewareWithMeta) Process(ctx context.Context, input any, next NextFunc) (any, error) {
	return m.middleware.Process(ctx, input, next)
}

// GetMetadata 获取元数据。
func (m *MiddlewareWithMeta) GetMetadata() *Metadata {
	return m.metadata
}

// GetMiddleware 获取底层中间件。
func (m *MiddlewareWithMeta) GetMiddleware() Middleware {
	return m.middleware
}

// Context 相关的键类型
type contextKey string

const (
	// ContextKeyMiddlewareChain 中间件链上下文键
	ContextKeyMiddlewareChain contextKey = "middleware:chain"

	// ContextKeyMiddlewareName 当前中间件名称
	ContextKeyMiddlewareName contextKey = "middleware:name"
)

// GetMiddlewareNameFromContext 从上下文获取中间件名称。
func GetMiddlewareNameFromContext(ctx context.Context) string {
	if name, ok := ctx.Value(ContextKeyMiddlewareName).(string); ok {
		return name
	}
	return ""
}

// WithMiddlewareName 设置中间件名称到上下文。
func WithMiddlewareName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, ContextKeyMiddlewareName, name)
}

// 错误定义
var (
	ErrMiddlewareNil    = fmt.Errorf("middleware: middleware is nil")
	ErrChainEmpty       = fmt.Errorf("middleware: chain is empty")
	ErrHandlerNil       = fmt.Errorf("middleware: handler is nil")
	ErrMiddlewarePanic  = fmt.Errorf("middleware: panic in middleware")
)

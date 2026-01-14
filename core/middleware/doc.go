// Package middleware 提供中间件系统。
//
// 中间件允许在执行流程中注入自定义逻辑，实现横切关注点（cross-cutting concerns）。
//
// # 核心概念
//
// Middleware 是一个可以在请求处理前后执行自定义逻辑的组件：
//   - 请求预处理
//   - 响应后处理
//   - 错误处理
//   - 日志记录
//   - 性能监控
//   - 权限验证
//
// # 基本使用
//
// 创建中间件：
//
//	logMiddleware := middleware.NewFunc(func(ctx context.Context, input any, next NextFunc) (any, error) {
//	    log.Printf("Input: %v", input)
//	    result, err := next(ctx, input)
//	    log.Printf("Output: %v, Error: %v", result, err)
//	    return result, err
//	})
//
// 使用中间件链：
//
//	chain := middleware.NewChain().
//	    Use(loggingMiddleware).
//	    Use(authMiddleware).
//	    Use(rateLimitMiddleware)
//
//	result, err := chain.Execute(ctx, input, handler)
//
// # 执行顺序
//
// 中间件按照添加顺序执行（洋葱模型）：
//
//	Middleware1.Before
//	  Middleware2.Before
//	    Middleware3.Before
//	      Handler
//	    Middleware3.After
//	  Middleware2.After
//	Middleware1.After
//
package middleware

package runnable

import (
	"context"
	"fmt"
)

// Chain 表示一个可执行链
//
// Chain 是 LCEL (LangChain Expression Language) 的 Go 等效实现。
// 它提供了一种声明式的方式来组合 Runnable 组件。
//
// 特性：
//   - 链式调用
//   - 管道操作符
//   - 并行执行
//   - 条件路由
//   - 错误处理
//
// 使用示例:
//
//	// 简单链
//	chain := NewChain(prompt).
//	    Pipe(llm).
//	    Pipe(parser)
//	
//	result, _ := chain.Invoke(ctx, input)
//	
//	// 并行执行
//	chain := NewChain(input).
//	    Parallel(
//	        prompt1.Pipe(llm1),
//	        prompt2.Pipe(llm2),
//	    )
//
type Chain[I, O any] struct {
	runnable Runnable[I, O]
	name     string
	metadata map[string]interface{}
}

// NewChain 创建新的链
func NewChain[I, O any](r Runnable[I, O]) *Chain[I, O] {
	return &Chain[I, O]{
		runnable: r,
		metadata: make(map[string]interface{}),
	}
}

// Invoke 执行链
func (c *Chain[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	return c.runnable.Invoke(ctx, input, opts...)
}

// Batch 批量执行
func (c *Chain[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	return c.runnable.Batch(ctx, inputs, opts...)
}

// Stream 流式执行
func (c *Chain[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	return c.runnable.Stream(ctx, input, opts...)
}

// Pipe 管道操作：将当前链的输出连接到下一个 Runnable 的输入
//
// 这是 LCEL 的核心操作符，相当于 Python 中的 | 操作符
//
// 注意：由于 Go 不支持方法的类型参数，这个方法返回 any 类型
// 使用时需要类型断言，或使用包级别的 PipeChain 函数
func (c *Chain[I, O]) Pipe(next any) any {
	// 这是一个临时解决方案
	// 实际使用时，请使用包级别的 PipeChain 函数
	return next
}

// WithName 设置链的名称
func (c *Chain[I, O]) WithName(name string) *Chain[I, O] {
	c.name = name
	return c
}

// WithMetadata 添加元数据
func (c *Chain[I, O]) WithMetadata(key string, value interface{}) *Chain[I, O] {
	c.metadata[key] = value
	return c
}

// ==================== 辅助函数 ====================

// Parallel 并行执行多个 Runnable
//
// 使用示例:
//
//	chain := Parallel(
//	    prompt1.Pipe(llm1),
//	    prompt2.Pipe(llm2),
//	    prompt3.Pipe(llm3),
//	)
//
func Parallel[I, O any](runnables ...Runnable[I, O]) Runnable[I, []O] {
	return &parallel[I, O]{
		runnables: runnables,
	}
}

// ParallelMap 并行执行并返回 map
//
// 使用示例:
//
//	chain := ParallelMap(map[string]Runnable[Input, Output]{
//	    "result1": runnable1,
//	    "result2": runnable2,
//	})
//
func ParallelMap[I, O any](runnables map[string]Runnable[I, O]) Runnable[I, map[string]O] {
	return &parallelMap[I, O]{
		runnables: runnables,
	}
}

// Route 条件路由
//
// 根据条件函数选择不同的执行路径
//
// 使用示例:
//
//	chain := Route(
//	    func(input string) string {
//	        if containsQuestion(input) {
//	            return "qa"
//	        }
//	        return "chat"
//	    },
//	    map[string]Runnable[string, string]{
//	        "qa":   qaChain,
//	        "chat": chatChain,
//	    },
//	)
//
func Route[I, O any, K comparable](
	selector func(I) K,
	routes map[K]Runnable[I, O],
) Runnable[I, O] {
	return &router[I, O, K]{
		selector: selector,
		routes:   routes,
	}
}

// Fallback 失败回退
//
// 如果主 Runnable 失败，尝试备用 Runnable
//
// 使用示例:
//
//	chain := Fallback(
//	    primaryLLM,
//	    fallbackLLM,
//	    localLLM,
//	)
//
func Fallback[I, O any](runnables ...Runnable[I, O]) Runnable[I, O] {
	return &fallback[I, O]{
		runnables: runnables,
	}
}

// Retry 重试包装器
//
// 为 Runnable 添加重试逻辑
//
// 使用示例:
//
//	chain := Retry(llm, 3, time.Second)
//
func Retry[I, O any](r Runnable[I, O], maxRetries int, delay interface{}) Runnable[I, O] {
	return &retry[I, O]{
		runnable:   r,
		maxRetries: maxRetries,
		delay:      delay,
	}
}

// Map 转换函数
//
// 将输入转换为输出的简单函数包装
//
// 使用示例:
//
//	uppercase := Map(func(ctx context.Context, s string) (string, error) {
//	    return strings.ToUpper(s), nil
//	})
//	
//	chain := NewChain(prompt).Pipe(llm).Pipe(uppercase)
//
func Map[I, O any](fn func(context.Context, I) (O, error)) Runnable[I, O] {
	return &mapper[I, O]{
		fn: fn,
	}
}

// Filter 过滤器
//
// 根据条件过滤输入
//
// 使用示例:
//
//	notEmpty := Filter(func(s string) bool {
//	    return len(s) > 0
//	})
//
func Filter[I any](predicate func(I) bool) Runnable[I, I] {
	return &filter[I]{
		predicate: predicate,
	}
}

// ==================== 内部实现 ====================

// router 路由器实现
type router[I, O any, K comparable] struct {
	selector func(I) K
	routes   map[K]Runnable[I, O]
}

func (r *router[I, O, K]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	key := r.selector(input)
	
	runnable, ok := r.routes[key]
	if !ok {
		var zero O
		return zero, fmt.Errorf("no route found for key: %v", key)
	}
	
	return runnable.Invoke(ctx, input, opts...)
}

func (r *router[I, O, K]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	results := make([]O, len(inputs))
	
	for i, input := range inputs {
		result, err := r.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	
	return results, nil
}

func (r *router[I, O, K]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	key := r.selector(input)
	
	runnable, ok := r.routes[key]
	if !ok {
		ch := make(chan StreamEvent[O], 1)
		ch <- StreamEvent[O]{Error: fmt.Errorf("no route found for key: %v", key)}
		close(ch)
		return ch, nil
	}
	
	return runnable.Stream(ctx, input, opts...)
}

// fallback 回退实现
type fallback[I, O any] struct {
	runnables []Runnable[I, O]
}

func (f *fallback[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	var lastErr error
	
	for _, r := range f.runnables {
		result, err := r.Invoke(ctx, input, opts...)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	
	var zero O
	return zero, fmt.Errorf("all runnables failed, last error: %w", lastErr)
}

func (f *fallback[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	results := make([]O, len(inputs))
	
	for i, input := range inputs {
		result, err := f.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	
	return results, nil
}

func (f *fallback[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	// 尝试每个 runnable 直到成功
	for _, r := range f.runnables {
		ch, err := r.Stream(ctx, input, opts...)
		if err == nil {
			return ch, nil
		}
	}
	
	ch := make(chan StreamEvent[O], 1)
	ch <- StreamEvent[O]{Error: fmt.Errorf("all runnables failed")}
	close(ch)
	return ch, nil
}

// mapper 映射函数实现
type mapper[I, O any] struct {
	fn func(context.Context, I) (O, error)
}

func (m *mapper[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (O, error) {
	return m.fn(ctx, input)
}

func (m *mapper[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	results := make([]O, len(inputs))
	
	for i, input := range inputs {
		result, err := m.fn(ctx, input)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	
	return results, nil
}

func (m *mapper[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[O], error) {
	ch := make(chan StreamEvent[O], 1)
	
	go func() {
		defer close(ch)
		
		result, err := m.fn(ctx, input)
		if err != nil {
			ch <- StreamEvent[O]{Error: err}
			return
		}
		
		ch <- StreamEvent[O]{Data: result}
	}()
	
	return ch, nil
}

// filter 过滤器实现
type filter[I any] struct {
	predicate func(I) bool
}

func (f *filter[I]) Invoke(ctx context.Context, input I, opts ...Option) (I, error) {
	if !f.predicate(input) {
		var zero I
		return zero, fmt.Errorf("input filtered out")
	}
	return input, nil
}

func (f *filter[I]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]I, error) {
	var results []I
	
	for _, input := range inputs {
		if f.predicate(input) {
			results = append(results, input)
		}
	}
	
	return results, nil
}

func (f *filter[I]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[I], error) {
	ch := make(chan StreamEvent[I], 1)
	
	go func() {
		defer close(ch)
		
		if f.predicate(input) {
			ch <- StreamEvent[I]{Data: input}
		} else {
			ch <- StreamEvent[I]{Error: fmt.Errorf("input filtered out")}
		}
	}()
	
	return ch, nil
}

// parallelMap 并行 map 实现
type parallelMap[I, O any] struct {
	runnables map[string]Runnable[I, O]
}

func (p *parallelMap[I, O]) Invoke(ctx context.Context, input I, opts ...Option) (map[string]O, error) {
	type result struct {
		key   string
		value O
		err   error
	}
	
	results := make(chan result, len(p.runnables))
	
	// 并行执行所有 runnable
	for key, r := range p.runnables {
		go func(k string, runnable Runnable[I, O]) {
			value, err := runnable.Invoke(ctx, input, opts...)
			results <- result{key: k, value: value, err: err}
		}(key, r)
	}
	
	// 收集结果
	output := make(map[string]O)
	for i := 0; i < len(p.runnables); i++ {
		res := <-results
		if res.err != nil {
			return nil, fmt.Errorf("runnable %s failed: %w", res.key, res.err)
		}
		output[res.key] = res.value
	}
	
	return output, nil
}

func (p *parallelMap[I, O]) Batch(ctx context.Context, inputs []I, opts ...Option) ([]map[string]O, error) {
	results := make([]map[string]O, len(inputs))
	
	for i, input := range inputs {
		result, err := p.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	
	return results, nil
}

func (p *parallelMap[I, O]) Stream(ctx context.Context, input I, opts ...Option) (<-chan StreamEvent[map[string]O], error) {
	ch := make(chan StreamEvent[map[string]O], 1)
	
	go func() {
		defer close(ch)
		
		result, err := p.Invoke(ctx, input, opts...)
		if err != nil {
			ch <- StreamEvent[map[string]O]{Error: err}
			return
		}
		
		ch <- StreamEvent[map[string]O]{Data: result}
	}()
	
	return ch, nil
}

package edge

import (
	"fmt"
	"sync"
)

// ConditionFunc 是条件函数的类型。
//
// ConditionFunc 判断给定状态是否满足某个条件。
//
// 类型参数：
//   - S: 状态类型
//
type ConditionFunc[S any] func(state S) bool

// Route 是路由规则。
//
// Route 包含路径名称、目标节点和条件函数。
//
type Route[S any] struct {
	Name      string
	Target    string
	Condition ConditionFunc[S]
	Priority  int // 优先级（数字越大优先级越高）
}

// Router 是路由器。
//
// Router 提供更灵活的路由逻辑，支持：
//   - 多个路由规则
//   - 优先级
//   - 默认路由
//   - 动态路由
//
// 示例：
//
//	router := NewRouter[MyState]()
//
//	// 添加路由规则
//	router.AddRoute("positive", "positive_node", func(s MyState) bool {
//	    return s.Counter > 0
//	})
//
//	router.AddRoute("negative", "negative_node", func(s MyState) bool {
//	    return s.Counter < 0
//	})
//
//	// 设置默认路由
//	router.SetDefault("zero_node")
//
//	// 路由
//	target, _ := router.Route(state)
//
type Router[S any] struct {
	routes        []*Route[S]
	defaultTarget string
	mu            sync.RWMutex
}

// NewRouter 创建路由器。
//
// 返回：
//   - *Router[S]: 路由器实例
//
func NewRouter[S any]() *Router[S] {
	return &Router[S]{
		routes: make([]*Route[S], 0),
	}
}

// AddRoute 添加路由规则。
//
// 参数：
//   - name: 路由名称
//   - target: 目标节点
//   - condition: 条件函数
//
// 返回：
//   - *Router[S]: 返回自身，支持链式调用
//
func (r *Router[S]) AddRoute(name, target string, condition ConditionFunc[S]) *Router[S] {
	return r.AddRouteWithPriority(name, target, condition, 0)
}

// AddRouteWithPriority 添加带优先级的路由规则。
//
// 参数：
//   - name: 路由名称
//   - target: 目标节点
//   - condition: 条件函数
//   - priority: 优先级（数字越大优先级越高）
//
// 返回：
//   - *Router[S]: 返回自身，支持链式调用
//
func (r *Router[S]) AddRouteWithPriority(
	name, target string,
	condition ConditionFunc[S],
	priority int,
) *Router[S] {
	r.mu.Lock()
	defer r.mu.Unlock()

	route := &Route[S]{
		Name:      name,
		Target:    target,
		Condition: condition,
		Priority:  priority,
	}

	// 按优先级插入（优先级高的在前）
	inserted := false
	for i, existing := range r.routes {
		if route.Priority > existing.Priority {
			r.routes = append(r.routes[:i], append([]*Route[S]{route}, r.routes[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		r.routes = append(r.routes, route)
	}

	return r
}

// RemoveRoute 移除路由规则。
//
// 参数：
//   - name: 路由名称
//
// 返回：
//   - *Router[S]: 返回自身，支持链式调用
//
func (r *Router[S]) RemoveRoute(name string) *Router[S] {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, route := range r.routes {
		if route.Name == name {
			r.routes = append(r.routes[:i], r.routes[i+1:]...)
			break
		}
	}

	return r
}

// SetDefault 设置默认路由。
//
// 如果所有路由规则都不匹配，会使用默认路由。
//
// 参数：
//   - target: 默认目标节点
//
// 返回：
//   - *Router[S]: 返回自身，支持链式调用
//
func (r *Router[S]) SetDefault(target string) *Router[S] {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultTarget = target
	return r
}

// Route 执行路由。
//
// 按优先级顺序检查路由规则，返回第一个匹配的目标节点。
// 如果没有规则匹配，返回默认路由。
//
// 参数：
//   - state: 当前状态
//
// 返回：
//   - string: 目标节点名称
//   - error: 路由错误
//
func (r *Router[S]) Route(state S) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 按优先级检查路由规则
	for _, route := range r.routes {
		if route.Condition != nil && route.Condition(state) {
			return route.Target, nil
		}
	}

	// 使用默认路由
	if r.defaultTarget != "" {
		return r.defaultTarget, nil
	}

	return "", ErrNoRouteMatched
}

// GetRoutes 返回所有路由规则（副本）。
func (r *Router[S]) GetRoutes() []*Route[S] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Route[S], len(r.routes))
	copy(result, r.routes)
	return result
}

// GetDefault 返回默认路由。
func (r *Router[S]) GetDefault() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaultTarget
}

// Clear 清空所有路由规则。
func (r *Router[S]) Clear() *Router[S] {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = make([]*Route[S], 0)
	r.defaultTarget = ""
	return r
}

// Validate 验证路由器配置。
func (r *Router[S]) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.routes) == 0 && r.defaultTarget == "" {
		return fmt.Errorf("router: no routes and no default target")
	}

	for _, route := range r.routes {
		if route.Name == "" {
			return fmt.Errorf("router: route name cannot be empty")
		}
		if route.Target == "" {
			return fmt.Errorf("router: target cannot be empty for route '%s'", route.Name)
		}
		if route.Condition == nil {
			return fmt.Errorf("router: condition cannot be nil for route '%s'", route.Name)
		}
	}

	return nil
}

// MatchedRoute 返回匹配的路由信息。
//
// 与 Route 类似，但返回完整的路由信息而不仅是目标节点。
//
// 参数：
//   - state: 当前状态
//
// 返回：
//   - *Route[S]: 匹配的路由
//   - bool: 是否匹配成功
//
func (r *Router[S]) MatchedRoute(state S) (*Route[S], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.Condition != nil && route.Condition(state) {
			return route, true
		}
	}

	return nil, false
}

// RouterBuilder 是路由器构建器。
//
// RouterBuilder 提供链式 API 来构建路由器。
//
type RouterBuilder[S any] struct {
	router *Router[S]
}

// NewRouterBuilder 创建路由器构建器。
//
// 返回：
//   - *RouterBuilder[S]: 构建器实例
//
func NewRouterBuilder[S any]() *RouterBuilder[S] {
	return &RouterBuilder[S]{
		router: NewRouter[S](),
	}
}

// When 添加条件路由。
//
// 参数：
//   - condition: 条件函数
//
// 返回：
//   - *RouteBuilder[S]: 路由构建器
//
func (rb *RouterBuilder[S]) When(condition ConditionFunc[S]) *RouteBuilder[S] {
	return &RouteBuilder[S]{
		builder:   rb,
		condition: condition,
	}
}

// Default 设置默认路由。
//
// 参数：
//   - target: 默认目标节点
//
// 返回：
//   - *RouterBuilder[S]: 返回自身
//
func (rb *RouterBuilder[S]) Default(target string) *RouterBuilder[S] {
	rb.router.SetDefault(target)
	return rb
}

// Build 构建路由器。
//
// 返回：
//   - *Router[S]: 路由器实例
//
func (rb *RouterBuilder[S]) Build() *Router[S] {
	return rb.router
}

// RouteBuilder 是路由构建器。
type RouteBuilder[S any] struct {
	builder   *RouterBuilder[S]
	condition ConditionFunc[S]
}

// Then 指定目标节点。
//
// 参数：
//   - name: 路由名称
//   - target: 目标节点
//
// 返回：
//   - *RouterBuilder[S]: 返回父构建器
//
func (rb *RouteBuilder[S]) Then(name, target string) *RouterBuilder[S] {
	rb.builder.router.AddRoute(name, target, rb.condition)
	return rb.builder
}

// ThenWithPriority 指定带优先级的目标节点。
//
// 参数：
//   - name: 路由名称
//   - target: 目标节点
//   - priority: 优先级
//
// 返回：
//   - *RouterBuilder[S]: 返回父构建器
//
func (rb *RouteBuilder[S]) ThenWithPriority(name, target string, priority int) *RouterBuilder[S] {
	rb.builder.router.AddRouteWithPriority(name, target, rb.condition, priority)
	return rb.builder
}

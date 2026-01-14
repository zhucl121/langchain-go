package edge

import (
	"errors"
	"testing"
)

// TestNewRouter 测试创建路由器
func TestNewRouter(t *testing.T) {
	router := NewRouter[TestState]()

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if len(router.routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(router.routes))
	}
}

// TestRouter_AddRoute 测试添加路由
func TestRouter_AddRoute(t *testing.T) {
	router := NewRouter[TestState]()

	router.AddRoute("positive", "positive_node", func(s TestState) bool {
		return s.Counter > 0
	})

	routes := router.GetRoutes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(routes))
	}

	if routes[0].Name != "positive" {
		t.Errorf("expected route name 'positive', got %s", routes[0].Name)
	}
}

// TestRouter_AddRouteWithPriority 测试添加带优先级的路由
func TestRouter_AddRouteWithPriority(t *testing.T) {
	router := NewRouter[TestState]()

	// 添加低优先级路由
	router.AddRouteWithPriority("low", "low_node", func(s TestState) bool {
		return true
	}, 1)

	// 添加高优先级路由
	router.AddRouteWithPriority("high", "high_node", func(s TestState) bool {
		return true
	}, 10)

	// 添加中优先级路由
	router.AddRouteWithPriority("mid", "mid_node", func(s TestState) bool {
		return true
	}, 5)

	routes := router.GetRoutes()

	// 验证顺序：high (10) -> mid (5) -> low (1)
	if routes[0].Name != "high" {
		t.Errorf("expected first route 'high', got %s", routes[0].Name)
	}
	if routes[1].Name != "mid" {
		t.Errorf("expected second route 'mid', got %s", routes[1].Name)
	}
	if routes[2].Name != "low" {
		t.Errorf("expected third route 'low', got %s", routes[2].Name)
	}
}

// TestRouter_RemoveRoute 测试移除路由
func TestRouter_RemoveRoute(t *testing.T) {
	router := NewRouter[TestState]()

	router.AddRoute("route1", "node1", func(s TestState) bool { return true })
	router.AddRoute("route2", "node2", func(s TestState) bool { return true })

	router.RemoveRoute("route1")

	routes := router.GetRoutes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route after removal, got %d", len(routes))
	}

	if routes[0].Name != "route2" {
		t.Errorf("expected remaining route 'route2', got %s", routes[0].Name)
	}
}

// TestRouter_SetDefault 测试设置默认路由
func TestRouter_SetDefault(t *testing.T) {
	router := NewRouter[TestState]()

	router.SetDefault("default_node")

	if router.GetDefault() != "default_node" {
		t.Errorf("expected default 'default_node', got %s", router.GetDefault())
	}
}

// TestRouter_Route 测试路由
func TestRouter_Route(t *testing.T) {
	router := NewRouter[TestState]()

	router.AddRoute("positive", "positive_node", func(s TestState) bool {
		return s.Counter > 0
	})

	router.AddRoute("negative", "negative_node", func(s TestState) bool {
		return s.Counter < 0
	})

	router.SetDefault("zero_node")

	// 测试正数
	target1, err := router.Route(TestState{Counter: 10})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if target1 != "positive_node" {
		t.Errorf("expected 'positive_node', got %s", target1)
	}

	// 测试负数
	target2, err := router.Route(TestState{Counter: -5})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if target2 != "negative_node" {
		t.Errorf("expected 'negative_node', got %s", target2)
	}

	// 测试零（使用默认）
	target3, err := router.Route(TestState{Counter: 0})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if target3 != "zero_node" {
		t.Errorf("expected 'zero_node', got %s", target3)
	}
}

// TestRouter_Route_NoMatch 测试无匹配路由
func TestRouter_Route_NoMatch(t *testing.T) {
	router := NewRouter[TestState]()

	router.AddRoute("never_match", "node", func(s TestState) bool {
		return false
	})

	_, err := router.Route(TestState{})
	if !errors.Is(err, ErrNoRouteMatched) {
		t.Errorf("expected ErrNoRouteMatched, got %v", err)
	}
}

// TestRouter_Route_Priority 测试优先级路由
func TestRouter_Route_Priority(t *testing.T) {
	router := NewRouter[TestState]()

	// 两个都会匹配的路由，高优先级应该先匹配
	router.AddRouteWithPriority("low", "low_node", func(s TestState) bool {
		return true
	}, 1)

	router.AddRouteWithPriority("high", "high_node", func(s TestState) bool {
		return true
	}, 10)

	target, err := router.Route(TestState{})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// 应该匹配高优先级
	if target != "high_node" {
		t.Errorf("expected 'high_node', got %s", target)
	}
}

// TestRouter_Clear 测试清空路由
func TestRouter_Clear(t *testing.T) {
	router := NewRouter[TestState]()

	router.AddRoute("route1", "node1", func(s TestState) bool { return true })
	router.SetDefault("default")

	router.Clear()

	if len(router.GetRoutes()) != 0 {
		t.Error("expected routes to be cleared")
	}

	if router.GetDefault() != "" {
		t.Error("expected default to be cleared")
	}
}

// TestRouter_Validate 测试验证路由器
func TestRouter_Validate(t *testing.T) {
	// 有效路由器
	validRouter := NewRouter[TestState]()
	validRouter.AddRoute("route1", "node1", func(s TestState) bool { return true })

	if err := validRouter.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 有效路由器（仅默认）
	validRouter2 := NewRouter[TestState]()
	validRouter2.SetDefault("default")

	if err := validRouter2.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 无效路由器（无路由也无默认）
	invalidRouter1 := NewRouter[TestState]()

	if err := invalidRouter1.Validate(); err == nil {
		t.Error("expected error for empty router")
	}

	// 无效路由器（路由缺少条件）
	invalidRouter2 := NewRouter[TestState]()
	invalidRouter2.routes = []*Route[TestState]{
		{Name: "test", Target: "node", Condition: nil},
	}

	if err := invalidRouter2.Validate(); err == nil {
		t.Error("expected error for route without condition")
	}
}

// TestRouter_MatchedRoute 测试获取匹配的路由
func TestRouter_MatchedRoute(t *testing.T) {
	router := NewRouter[TestState]()

	router.AddRoute("positive", "positive_node", func(s TestState) bool {
		return s.Counter > 0
	})

	// 匹配成功
	route, matched := router.MatchedRoute(TestState{Counter: 10})
	if !matched {
		t.Fatal("expected route to match")
	}

	if route.Name != "positive" {
		t.Errorf("expected route 'positive', got %s", route.Name)
	}

	// 匹配失败
	_, matched2 := router.MatchedRoute(TestState{Counter: -5})
	if matched2 {
		t.Error("expected no match")
	}
}

// TestRouter_ChainCall 测试链式调用
func TestRouter_ChainCall(t *testing.T) {
	router := NewRouter[TestState]()

	result := router.
		AddRoute("route1", "node1", func(s TestState) bool { return true }).
		SetDefault("default")

	if result != router {
		t.Error("chain call should return self")
	}

	if len(router.GetRoutes()) != 1 {
		t.Error("route should be added")
	}

	if router.GetDefault() != "default" {
		t.Error("default should be set")
	}
}

// TestRouterBuilder 测试路由器构建器
func TestRouterBuilder(t *testing.T) {
	router := NewRouterBuilder[TestState]().
		When(func(s TestState) bool { return s.Counter > 0 }).
		Then("positive", "positive_node").
		When(func(s TestState) bool { return s.Counter < 0 }).
		Then("negative", "negative_node").
		Default("zero_node").
		Build()

	// 测试正数
	target1, _ := router.Route(TestState{Counter: 10})
	if target1 != "positive_node" {
		t.Errorf("expected 'positive_node', got %s", target1)
	}

	// 测试负数
	target2, _ := router.Route(TestState{Counter: -5})
	if target2 != "negative_node" {
		t.Errorf("expected 'negative_node', got %s", target2)
	}

	// 测试零（默认）
	target3, _ := router.Route(TestState{Counter: 0})
	if target3 != "zero_node" {
		t.Errorf("expected 'zero_node', got %s", target3)
	}
}

// TestRouterBuilder_WithPriority 测试带优先级的构建器
func TestRouterBuilder_WithPriority(t *testing.T) {
	router := NewRouterBuilder[TestState]().
		When(func(s TestState) bool { return true }).
		ThenWithPriority("high", "high_node", 10).
		When(func(s TestState) bool { return true }).
		ThenWithPriority("low", "low_node", 1).
		Build()

	// 应该匹配高优先级
	target, _ := router.Route(TestState{})
	if target != "high_node" {
		t.Errorf("expected 'high_node', got %s", target)
	}
}

// TestRouter_Concurrency 测试并发安全
func TestRouter_Concurrency(t *testing.T) {
	router := NewRouter[TestState]()

	router.AddRoute("route1", "node1", func(s TestState) bool {
		return s.Counter > 0
	})

	// 并发读取
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = router.Route(TestState{Counter: 5})
			done <- true
		}()
	}

	// 等待完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestBranchEdge_Basic 测试分支边基础功能
func TestBranchEdge_Basic(t *testing.T) {
	branches := map[string]string{
		"branch1": "node1",
		"branch2": "node2",
	}

	edge := NewBranchEdge[TestState]("source", branches, nil)

	if edge.GetSource() != "source" {
		t.Errorf("expected source 'source', got %s", edge.GetSource())
	}

	if edge.GetType() != TypeBranch {
		t.Errorf("expected type %s, got %s", TypeBranch, edge.GetType())
	}
}

// TestBranchEdge_Select 测试分支选择
func TestBranchEdge_Select(t *testing.T) {
	branches := map[string]string{
		"branch1": "node1",
		"branch2": "node2",
		"branch3": "node3",
	}

	selector := func(s TestState) []string {
		if s.Done {
			return []string{"branch1"}
		}
		return []string{"branch2", "branch3"}
	}

	edge := NewBranchEdge[TestState]("source", branches, selector)

	// 测试选择单个分支
	targets1, err := edge.Select(TestState{Done: true})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(targets1) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets1))
	}

	if targets1[0] != "node1" {
		t.Errorf("expected 'node1', got %s", targets1[0])
	}

	// 测试选择多个分支
	targets2, err := edge.Select(TestState{Done: false})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(targets2) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets2))
	}
}

// TestBranchEdge_Select_NoSelector 测试无选择器的分支
func TestBranchEdge_Select_NoSelector(t *testing.T) {
	branches := map[string]string{
		"branch1": "node1",
		"branch2": "node2",
	}

	edge := NewBranchEdge[TestState]("source", branches, nil)

	// 无选择器时应返回所有分支
	targets, err := edge.Select(TestState{})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
}

// TestBranchEdge_Validate 测试验证分支边
func TestBranchEdge_Validate(t *testing.T) {
	// 有效边
	validEdge := NewBranchEdge[TestState]("source",
		map[string]string{"branch1": "node1"},
		nil,
	)
	if err := validEdge.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 无效边（空源节点）
	invalidEdge1 := NewBranchEdge[TestState]("",
		map[string]string{"branch1": "node1"},
		nil,
	)
	if !errors.Is(invalidEdge1.Validate(), ErrEmptySourceNode) {
		t.Error("expected ErrEmptySourceNode")
	}

	// 无效边（无分支）
	invalidEdge2 := NewBranchEdge[TestState]("source",
		map[string]string{},
		nil,
	)
	if err := invalidEdge2.Validate(); err == nil {
		t.Error("expected error for empty branches")
	}
}

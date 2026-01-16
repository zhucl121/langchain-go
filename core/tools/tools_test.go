package tools_test

import (
	"context"
	"testing"
	"time"
	
	"langchain-go/core/tools"
)

// TestGetTimeTool 测试获取时间工具。
func TestGetTimeTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewGetTimeTool(nil)
	
	result, err := tool.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	timeStr, ok := result.(string)
	if !ok {
		t.Fatal("result should be a string")
	}
	
	if len(timeStr) == 0 {
		t.Error("time string should not be empty")
	}
	
	t.Logf("Current time: %s", timeStr)
}

// TestGetDateTool 测试获取日期工具。
func TestGetDateTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewGetDateTool(nil)
	
	result, err := tool.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	dateStr, ok := result.(string)
	if !ok {
		t.Fatal("result should be a string")
	}
	
	if len(dateStr) != 10 { // YYYY-MM-DD format
		t.Errorf("expected date format YYYY-MM-DD, got %s", dateStr)
	}
	
	t.Logf("Current date: %s", dateStr)
}

// TestFormatTimeTool 测试格式化时间工具。
func TestFormatTimeTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewFormatTimeTool()
	
	args := map[string]any{
		"time":          "2026-01-16 15:04:05",
		"input_format":  "2006-01-02 15:04:05",
		"output_format": "January 02, 2006",
	}
	
	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	formatted, ok := result.(string)
	if !ok {
		t.Fatal("result should be a string")
	}
	
	if formatted != "January 16, 2026" {
		t.Errorf("expected 'January 16, 2026', got '%s'", formatted)
	}
}

// TestGetDayOfWeekTool 测试获取星期几工具。
func TestGetDayOfWeekTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewGetDayOfWeekTool()
	
	args := map[string]any{
		"date": "2026-01-16",
	}
	
	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	dayOfWeek, ok := result.(string)
	if !ok {
		t.Fatal("result should be a string")
	}
	
	if dayOfWeek != "Friday" {
		t.Errorf("expected 'Friday', got '%s'", dayOfWeek)
	}
}

// TestRandomNumberTool 测试随机数工具。
func TestRandomNumberTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewRandomNumberTool()
	
	args := map[string]any{
		"min": float64(1),
		"max": float64(10),
	}
	
	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	num, ok := result.(int)
	if !ok {
		t.Fatal("result should be an int")
	}
	
	if num < 1 || num > 10 {
		t.Errorf("expected number between 1 and 10, got %d", num)
	}
}

// TestGetBuiltinTools 测试获取内置工具。
func TestGetBuiltinTools(t *testing.T) {
	allTools := tools.GetBuiltinTools()
	
	if len(allTools) == 0 {
		t.Fatal("builtin tools should not be empty")
	}
	
	t.Logf("Total builtin tools: %d", len(allTools))
	
	// 验证每个工具都有名称和描述
	for _, tool := range allTools {
		if tool.GetName() == "" {
			t.Error("tool name should not be empty")
		}
		if tool.GetDescription() == "" {
			t.Error("tool description should not be empty")
		}
	}
}

// TestGetBasicTools 测试获取基础工具。
func TestGetBasicTools(t *testing.T) {
	basicTools := tools.GetBasicTools()
	
	if len(basicTools) == 0 {
		t.Fatal("basic tools should not be empty")
	}
	
	t.Logf("Total basic tools: %d", len(basicTools))
}

// TestToolRegistry 测试工具注册表。
func TestToolRegistry(t *testing.T) {
	registry := tools.NewToolRegistry()
	
	// 注册工具
	calculator := tools.NewCalculatorTool()
	registry.Register(calculator)
	
	// 检查是否存在
	if !registry.Has("calculator") {
		t.Error("calculator should be registered")
	}
	
	// 获取工具
	tool, exists := registry.Get("calculator")
	if !exists {
		t.Fatal("calculator should exist")
	}
	
	if tool.GetName() != "calculator" {
		t.Errorf("expected name 'calculator', got '%s'", tool.GetName())
	}
	
	// 计数
	if registry.Count() != 1 {
		t.Errorf("expected count 1, got %d", registry.Count())
	}
	
	// 移除工具
	registry.Remove("calculator")
	if registry.Has("calculator") {
		t.Error("calculator should be removed")
	}
}

// TestDefaultRegistry 测试默认注册表。
func TestDefaultRegistry(t *testing.T) {
	if tools.DefaultRegistry == nil {
		t.Fatal("default registry should not be nil")
	}
	
	count := tools.DefaultRegistry.Count()
	if count == 0 {
		t.Error("default registry should have tools")
	}
	
	t.Logf("Default registry has %d tools", count)
}

// BenchmarkGetTimeTool Benchmark 获取时间工具。
func BenchmarkGetTimeTool(b *testing.B) {
	ctx := context.Background()
	tool := tools.NewGetTimeTool(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, nil)
	}
}

// BenchmarkCalculator Benchmark 计算器工具。
func BenchmarkCalculator(b *testing.B) {
	ctx := context.Background()
	tool := tools.NewCalculatorTool()
	
	args := map[string]any{
		"expression": "123 + 456",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args)
	}
}

// ExampleNewGetTimeTool 示例：使用时间工具。
func ExampleNewGetTimeTool() {
	ctx := context.Background()
	
	// 创建时间工具
	timeTool := tools.NewGetTimeTool(nil)
	
	// 执行工具
	result, _ := timeTool.Execute(ctx, nil)
	println("Current time:", result)
}

// ExampleNewHTTPGetTool 示例：使用 HTTP GET 工具。
func ExampleNewHTTPGetTool() {
	ctx := context.Background()
	
	// 创建 HTTP GET 工具
	httpTool := tools.NewHTTPGetTool(&tools.HTTPGetToolConfig{
		Timeout: 10 * time.Second,
	})
	
	// 执行 HTTP 请求
	result, _ := httpTool.Execute(ctx, map[string]any{
		"url": "https://api.github.com",
	})
	
	println("Response:", result)
}

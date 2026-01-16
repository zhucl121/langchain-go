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

// TestJSONParseTool 测试 JSON 解析工具。
func TestJSONParseTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewJSONParseTool()
	
	args := map[string]any{
		"json_string": `{"name": "John", "age": 30}`,
	}
	
	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	obj, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result should be a map")
	}
	
	if obj["name"] != "John" {
		t.Errorf("expected name 'John', got '%v'", obj["name"])
	}
	
	if obj["age"].(float64) != 30 {
		t.Errorf("expected age 30, got %v", obj["age"])
	}
}

// TestJSONStringifyTool 测试 JSON 序列化工具。
func TestJSONStringifyTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewJSONStringifyTool()
	
	args := map[string]any{
		"object": map[string]any{
			"name": "John",
			"age":  30,
		},
		"pretty": false,
	}
	
	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	jsonStr, ok := result.(string)
	if !ok {
		t.Fatal("result should be a string")
	}
	
	if len(jsonStr) == 0 {
		t.Error("JSON string should not be empty")
	}
	
	t.Logf("JSON: %s", jsonStr)
}

// TestJSONExtractTool 测试 JSON 提取工具。
func TestJSONExtractTool(t *testing.T) {
	ctx := context.Background()
	tool := tools.NewJSONExtractTool()
	
	args := map[string]any{
		"json_string": `{"user": {"name": "John", "age": 30}}`,
		"path":        "user.name",
	}
	
	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	
	name, ok := result.(string)
	if !ok {
		t.Fatal("result should be a string")
	}
	
	if name != "John" {
		t.Errorf("expected 'John', got '%s'", name)
	}
}

// TestStringTools 测试字符串工具。
func TestStringTools(t *testing.T) {
	ctx := context.Background()
	
	// 测试字符串长度
	lengthTool := tools.NewStringLengthTool()
	result, err := lengthTool.Execute(ctx, map[string]any{"text": "hello"})
	if err != nil || result.(int) != 5 {
		t.Errorf("string length test failed")
	}
	
	// 测试字符串分割
	splitTool := tools.NewStringSplitTool()
	result, err = splitTool.Execute(ctx, map[string]any{
		"text":      "a,b,c",
		"delimiter": ",",
	})
	if err != nil {
		t.Fatalf("string split failed: %v", err)
	}
	parts := result.([]string)
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
	
	// 测试字符串连接
	joinTool := tools.NewStringJoinTool()
	result, err = joinTool.Execute(ctx, map[string]any{
		"strings":   []any{"a", "b", "c"},
		"delimiter": "-",
	})
	if err != nil {
		t.Fatalf("string join failed: %v", err)
	}
	joined := result.(string)
	if joined != "a-b-c" {
		t.Errorf("expected 'a-b-c', got '%s'", joined)
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

// TestToolsByCategory 测试按分类获取工具。
func TestToolsByCategory(t *testing.T) {
	tests := []struct {
		category tools.ToolCategory
		minCount int
	}{
		{tools.CategoryTime, 3},
		{tools.CategoryHTTP, 2},
		{tools.CategoryJSON, 3},
		{tools.CategoryString, 2},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			categoryTools := tools.GetToolsByCategory(tt.category)
			if len(categoryTools) < tt.minCount {
				t.Errorf("expected at least %d tools, got %d", tt.minCount, len(categoryTools))
			}
		})
	}
}

// TestToolRegistry 测试工具注册表。
func TestToolRegistry(t *testing.T) {
	registry := tools.NewToolRegistry()
	
	// 注册工具
	calculator := tools.NewCalculator()
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
	tool := tools.NewCalculator()
	
	args := map[string]any{
		"expression": "123 + 456",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args)
	}
}

// ExampleGetBuiltinTools 示例：获取所有内置工具。
func ExampleGetBuiltinTools() {
	// 获取所有内置工具
	allTools := tools.GetBuiltinTools()
	
	println("Total tools:", len(allTools))
	
	// 列出所有工具
	for _, tool := range allTools {
		println("-", tool.GetName(), ":", tool.GetDescription())
	}
}

// ExampleGetBasicTools 示例：获取基础工具。
func ExampleGetBasicTools() {
	// 获取基础工具（最常用）
	basicTools := tools.GetBasicTools()
	
	// 使用基础工具创建 Agent
	// agent := agents.CreateReActAgent(llm, basicTools)
	
	println("Basic tools:", len(basicTools))
}

// ExampleToolRegistry 示例：使用工具注册表。
func ExampleToolRegistry() {
	// 创建自定义注册表
	registry := tools.NewToolRegistry()
	
	// 注册工具
	registry.Register(tools.NewCalculator())
	registry.Register(tools.NewGetTimeTool(nil))
	registry.Register(tools.NewGetDateTool(nil))
	
	// 获取所有工具
	allTools := registry.GetAll()
	println("Total registered tools:", len(allTools))
	
	// 检查工具是否存在
	if registry.Has("calculator") {
		println("Calculator is available")
	}
	
	// 获取特定工具
	if tool, exists := registry.Get("calculator"); exists {
		println("Found tool:", tool.GetName())
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

// ExampleNewJSONParseTool 示例：使用 JSON 解析工具。
func ExampleNewJSONParseTool() {
	ctx := context.Background()
	
	// 创建 JSON 解析工具
	jsonTool := tools.NewJSONParseTool()
	
	// 解析 JSON
	result, _ := jsonTool.Execute(ctx, map[string]any{
		"json_string": `{"name": "John", "age": 30}`,
	})
	
	// 访问解析后的对象
	obj := result.(map[string]any)
	println("Name:", obj["name"])
	println("Age:", obj["age"])
}

package observability

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestMetricsCollectorCreation 测试指标收集器创建
func TestMetricsCollectorCreation(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		mc := NewMetricsCollector(MetricsConfig{})
		
		if mc.config.Namespace != "langchain" {
			t.Errorf("Expected namespace 'langchain', got '%s'", mc.config.Namespace)
		}
		
		if mc.config.Subsystem != "go" {
			t.Errorf("Expected subsystem 'go', got '%s'", mc.config.Subsystem)
		}
		
		if mc.config.HTTPPath != "/metrics" {
			t.Errorf("Expected path '/metrics', got '%s'", mc.config.HTTPPath)
		}
		
		if mc.config.HTTPPort != "9090" {
			t.Errorf("Expected port '9090', got '%s'", mc.config.HTTPPort)
		}
	})
	
	t.Run("custom config", func(t *testing.T) {
		mc := NewMetricsCollector(MetricsConfig{
			Namespace:            "test",
			Subsystem:            "app",
			EnableDefaultMetrics: true,
			HTTPPath:             "/custom_metrics",
			HTTPPort:             "8080",
		})
		
		if mc.config.Namespace != "test" {
			t.Errorf("Expected namespace 'test', got '%s'", mc.config.Namespace)
		}
		
		if mc.config.Subsystem != "app" {
			t.Errorf("Expected subsystem 'app', got '%s'", mc.config.Subsystem)
		}
	})
}

// TestLLMMetrics 测试 LLM 指标
func TestLLMMetrics(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	t.Run("successful call", func(t *testing.T) {
		mc.RecordLLMCall("openai", "gpt-4", 1*time.Second, nil)
		
		// 验证指标值（通过 HTTP handler）
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_llm_calls_total") {
			t.Error("Expected llm_calls_total metric")
		}
		
		if !strings.Contains(body, "langchain_go_llm_call_duration_seconds") {
			t.Error("Expected llm_call_duration_seconds metric")
		}
	})
	
	t.Run("failed call", func(t *testing.T) {
		mc.RecordLLMCall("openai", "gpt-4", 500*time.Millisecond, errors.New("test error"))
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_llm_errors_total") {
			t.Error("Expected llm_errors_total metric")
		}
	})
	
	t.Run("token usage", func(t *testing.T) {
		mc.RecordLLMTokens("openai", "gpt-4", 100, 50)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_llm_tokens_total") {
			t.Error("Expected llm_tokens_total metric")
		}
	})
}

// TestAgentMetrics 测试 Agent 指标
func TestAgentMetrics(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	t.Run("agent step", func(t *testing.T) {
		mc.RecordAgentStep("react", 2*time.Second, nil)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_agent_steps_total") {
			t.Error("Expected agent_steps_total metric")
		}
		
		if !strings.Contains(body, "langchain_go_agent_step_duration_seconds") {
			t.Error("Expected agent_step_duration_seconds metric")
		}
	})
	
	t.Run("agent iteration", func(t *testing.T) {
		mc.RecordAgentIteration("react")
		mc.RecordAgentIteration("react")
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_agent_iterations_total") {
			t.Error("Expected agent_iterations_total metric")
		}
	})
	
	t.Run("agent error", func(t *testing.T) {
		mc.RecordAgentStep("react", 1*time.Second, errors.New("test error"))
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_agent_errors_total") {
			t.Error("Expected agent_errors_total metric")
		}
	})
}

// TestToolMetrics 测试工具指标
func TestToolMetrics(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	t.Run("tool call", func(t *testing.T) {
		mc.RecordToolCall("calculator", 100*time.Millisecond, nil)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_tool_calls_total") {
			t.Error("Expected tool_calls_total metric")
		}
		
		if !strings.Contains(body, "langchain_go_tool_call_duration_seconds") {
			t.Error("Expected tool_call_duration_seconds metric")
		}
	})
	
	t.Run("tool error", func(t *testing.T) {
		mc.RecordToolCall("calculator", 50*time.Millisecond, errors.New("test error"))
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_tool_errors_total") {
			t.Error("Expected tool_errors_total metric")
		}
	})
}

// TestRAGMetrics 测试 RAG 指标
func TestRAGMetrics(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	t.Run("rag query", func(t *testing.T) {
		mc.RecordRAGQuery("milvus", 200*time.Millisecond, 10, nil)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_rag_queries_total") {
			t.Error("Expected rag_queries_total metric")
		}
		
		if !strings.Contains(body, "langchain_go_rag_query_duration_seconds") {
			t.Error("Expected rag_query_duration_seconds metric")
		}
		
		if !strings.Contains(body, "langchain_go_rag_documents_retrieved") {
			t.Error("Expected rag_documents_retrieved metric")
		}
	})
	
	t.Run("rag error", func(t *testing.T) {
		mc.RecordRAGQuery("milvus", 100*time.Millisecond, 0, errors.New("test error"))
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_rag_errors_total") {
			t.Error("Expected rag_errors_total metric")
		}
	})
}

// TestChainMetrics 测试 Chain 指标
func TestChainMetrics(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	t.Run("chain execution", func(t *testing.T) {
		mc.RecordChainExecution("my_chain", 3*time.Second, nil)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_chain_executions_total") {
			t.Error("Expected chain_executions_total metric")
		}
		
		if !strings.Contains(body, "langchain_go_chain_duration_seconds") {
			t.Error("Expected chain_duration_seconds metric")
		}
	})
	
	t.Run("chain error", func(t *testing.T) {
		mc.RecordChainExecution("my_chain", 1*time.Second, errors.New("test error"))
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_chain_errors_total") {
			t.Error("Expected chain_errors_total metric")
		}
	})
}

// TestMemoryMetrics 测试 Memory 指标
func TestMemoryMetrics(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	t.Run("memory operations", func(t *testing.T) {
		mc.RecordMemoryOperation("buffer", "load")
		mc.RecordMemoryOperation("buffer", "save")
		mc.RecordMemoryOperation("buffer", "clear")
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_memory_operations_total") {
			t.Error("Expected memory_operations_total metric")
		}
	})
	
	t.Run("memory size", func(t *testing.T) {
		mc.SetMemorySize("buffer", 10)
		mc.SetMemorySize("buffer", 20)
		
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		mc.Handler().ServeHTTP(w, req)
		
		body := w.Body.String()
		if !strings.Contains(body, "langchain_go_memory_size_messages") {
			t.Error("Expected memory_size_messages metric")
		}
	})
}

// TestHTTPHandler 测试 HTTP 处理器
func TestHTTPHandler(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	// 记录一些指标
	mc.RecordLLMCall("openai", "gpt-4", 1*time.Second, nil)
	mc.RecordAgentStep("react", 2*time.Second, nil)
	mc.RecordToolCall("calculator", 100*time.Millisecond, nil)
	
	t.Run("metrics endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		
		handler := mc.Handler()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		body := w.Body.String()
		if body == "" {
			t.Error("Expected non-empty response body")
		}
		
		// 验证包含 Prometheus 格式
		if !strings.Contains(body, "# HELP") {
			t.Error("Expected Prometheus HELP comments")
		}
		
		if !strings.Contains(body, "# TYPE") {
			t.Error("Expected Prometheus TYPE comments")
		}
	})
}

// TestMetricsRegistry 测试 Registry
func TestMetricsRegistry(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	registry := mc.GetRegistry()
	if registry == nil {
		t.Error("Expected non-nil registry")
	}
	
	// 记录一些指标以触发注册
	mc.RecordLLMCall("openai", "gpt-4", 1*time.Second, nil)
	mc.RecordAgentStep("react", 1*time.Second, nil)
	
	// 验证指标已注册
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}
	
	if len(metrics) == 0 {
		t.Error("Expected registered metrics")
	}
	
	// 查找特定指标
	foundLLM := false
	foundAgent := false
	for _, m := range metrics {
		if strings.Contains(*m.Name, "llm_calls_total") {
			foundLLM = true
		}
		if strings.Contains(*m.Name, "agent_steps_total") {
			foundAgent = true
		}
	}
	
	if !foundLLM {
		t.Error("Expected to find LLM metric")
	}
	
	if !foundAgent {
		t.Error("Expected to find Agent metric")
	}
}

// TestConcurrentMetrics 测试并发记录
func TestConcurrentMetrics(t *testing.T) {
	mc := NewMetricsCollector(MetricsConfig{})
	
	// 并发记录指标
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			mc.RecordLLMCall("openai", "gpt-4", 1*time.Second, nil)
			mc.RecordAgentStep("react", 1*time.Second, nil)
			mc.RecordToolCall("calculator", 100*time.Millisecond, nil)
			done <- true
		}()
	}
	
	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// 验证指标
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	mc.Handler().ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// +build postgres

package checkpoint

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestState 测试状态结构
type TestState struct {
	Step    int               `json:"step"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

// getTestConnString 获取测试连接字符串
func getTestConnString() string {
	connStr := os.Getenv("TEST_POSTGRES_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres123@localhost:5432/langchain_test?sslmode=disable"
	}
	return connStr
}

// TestPostgresCheckpointSaver_WithNamespace 测试命名空间支持
func TestPostgresCheckpointSaver_WithNamespace(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 创建带命名空间的配置
	config := NewCheckpointConfig("thread-1").
		WithNamespace("subgraph.level1")

	// 创建 checkpoint
	state := TestState{
		Step:    1,
		Message: "Test checkpoint with namespace",
		Data:    map[string]string{"key": "value"},
	}

	cp := NewCheckpoint("cp-1", state, config)
	cp.SetType("test")

	// 保存
	err = saver.Save(ctx, cp)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// 加载
	loaded, err := saver.Load(ctx, config)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	// 验证
	if loaded.ID != cp.ID {
		t.Errorf("Expected ID %s, got %s", cp.ID, loaded.ID)
	}
	if loaded.CheckpointNS != "subgraph.level1" {
		t.Errorf("Expected namespace subgraph.level1, got %s", loaded.CheckpointNS)
	}
	if loaded.Type != "test" {
		t.Errorf("Expected type test, got %s", loaded.Type)
	}
	if loaded.State.Step != state.Step {
		t.Errorf("Expected step %d, got %d", state.Step, loaded.State.Step)
	}

	// 清理
	saver.Delete(ctx, config.WithCheckpointID("cp-1"))
}

// TestPostgresCheckpointSaver_MultipleNamespaces 测试多个命名空间
func TestPostgresCheckpointSaver_MultipleNamespaces(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()
	threadID := "thread-multi"

	// 创建三个不同命名空间的 checkpoint
	namespaces := []string{"", "subgraph.level1", "subgraph.level1.level2"}

	for i, ns := range namespaces {
		config := NewCheckpointConfig(threadID).WithNamespace(ns)
		state := TestState{
			Step:    i + 1,
			Message: "Checkpoint in namespace: " + ns,
		}

		cp := NewCheckpoint("cp-"+string(rune('a'+i)), state, config)
		err = saver.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save checkpoint %d: %v", i, err)
		}
	}

	// 验证每个命名空间的 checkpoint 都独立存在
	for i, ns := range namespaces {
		config := NewCheckpointConfig(threadID).WithNamespace(ns)
		loaded, err := saver.Load(ctx, config)
		if err != nil {
			t.Fatalf("Failed to load checkpoint from namespace %s: %v", ns, err)
		}

		if loaded.CheckpointNS != ns {
			t.Errorf("Expected namespace %s, got %s", ns, loaded.CheckpointNS)
		}
		if loaded.State.Step != i+1 {
			t.Errorf("Expected step %d, got %d", i+1, loaded.State.Step)
		}
	}

	// 清理
	for i, ns := range namespaces {
		config := NewCheckpointConfig(threadID).
			WithNamespace(ns).
			WithCheckpointID("cp-" + string(rune('a'+i)))
		saver.Delete(ctx, config)
	}
}

// TestPostgresCheckpointSaver_TypeField 测试 Type 字段
func TestPostgresCheckpointSaver_TypeField(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	config := NewCheckpointConfig("thread-type")
	state := TestState{Step: 1, Message: "Test type"}

	cp := NewCheckpoint("cp-type", state, config)
	cp.SetType("test-state")

	// 保存
	err = saver.Save(ctx, cp)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// 加载
	loaded, err := saver.Load(ctx, config)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	// 验证 Type
	if loaded.Type != "test-state" {
		t.Errorf("Expected type test-state, got %s", loaded.Type)
	}

	// 清理
	saver.Delete(ctx, config.WithCheckpointID("cp-type"))
}

// TestPostgresCheckpointSaver_BackwardCompatibility 测试向后兼容性
func TestPostgresCheckpointSaver_BackwardCompatibility(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 测试没有 namespace 的旧行为（默认为空字符串）
	config := NewCheckpointConfig("thread-compat")
	// 不设置 namespace，应该默认为空字符串

	state := TestState{Step: 1, Message: "Backward compatible"}
	cp := NewCheckpoint("cp-compat", state, config)

	// 保存
	err = saver.Save(ctx, cp)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// 加载
	loaded, err := saver.Load(ctx, config)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	// 验证 namespace 为空字符串
	if loaded.CheckpointNS != "" {
		t.Errorf("Expected empty namespace, got %s", loaded.CheckpointNS)
	}

	// 清理
	saver.Delete(ctx, config.WithCheckpointID("cp-compat"))
}

// TestPostgresCheckpointSaver_ComplexNamespace 测试复杂命名空间
func TestPostgresCheckpointSaver_ComplexNamespace(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 测试深层嵌套命名空间
	config := NewCheckpointConfig("thread-complex").
		WithNamespace("main.sub1.sub2.sub3")

	state := TestState{
		Step:    1,
		Message: "Deep nested namespace",
		Data:    map[string]string{"level": "4"},
	}

	cp := NewCheckpoint("cp-complex", state, config)

	// 保存
	err = saver.Save(ctx, cp)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// 加载
	loaded, err := saver.Load(ctx, config)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	// 验证
	if loaded.CheckpointNS != "main.sub1.sub2.sub3" {
		t.Errorf("Expected namespace main.sub1.sub2.sub3, got %s", loaded.CheckpointNS)
	}

	// 清理
	saver.Delete(ctx, config.WithCheckpointID("cp-complex"))
}

// BenchmarkPostgresCheckpointSaver_SaveWithNamespace 性能测试
func BenchmarkPostgresCheckpointSaver_SaveWithNamespace(b *testing.B) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		b.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()
	config := NewCheckpointConfig("thread-bench").
		WithNamespace("benchmark")

	state := TestState{
		Step:    1,
		Message: "Benchmark checkpoint",
		Data:    make(map[string]string),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := config.WithCheckpointID("cp-" + string(rune(i)))
		cp := NewCheckpoint("cp-"+string(rune(i)), state, config)
		saver.Save(ctx, cp)
	}
}

// BenchmarkPostgresCheckpointSaver_LoadWithNamespace 加载性能测试
func BenchmarkPostgresCheckpointSaver_LoadWithNamespace(b *testing.B) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		b.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()
	config := NewCheckpointConfig("thread-bench-load").
		WithNamespace("benchmark")

	// 准备数据
	state := TestState{Step: 1, Message: "Benchmark"}
	cp := NewCheckpoint("cp-bench", state, config)
	saver.Save(ctx, cp)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		saver.Load(ctx, config)
	}

	// 清理
	saver.Delete(ctx, config.WithCheckpointID("cp-bench"))
}

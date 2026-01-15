// +build sqlite

package checkpoint

import (
	"context"
	"os"
	"testing"
	"time"
)

// getSQLiteTestDB 获取测试数据库
func getSQLiteTestDB() string {
	// 使用临时内存数据库进行测试
	return ":memory:"
}

// TestSQLiteCheckpointSaver_ThreeTableSchema 测试三表Schema创建
func TestSQLiteCheckpointSaver_ThreeTableSchema(t *testing.T) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		t.Fatalf("Failed to create saver: %v", err)
	}
	defer saver.Close()

	// 验证三张表都已创建
	tables := []string{"checkpoints", "checkpoint_blobs", "checkpoint_writes"}
	for _, table := range tables {
		var name string
		query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
		err := saver.db.QueryRow(query, table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
	}
}

// TestSQLiteCheckpointSaver_WithNamespace 测试命名空间支持
func TestSQLiteCheckpointSaver_WithNamespace(t *testing.T) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		t.Fatalf("Failed to create saver: %v", err)
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
	if loaded.CheckpointNS != "subgraph.level1" {
		t.Errorf("Expected namespace subgraph.level1, got %s", loaded.CheckpointNS)
	}
	if loaded.Type != "test" {
		t.Errorf("Expected type test, got %s", loaded.Type)
	}

	// 清理
	saver.Delete(ctx, config.WithCheckpointID("cp-1"))
}

// TestSQLiteCheckpointSaver_SaveWrite 测试写入记录
func TestSQLiteCheckpointSaver_SaveWrite(t *testing.T) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		t.Fatalf("Failed to create saver: %v", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 创建写入记录
	write := NewCheckpointWrite("thread-1", "", "cp-1", "task-1", "state", 0).
		WithType("update").
		WithValue("step", 1).
		WithValue("message", "test write")

	// 保存
	err = saver.SaveWrite(ctx, write)
	if err != nil {
		t.Fatalf("Failed to save write: %v", err)
	}

	// 列出并验证
	writes, err := saver.ListWrites(ctx, "thread-1", "", "cp-1")
	if err != nil {
		t.Fatalf("Failed to list writes: %v", err)
	}

	if len(writes) != 1 {
		t.Errorf("Expected 1 write, got %d", len(writes))
	}

	if writes[0].Value["step"].(float64) != 1 {
		t.Errorf("Expected step=1")
	}

	// 清理
	saver.DeleteWrites(ctx, "thread-1", "", "cp-1")
}

// TestSQLiteCheckpointSaver_SaveBlob 测试 Blob 保存
func TestSQLiteCheckpointSaver_SaveBlob(t *testing.T) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		t.Fatalf("Failed to create saver: %v", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 创建 Blob 数据
	blob := &CheckpointBlob{
		ThreadID:     "thread-1",
		CheckpointNS: "",
		Channel:      "state",
		Version:      "v1",
		Type:         "test-state",
		Data:         []byte("large data content here..."),
		CreatedAt:    time.Now(),
	}

	// 保存
	err = saver.SaveBlob(ctx, blob)
	if err != nil {
		t.Fatalf("Failed to save blob: %v", err)
	}

	// 加载并验证
	loaded, err := saver.LoadBlob(ctx, "thread-1", "", "state", "v1")
	if err != nil {
		t.Fatalf("Failed to load blob: %v", err)
	}

	if loaded.Type != "test-state" {
		t.Errorf("Expected type=test-state, got %s", loaded.Type)
	}

	if string(loaded.Data) != "large data content here..." {
		t.Errorf("Data mismatch")
	}

	// 清理
	saver.DeleteBlob(ctx, "thread-1", "", "state", "v1")
}

// TestSQLiteCheckpointSaver_MultipleNamespaces 测试多个命名空间
func TestSQLiteCheckpointSaver_MultipleNamespaces(t *testing.T) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		t.Fatalf("Failed to create saver: %v", err)
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

// TestSQLiteCheckpointSaver_CompleteWorkflow 测试完整工作流
func TestSQLiteCheckpointSaver_CompleteWorkflow(t *testing.T) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		t.Fatalf("Failed to create saver: %v", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 1. 创建 checkpoint
	config := NewCheckpointConfig("thread-workflow")
	state := TestState{
		Step:    1,
		Message: "Initial state",
		Data:    map[string]string{"key": "value"},
	}

	cp := NewCheckpoint("cp-1", state, config)
	cp.SetType("workflow-state")

	err = saver.Save(ctx, cp)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// 2. 保存写入记录
	write1 := NewCheckpointWrite("thread-workflow", "", "cp-1", "task-1", "state", 0).
		WithType("init").
		WithValue("action", "initialize")

	write2 := NewCheckpointWrite("thread-workflow", "", "cp-1", "task-1", "state", 1).
		WithType("update").
		WithValue("action", "process")

	err = saver.SaveWrite(ctx, write1)
	if err != nil {
		t.Fatalf("Failed to save write1: %v", err)
	}

	err = saver.SaveWrite(ctx, write2)
	if err != nil {
		t.Fatalf("Failed to save write2: %v", err)
	}

	// 3. 保存 Blob
	blob := &CheckpointBlob{
		ThreadID:     "thread-workflow",
		CheckpointNS: "",
		Channel:      "large-data",
		Version:      "cp-1",
		Type:         "binary",
		Data:         []byte("This is a large data blob..."),
	}

	err = saver.SaveBlob(ctx, blob)
	if err != nil {
		t.Fatalf("Failed to save blob: %v", err)
	}

	// 4. 验证 checkpoint
	loaded, err := saver.Load(ctx, config)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	if loaded.ID != "cp-1" {
		t.Errorf("Expected cp-1, got %s", loaded.ID)
	}

	// 5. 验证写入记录
	writes, err := saver.ListWrites(ctx, "thread-workflow", "", "cp-1")
	if err != nil {
		t.Fatalf("Failed to list writes: %v", err)
	}

	if len(writes) != 2 {
		t.Errorf("Expected 2 writes, got %d", len(writes))
	}

	// 6. 验证 Blob
	loadedBlob, err := saver.LoadBlob(ctx, "thread-workflow", "", "large-data", "cp-1")
	if err != nil {
		t.Fatalf("Failed to load blob: %v", err)
	}

	if loadedBlob.Type != "binary" {
		t.Errorf("Expected type=binary, got %s", loadedBlob.Type)
	}

	// 7. 清理
	saver.Delete(ctx, config.WithCheckpointID("cp-1"))
	saver.DeleteWrites(ctx, "thread-workflow", "", "cp-1")
	saver.DeleteBlob(ctx, "thread-workflow", "", "large-data", "cp-1")
}

// TestSQLiteCheckpointSaver_BackwardCompatibility 测试向后兼容性
func TestSQLiteCheckpointSaver_BackwardCompatibility(t *testing.T) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		t.Fatalf("Failed to create saver: %v", err)
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

// BenchmarkSQLiteCheckpointSaver_SaveWrite 写入性能测试
func BenchmarkSQLiteCheckpointSaver_SaveWrite(b *testing.B) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		b.Skip("SQLite not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		write := NewCheckpointWrite("thread-bench", "", "cp-1", "task-1", "state", i).
			WithType("update").
			WithValue("index", i)

		saver.SaveWrite(ctx, write)
	}
}

// BenchmarkSQLiteCheckpointSaver_SaveBlob Blob 性能测试
func BenchmarkSQLiteCheckpointSaver_SaveBlob(b *testing.B) {
	saver, err := NewSQLiteCheckpointSaver[TestState](getSQLiteTestDB())
	if err != nil {
		b.Skip("SQLite not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()
	data := make([]byte, 1024*10) // 10KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blob := &CheckpointBlob{
			ThreadID:     "thread-bench",
			CheckpointNS: "",
			Channel:      "state",
			Version:      "v" + string(rune(i)),
			Type:         "binary",
			Data:         data,
		}

		saver.SaveBlob(ctx, blob)
	}
}

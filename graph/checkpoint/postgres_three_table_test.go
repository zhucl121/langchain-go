// +build postgres

package checkpoint

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestPostgresCheckpointSaver_ThreeTableSchema 测试三表Schema创建
func TestPostgresCheckpointSaver_ThreeTableSchema(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	// 验证三张表都已创建
	tables := []string{"checkpoints", "checkpoint_blobs", "checkpoint_writes"}
	for _, table := range tables {
		var exists bool
		query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = $1
		)`
		err := saver.db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

// TestPostgresCheckpointSaver_SaveWrite 测试写入记录保存
func TestPostgresCheckpointSaver_SaveWrite(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
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

	// 加载并验证
	writes, err := saver.ListWrites(ctx, "thread-1", "", "cp-1")
	if err != nil {
		t.Fatalf("Failed to list writes: %v", err)
	}

	if len(writes) != 1 {
		t.Errorf("Expected 1 write, got %d", len(writes))
	}

	if writes[0].TaskID != "task-1" {
		t.Errorf("Expected task-1, got %s", writes[0].TaskID)
	}

	if writes[0].Value["step"].(float64) != 1 {
		t.Errorf("Expected step=1, got %v", writes[0].Value["step"])
	}

	// 清理
	saver.DeleteWrites(ctx, "thread-1", "", "cp-1")
}

// TestPostgresCheckpointSaver_MultipleWrites 测试多个写入记录
func TestPostgresCheckpointSaver_MultipleWrites(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 创建多个写入记录(不同索引)
	for i := 0; i < 5; i++ {
		write := NewCheckpointWrite("thread-multi", "", "cp-1", "task-1", "state", i).
			WithType("update").
			WithValue("index", i)

		err = saver.SaveWrite(ctx, write)
		if err != nil {
			t.Fatalf("Failed to save write %d: %v", i, err)
		}
	}

	// 加载并验证顺序
	writes, err := saver.ListWrites(ctx, "thread-multi", "", "cp-1")
	if err != nil {
		t.Fatalf("Failed to list writes: %v", err)
	}

	if len(writes) != 5 {
		t.Errorf("Expected 5 writes, got %d", len(writes))
	}

	// 验证按 idx 排序
	for i, write := range writes {
		if write.Idx != i {
			t.Errorf("Expected idx=%d, got %d", i, write.Idx)
		}
	}

	// 清理
	saver.DeleteWrites(ctx, "thread-multi", "", "cp-1")
}

// TestPostgresCheckpointSaver_SaveBlob 测试 Blob 保存
func TestPostgresCheckpointSaver_SaveBlob(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
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

// TestPostgresCheckpointSaver_BlobWithNamespace 测试带命名空间的 Blob
func TestPostgresCheckpointSaver_BlobWithNamespace(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 不同命名空间的 Blob
	namespaces := []string{"", "sub1", "sub1.sub2"}

	for _, ns := range namespaces {
		blob := &CheckpointBlob{
			ThreadID:     "thread-1",
			CheckpointNS: ns,
			Channel:      "state",
			Version:      "v1",
			Type:         "test",
			Data:         []byte("data for namespace: " + ns),
		}

		err = saver.SaveBlob(ctx, blob)
		if err != nil {
			t.Fatalf("Failed to save blob for ns=%s: %v", ns, err)
		}
	}

	// 验证每个命名空间的 Blob 都独立存在
	for _, ns := range namespaces {
		loaded, err := saver.LoadBlob(ctx, "thread-1", ns, "state", "v1")
		if err != nil {
			t.Fatalf("Failed to load blob for ns=%s: %v", ns, err)
		}

		expectedData := "data for namespace: " + ns
		if string(loaded.Data) != expectedData {
			t.Errorf("Expected data '%s', got '%s'", expectedData, string(loaded.Data))
		}
	}

	// 清理
	for _, ns := range namespaces {
		saver.DeleteBlob(ctx, "thread-1", ns, "state", "v1")
	}
}

// TestPostgresCheckpointSaver_WriteWithNamespace 测试带命名空间的写入
func TestPostgresCheckpointSaver_WriteWithNamespace(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	defer saver.Close()

	ctx := context.Background()

	// 不同命名空间的写入
	namespaces := []string{"", "subgraph", "subgraph.level1"}

	for i, ns := range namespaces {
		write := NewCheckpointWrite("thread-1", ns, "cp-1", "task-1", "state", i).
			WithType("update").
			WithValue("namespace", ns)

		err = saver.SaveWrite(ctx, write)
		if err != nil {
			t.Fatalf("Failed to save write for ns=%s: %v", ns, err)
		}
	}

	// 验证每个命名空间的写入都独立
	for _, ns := range namespaces {
		writes, err := saver.ListWrites(ctx, "thread-1", ns, "cp-1")
		if err != nil {
			t.Fatalf("Failed to list writes for ns=%s: %v", ns, err)
		}

		// 每个命名空间应该只有一条写入
		if len(writes) != 1 {
			t.Errorf("Expected 1 write for ns=%s, got %d", ns, len(writes))
		}

		if writes[0].CheckpointNS != ns {
			t.Errorf("Expected namespace=%s, got %s", ns, writes[0].CheckpointNS)
		}
	}

	// 清理
	for _, ns := range namespaces {
		saver.DeleteWrites(ctx, "thread-1", ns, "cp-1")
	}
}

// TestPostgresCheckpointSaver_CompleteWorkflow 测试完整工作流
func TestPostgresCheckpointSaver_CompleteWorkflow(t *testing.T) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
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

	// 3. 保存 Blob(模拟大数据)
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

// BenchmarkPostgresCheckpointSaver_SaveWrite 写入性能测试
func BenchmarkPostgresCheckpointSaver_SaveWrite(b *testing.B) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		b.Skip("PostgreSQL not available:", err)
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

// BenchmarkPostgresCheckpointSaver_SaveBlob 写入 Blob 性能测试
func BenchmarkPostgresCheckpointSaver_SaveBlob(b *testing.B) {
	saver, err := NewPostgresCheckpointSaver[TestState](getTestConnString())
	if err != nil {
		b.Skip("PostgreSQL not available:", err)
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

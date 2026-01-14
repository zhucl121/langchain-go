package checkpoint

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestState 测试用状态
type TestState struct {
	Counter int
	Message string
}

// TestNewCheckpointConfig 测试创建配置
func TestNewCheckpointConfig(t *testing.T) {
	config := NewCheckpointConfig("thread-1")

	if config.ThreadID != "thread-1" {
		t.Errorf("expected ThreadID 'thread-1', got %s", config.ThreadID)
	}

	if config.Metadata == nil {
		t.Error("expected Metadata to be initialized")
	}
}

// TestCheckpointConfig_WithMethods 测试配置链式调用
func TestCheckpointConfig_WithMethods(t *testing.T) {
	config := NewCheckpointConfig("thread-1").
		WithCheckpointID("cp-1").
		WithMetadata("key", "value")

	if config.CheckpointID != "cp-1" {
		t.Errorf("expected CheckpointID 'cp-1', got %s", config.CheckpointID)
	}

	if config.Metadata["key"] != "value" {
		t.Error("expected metadata key to be 'value'")
	}
}

// TestCheckpointConfig_Validate 测试配置验证
func TestCheckpointConfig_Validate(t *testing.T) {
	validConfig := NewCheckpointConfig("thread-1")
	if err := validConfig.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	invalidConfig := NewCheckpointConfig("")
	if !errors.Is(invalidConfig.Validate(), ErrInvalidConfig) {
		t.Error("expected ErrInvalidConfig")
	}
}

// TestNewCheckpoint 测试创建检查点
func TestNewCheckpoint(t *testing.T) {
	state := TestState{Counter: 5, Message: "test"}
	config := NewCheckpointConfig("thread-1")
	checkpoint := NewCheckpoint("cp-1", state, config)

	if checkpoint.ID != "cp-1" {
		t.Errorf("expected ID 'cp-1', got %s", checkpoint.ID)
	}

	if checkpoint.ThreadID != "thread-1" {
		t.Errorf("expected ThreadID 'thread-1', got %s", checkpoint.ThreadID)
	}

	if checkpoint.State.Counter != 5 {
		t.Errorf("expected Counter 5, got %d", checkpoint.State.Counter)
	}
}

// TestCheckpoint_Clone 测试克隆检查点
func TestCheckpoint_Clone(t *testing.T) {
	original := NewCheckpoint("cp-1", TestState{Counter: 5}, NewCheckpointConfig("thread-1"))
	original.Metadata["key"] = "value"

	clone := original.Clone()

	if clone.ID != original.ID {
		t.Error("cloned ID mismatch")
	}

	if clone.State.Counter != original.State.Counter {
		t.Error("cloned State mismatch")
	}

	// 修改克隆不应影响原始
	clone.Metadata["key2"] = "value2"
	if _, exists := original.Metadata["key2"]; exists {
		t.Error("modifying clone affected original")
	}
}

// TestSerializableCheckpoint 测试序列化
func TestSerializableCheckpoint(t *testing.T) {
	checkpoint := NewCheckpoint("cp-1", TestState{Counter: 5}, NewCheckpointConfig("thread-1"))

	// 转换为可序列化格式
	scp, err := ToSerializable(checkpoint)
	if err != nil {
		t.Fatalf("ToSerializable failed: %v", err)
	}

	if scp.ID != "cp-1" {
		t.Errorf("expected ID 'cp-1', got %s", scp.ID)
	}

	// 从可序列化格式转换回来
	restored, err := FromSerializable[TestState](scp)
	if err != nil {
		t.Fatalf("FromSerializable failed: %v", err)
	}

	if restored.State.Counter != 5 {
		t.Errorf("expected Counter 5, got %d", restored.State.Counter)
	}
}

// TestCheckpointMetadata 测试元数据
func TestCheckpointMetadata(t *testing.T) {
	metadata := NewCheckpointMetadata().
		WithSource("manual").
		WithStep(10).
		WithNodeName("node1").
		WithDescription("test checkpoint")

	if metadata.Source != "manual" {
		t.Errorf("expected Source 'manual', got %s", metadata.Source)
	}

	if metadata.Step != 10 {
		t.Errorf("expected Step 10, got %d", metadata.Step)
	}

	m := metadata.ToMap()
	if m["source"] != "manual" {
		t.Error("ToMap failed for source")
	}
}

// TestMemoryCheckpointSaver_Basic 测试内存保存器基本功能
func TestMemoryCheckpointSaver_Basic(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()

	state := TestState{Counter: 5, Message: "test"}
	config := NewCheckpointConfig("thread-1")
	checkpoint := NewCheckpoint("cp-1", state, config)

	// 保存
	err := saver.Save(context.Background(), checkpoint)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 加载
	loaded, err := saver.Load(context.Background(), config.WithCheckpointID("cp-1"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.State.Counter != 5 {
		t.Errorf("expected Counter 5, got %d", loaded.State.Counter)
	}
}

// TestMemoryCheckpointSaver_LoadLatest 测试加载最新检查点
func TestMemoryCheckpointSaver_LoadLatest(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()

	// 保存多个检查点
	for i := 1; i <= 3; i++ {
		state := TestState{Counter: i}
		config := NewCheckpointConfig("thread-1")
		checkpoint := NewCheckpoint(string(rune('a'+i-1)), state, config)
		checkpoint.Timestamp = time.Now().Add(time.Duration(i) * time.Second)

		if err := saver.Save(context.Background(), checkpoint); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	}

	// 加载最新（不指定 CheckpointID）
	config := NewCheckpointConfig("thread-1")
	latest, err := saver.Load(context.Background(), config)
	if err != nil {
		t.Fatalf("Load latest failed: %v", err)
	}

	// 应该是最后保存的
	if latest.State.Counter != 3 {
		t.Errorf("expected latest Counter 3, got %d", latest.State.Counter)
	}
}

// TestMemoryCheckpointSaver_List 测试列出检查点
func TestMemoryCheckpointSaver_List(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()

	// 保存多个检查点
	for i := 1; i <= 3; i++ {
		state := TestState{Counter: i}
		config := NewCheckpointConfig("thread-1")
		checkpoint := NewCheckpoint(string(rune('a'+i-1)), state, config)

		if err := saver.Save(context.Background(), checkpoint); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// 列出检查点
	checkpoints, err := saver.List(context.Background(), "thread-1")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(checkpoints) != 3 {
		t.Errorf("expected 3 checkpoints, got %d", len(checkpoints))
	}
}

// TestMemoryCheckpointSaver_Delete 测试删除检查点
func TestMemoryCheckpointSaver_Delete(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()

	state := TestState{Counter: 5}
	config := NewCheckpointConfig("thread-1")
	checkpoint := NewCheckpoint("cp-1", state, config)

	// 保存
	if err := saver.Save(context.Background(), checkpoint); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 删除
	err := saver.Delete(context.Background(), config.WithCheckpointID("cp-1"))
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证已删除
	_, err = saver.Load(context.Background(), config.WithCheckpointID("cp-1"))
	if !errors.Is(err, ErrCheckpointNotFound) {
		t.Error("expected ErrCheckpointNotFound after deletion")
	}
}

// TestMemoryCheckpointSaver_MultiThread 测试多线程
func TestMemoryCheckpointSaver_MultiThread(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()

	// 为不同线程保存检查点
	for thread := 1; thread <= 3; thread++ {
		state := TestState{Counter: thread}
		threadID := fmt.Sprintf("thread-%d", thread)
		config := NewCheckpointConfig(threadID)
		checkpointID := fmt.Sprintf("cp-%d", thread)
		checkpoint := NewCheckpoint(checkpointID, state, config)

		if err := saver.Save(context.Background(), checkpoint); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// 验证统计
	stats := saver.GetStats()
	if stats["total_checkpoints"] != 3 {
		t.Errorf("expected 3 checkpoints, got %d", stats["total_checkpoints"])
	}

	if stats["total_threads"] != 3 {
		t.Errorf("expected 3 threads, got %d", stats["total_threads"])
	}
}

// TestMemoryCheckpointSaver_Clear 测试清空
func TestMemoryCheckpointSaver_Clear(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()

	// 保存检查点
	state := TestState{Counter: 5}
	config := NewCheckpointConfig("thread-1")
	checkpoint := NewCheckpoint("cp-1", state, config)

	if err := saver.Save(context.Background(), checkpoint); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 清空
	saver.Clear()

	// 验证已清空
	stats := saver.GetStats()
	if stats["total_checkpoints"] != 0 {
		t.Error("expected checkpoints to be cleared")
	}
}

// TestCheckpointManager_Basic 测试管理器基本功能
func TestCheckpointManager_Basic(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()
	manager := NewCheckpointManager(saver)

	state := TestState{Counter: 5}
	config := NewCheckpointConfig("thread-1")

	// 保存（自动生成 ID）
	checkpoint, err := manager.SaveCheckpoint(context.Background(), state, config)
	if err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	if checkpoint.ID == "" {
		t.Error("expected auto-generated ID")
	}

	// 加载
	loaded, err := manager.LoadCheckpoint(context.Background(), config.WithCheckpointID(checkpoint.ID))
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}

	if loaded.State.Counter != 5 {
		t.Errorf("expected Counter 5, got %d", loaded.State.Counter)
	}
}

// TestCheckpointManager_AutoSave 测试自动保存
func TestCheckpointManager_AutoSave(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()
	manager := NewCheckpointManager(saver)

	state := TestState{Counter: 5}

	// 自动保存
	checkpoint, err := manager.AutoSave(context.Background(), state, "thread-1", 10)
	if err != nil {
		t.Fatalf("AutoSave failed: %v", err)
	}

	// 验证元数据
	if checkpoint.Metadata["source"] != "auto" {
		t.Error("expected source to be 'auto'")
	}

	if checkpoint.Metadata["step"] != 10 {
		t.Error("expected step to be 10")
	}
}

// TestCheckpointManager_GetLatest 测试获取最新
func TestCheckpointManager_GetLatest(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()
	manager := NewCheckpointManager(saver)

	// 保存多个检查点
	for i := 1; i <= 3; i++ {
		state := TestState{Counter: i}
		_, err := manager.AutoSave(context.Background(), state, "thread-1", i)
		if err != nil {
			t.Fatalf("AutoSave failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 获取最新
	latest, err := manager.GetLatestCheckpoint(context.Background(), "thread-1")
	if err != nil {
		t.Fatalf("GetLatestCheckpoint failed: %v", err)
	}

	if latest.State.Counter != 3 {
		t.Errorf("expected latest Counter 3, got %d", latest.State.Counter)
	}
}

// TestCheckpointManager_Prune 测试清理
func TestCheckpointManager_Prune(t *testing.T) {
	saver := NewMemoryCheckpointSaver[TestState]()
	manager := NewCheckpointManager(saver)

	// 保存 5 个检查点
	for i := 1; i <= 5; i++ {
		state := TestState{Counter: i}
		_, err := manager.AutoSave(context.Background(), state, "thread-1", i)
		if err != nil {
			t.Fatalf("AutoSave failed: %v", err)
		}
	}

	// 清理，保留最近 2 个
	deleted, err := manager.PruneOldCheckpoints(context.Background(), "thread-1", 2)
	if err != nil {
		t.Fatalf("PruneOldCheckpoints failed: %v", err)
	}

	if deleted != 3 {
		t.Errorf("expected 3 deleted, got %d", deleted)
	}

	// 验证剩余 2 个
	checkpoints, _ := manager.ListCheckpoints(context.Background(), "thread-1")
	if len(checkpoints) != 2 {
		t.Errorf("expected 2 remaining checkpoints, got %d", len(checkpoints))
	}
}

// TestCheckpointIterator 测试迭代器
func TestCheckpointIterator(t *testing.T) {
	checkpoints := []*Checkpoint[TestState]{
		NewCheckpoint("cp-1", TestState{Counter: 1}, NewCheckpointConfig("thread-1")),
		NewCheckpoint("cp-2", TestState{Counter: 2}, NewCheckpointConfig("thread-1")),
		NewCheckpoint("cp-3", TestState{Counter: 3}, NewCheckpointConfig("thread-1")),
	}

	iterator := NewCheckpointIterator(checkpoints)

	// 当前应该是最新的（cp-3）
	current := iterator.Current()
	if current.State.Counter != 3 {
		t.Errorf("expected current Counter 3, got %d", current.State.Counter)
	}

	// 向前（到 cp-2）
	if !iterator.Prev() {
		t.Fatal("Prev() should return true")
	}

	current = iterator.Current()
	if current.State.Counter != 2 {
		t.Errorf("expected current Counter 2, got %d", current.State.Counter)
	}

	// 向后（到 cp-3）
	if !iterator.Next() {
		t.Fatal("Next() should return true")
	}

	current = iterator.Current()
	if current.State.Counter != 3 {
		t.Errorf("expected current Counter 3, got %d", current.State.Counter)
	}

	// 无法再向后
	if iterator.Next() {
		t.Error("Next() should return false at the end")
	}
}

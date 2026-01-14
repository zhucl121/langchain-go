package state

import (
	"testing"
)

// TestLastValueChannel 测试覆盖通道
func TestLastValueChannel(t *testing.T) {
	channel := NewLastValueChannel("status")

	if channel.GetName() != "status" {
		t.Errorf("expected name 'status', got %s", channel.GetName())
	}

	// 测试更新
	result, err := channel.Update("old", "new")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if result != "new" {
		t.Errorf("expected 'new', got %v", result)
	}
}

// TestLastValueChannel_NilCurrent 测试 nil 当前值
func TestLastValueChannel_NilCurrent(t *testing.T) {
	channel := NewLastValueChannel("test")

	result, err := channel.Update(nil, "value")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if result != "value" {
		t.Errorf("expected 'value', got %v", result)
	}
}

// TestAppendChannel 测试追加通道
func TestAppendChannel(t *testing.T) {
	channel := NewAppendChannel("messages")

	if channel.GetName() != "messages" {
		t.Errorf("expected name 'messages', got %s", channel.GetName())
	}

	// 测试追加到现有切片
	current := []any{"a", "b"}
	result, err := channel.Update(current, "c")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	resultSlice, ok := result.([]any)
	if !ok {
		t.Fatal("result is not a slice")
	}

	if len(resultSlice) != 3 {
		t.Errorf("expected length 3, got %d", len(resultSlice))
	}

	if resultSlice[2] != "c" {
		t.Errorf("expected last element 'c', got %v", resultSlice[2])
	}
}

// TestAppendChannel_NilCurrent 测试 nil 当前值
func TestAppendChannel_NilCurrent(t *testing.T) {
	channel := NewAppendChannel("test")

	result, err := channel.Update(nil, "value")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	resultSlice, ok := result.([]any)
	if !ok {
		t.Fatal("result is not a slice")
	}

	if len(resultSlice) != 1 {
		t.Errorf("expected length 1, got %d", len(resultSlice))
	}

	if resultSlice[0] != "value" {
		t.Errorf("expected 'value', got %v", resultSlice[0])
	}
}

// TestAppendChannel_NotSlice 测试非切片当前值
func TestAppendChannel_NotSlice(t *testing.T) {
	channel := NewAppendChannel("test")

	// 传入非切片值，应该创建新切片
	result, err := channel.Update("not_a_slice", "value")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	resultSlice, ok := result.([]any)
	if !ok {
		t.Fatal("result is not a slice")
	}

	if len(resultSlice) != 1 {
		t.Errorf("expected length 1, got %d", len(resultSlice))
	}

	if resultSlice[0] != "value" {
		t.Errorf("expected 'value', got %v", resultSlice[0])
	}
}

// TestLastValueReducer 测试覆盖归约器
func TestLastValueReducer(t *testing.T) {
	reducer := LastValueReducer[int]()

	// 测试无更新
	result := reducer(10)
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}

	// 测试单个更新
	result = reducer(10, 20)
	if result != 20 {
		t.Errorf("expected 20, got %d", result)
	}

	// 测试多个更新（返回最后一个）
	result = reducer(10, 20, 30, 40)
	if result != 40 {
		t.Errorf("expected 40, got %d", result)
	}
}

// TestLastValueReducer_String 测试字符串类型
func TestLastValueReducer_String(t *testing.T) {
	reducer := LastValueReducer[string]()

	result := reducer("old", "new1", "new2")
	if result != "new2" {
		t.Errorf("expected 'new2', got %s", result)
	}
}

// TestMergeReducer 测试合并归约器
func TestMergeReducer(t *testing.T) {
	reducer := MergeReducer()

	// 测试 nil 当前值
	m1 := map[string]any{"a": 1, "b": 2}
	result := reducer(nil, m1)

	if len(result) != 2 {
		t.Errorf("expected length 2, got %d", len(result))
	}

	if result["a"] != 1 || result["b"] != 2 {
		t.Error("unexpected result values")
	}
}

// TestMergeReducer_Overwrite 测试覆盖
func TestMergeReducer_Overwrite(t *testing.T) {
	reducer := MergeReducer()

	current := map[string]any{"a": 1, "b": 2}
	update1 := map[string]any{"b": 3, "c": 4}
	update2 := map[string]any{"a": 5}

	result := reducer(current, update1, update2)

	expected := map[string]any{"a": 5, "b": 3, "c": 4}

	if len(result) != len(expected) {
		t.Errorf("expected length %d, got %d", len(expected), len(result))
	}

	for k, expectedVal := range expected {
		if result[k] != expectedVal {
			t.Errorf("expected %s=%v, got %v", k, expectedVal, result[k])
		}
	}
}

// TestMergeReducer_NilUpdates 测试 nil 更新
func TestMergeReducer_NilUpdates(t *testing.T) {
	reducer := MergeReducer()

	current := map[string]any{"a": 1}
	result := reducer(current, nil, nil)

	if len(result) != 1 {
		t.Errorf("expected length 1, got %d", len(result))
	}

	if result["a"] != 1 {
		t.Error("unexpected result value")
	}
}

// TestAppendReducer 测试追加归约器
func TestAppendReducer(t *testing.T) {
	reducer := AppendReducer()

	// 测试 nil 当前值
	s1 := []any{1, 2}
	result := reducer(nil, s1)

	if len(result) != 2 {
		t.Errorf("expected length 2, got %d", len(result))
	}

	if result[0] != 1 || result[1] != 2 {
		t.Error("unexpected result values")
	}
}

// TestAppendReducer_Multiple 测试多个更新
func TestAppendReducer_Multiple(t *testing.T) {
	reducer := AppendReducer()

	current := []any{1, 2}
	s1 := []any{3, 4}
	s2 := []any{5}

	result := reducer(current, s1, s2)

	expected := []any{1, 2, 3, 4, 5}

	if len(result) != len(expected) {
		t.Errorf("expected length %d, got %d", len(expected), len(result))
	}

	for i, expectedVal := range expected {
		if result[i] != expectedVal {
			t.Errorf("expected [%d]=%v, got %v", i, expectedVal, result[i])
		}
	}
}

// TestAppendReducer_NilUpdates 测试 nil 更新
func TestAppendReducer_NilUpdates(t *testing.T) {
	reducer := AppendReducer()

	current := []any{1, 2}
	result := reducer(current, nil, nil)

	if len(result) != 2 {
		t.Errorf("expected length 2, got %d", len(result))
	}
}

// TestSumReducer_Int 测试整数求和
func TestSumReducer_Int(t *testing.T) {
	reducer := SumReducer[int]()

	// 测试无更新
	result := reducer(10)
	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}

	// 测试单个更新
	result = reducer(10, 5)
	if result != 15 {
		t.Errorf("expected 15, got %d", result)
	}

	// 测试多个更新
	result = reducer(10, 1, 2, 3)
	if result != 16 {
		t.Errorf("expected 16, got %d", result)
	}
}

// TestSumReducer_Float 测试浮点数求和
func TestSumReducer_Float(t *testing.T) {
	reducer := SumReducer[float64]()

	result := reducer(10.5, 2.5, 3.0)
	expected := 16.0

	if result != expected {
		t.Errorf("expected %f, got %f", expected, result)
	}
}

// TestSumReducer_Negative 测试负数
func TestSumReducer_Negative(t *testing.T) {
	reducer := SumReducer[int]()

	result := reducer(10, -5, -3)
	expected := 2

	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

// TestCustomReducer 测试自定义归约器
func TestCustomReducer(t *testing.T) {
	// 创建取最大值的归约器
	maxReducer := CustomReducer(func(current int, updates ...int) int {
		max := current
		for _, v := range updates {
			if v > max {
				max = v
			}
		}
		return max
	})

	result := maxReducer(10, 5, 20, 15, 30, 25)
	expected := 30

	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

// TestCustomReducer_Min 测试取最小值
func TestCustomReducer_Min(t *testing.T) {
	minReducer := CustomReducer(func(current int, updates ...int) int {
		min := current
		for _, v := range updates {
			if v < min {
				min = v
			}
		}
		return min
	})

	result := minReducer(10, 5, 20, 3, 30, 25)
	expected := 3

	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}
}

// TestCustomReducer_Concatenate 测试字符串拼接
func TestCustomReducer_Concatenate(t *testing.T) {
	concatReducer := CustomReducer(func(current string, updates ...string) string {
		result := current
		for _, v := range updates {
			result += v
		}
		return result
	})

	result := concatReducer("Hello", " ", "World", "!")
	expected := "Hello World!"

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

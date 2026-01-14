package state

// Reducer 是状态归约函数。
//
// Reducer 定义了如何将多个状态更新合并为一个最终状态。
// 在并行节点或复杂状态更新场景中，Reducer 决定如何组合多个更新。
//
// 类型参数：
//   - S: 状态类型
//
type Reducer[S any] func(current S, updates ...S) S

// LastValueReducer 是简单的覆盖归约器。
//
// LastValueReducer 总是返回最后一个更新值。
// 这是默认的归约策略。
//
// 示例：
//
//	reducer := LastValueReducer[int]()
//	result := reducer(0, 1, 2, 3)
//	// result == 3
//
func LastValueReducer[S any]() Reducer[S] {
	return func(current S, updates ...S) S {
		if len(updates) == 0 {
			return current
		}
		return updates[len(updates)-1]
	}
}

// MergeReducer 是合并归约器（用于 map 类型）。
//
// MergeReducer 将多个 map 合并为一个。
// 后面的值会覆盖前面的值。
//
// 注意：
//   - 目前仅支持 map[string]any 类型
//   - 后续版本将支持泛型 map
//
// 示例：
//
//	reducer := MergeReducer()
//	m1 := map[string]any{"a": 1, "b": 2}
//	m2 := map[string]any{"b": 3, "c": 4}
//	result := reducer(nil, m1, m2)
//	// result == map[string]any{"a": 1, "b": 3, "c": 4}
//
func MergeReducer() Reducer[map[string]any] {
	return func(current map[string]any, updates ...map[string]any) map[string]any {
		result := make(map[string]any)

		// 复制 current
		if current != nil {
			for k, v := range current {
				result[k] = v
			}
		}

		// 依次合并 updates
		for _, update := range updates {
			if update != nil {
				for k, v := range update {
					result[k] = v
				}
			}
		}

		return result
	}
}

// AppendReducer 是追加归约器（用于切片类型）。
//
// AppendReducer 将多个切片合并为一个。
//
// 注意：
//   - 目前仅支持 []any 类型
//   - 后续版本将支持泛型切片
//
// 示例：
//
//	reducer := AppendReducer()
//	s1 := []any{1, 2}
//	s2 := []any{3, 4}
//	result := reducer(nil, s1, s2)
//	// result == []any{1, 2, 3, 4}
//
func AppendReducer() Reducer[[]any] {
	return func(current []any, updates ...[]any) []any {
		result := make([]any, 0)

		// 复制 current
		if current != nil {
			result = append(result, current...)
		}

		// 依次追加 updates
		for _, update := range updates {
			if update != nil {
				result = append(result, update...)
			}
		}

		return result
	}
}

// SumReducer 是求和归约器（用于数值类型）。
//
// SumReducer 将所有值相加。
//
// 示例：
//
//	reducer := SumReducer[int]()
//	result := reducer(10, 1, 2, 3)
//	// result == 16 (10 + 1 + 2 + 3)
//
func SumReducer[S interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}]() Reducer[S] {
	return func(current S, updates ...S) S {
		result := current
		for _, update := range updates {
			result += update
		}
		return result
	}
}

// CustomReducer 创建自定义归约器。
//
// 参数：
//   - fn: 归约函数
//
// 返回：
//   - Reducer[S]: 归约器
//
// 示例：
//
//	// 创建一个取最大值的归约器
//	maxReducer := CustomReducer(func(current int, updates ...int) int {
//	    max := current
//	    for _, v := range updates {
//	        if v > max {
//	            max = v
//	        }
//	    }
//	    return max
//	})
//
func CustomReducer[S any](fn func(current S, updates ...S) S) Reducer[S] {
	return fn
}

package state

// Channel 表示状态通道。
//
// Channel 用于管理状态的特定字段，支持自定义的更新策略。
// 在 LangGraph 中，Channel 允许对状态的不同部分使用不同的更新逻辑。
//
// 例如：
//   - 某些字段可能需要累加（如计数器）
//   - 某些字段可能需要追加（如消息列表）
//   - 某些字段可能需要覆盖（如状态标志）
//
// Channel 接口定义了如何更新状态的特定部分。
//
// 注意：
//   - 这是高级特性，大多数情况下直接更新状态即可
//   - 将在后续版本中提供更多内置 Channel 类型
//
type Channel interface {
	// GetName 返回通道名称
	GetName() string

	// Update 更新通道值
	//
	// 参数：
	//   - current: 当前值
	//   - update: 更新值
	//
	// 返回：
	//   - any: 新值
	//   - error: 更新错误
	//
	Update(current any, update any) (any, error)
}

// LastValueChannel 是简单的覆盖通道。
//
// LastValueChannel 总是使用最新的值覆盖旧值。
// 这是最简单的通道类型，适用于大多数场景。
//
// 示例：
//
//	channel := NewLastValueChannel("status")
//	newValue, _ := channel.Update("old", "new")
//	// newValue == "new"
//
type LastValueChannel struct {
	name string
}

// NewLastValueChannel 创建覆盖通道。
//
// 参数：
//   - name: 通道名称
//
// 返回：
//   - *LastValueChannel: 通道实例
//
func NewLastValueChannel(name string) *LastValueChannel {
	return &LastValueChannel{name: name}
}

// GetName 实现 Channel 接口。
func (c *LastValueChannel) GetName() string {
	return c.name
}

// Update 实现 Channel 接口。
//
// LastValueChannel 总是返回 update 值，忽略 current。
//
func (c *LastValueChannel) Update(current any, update any) (any, error) {
	return update, nil
}

// AppendChannel 是追加通道。
//
// AppendChannel 将新值追加到切片中。
// 如果 current 不是切片，会创建新切片。
//
// 示例：
//
//	channel := NewAppendChannel("messages")
//	newValue, _ := channel.Update([]string{"a", "b"}, "c")
//	// newValue == []string{"a", "b", "c"}
//
type AppendChannel struct {
	name string
}

// NewAppendChannel 创建追加通道。
//
// 参数：
//   - name: 通道名称
//
// 返回：
//   - *AppendChannel: 通道实例
//
func NewAppendChannel(name string) *AppendChannel {
	return &AppendChannel{name: name}
}

// GetName 实现 Channel 接口。
func (c *AppendChannel) GetName() string {
	return c.name
}

// Update 实现 Channel 接口。
//
// AppendChannel 将 update 追加到 current 切片中。
// 如果 current 为 nil 或不是切片，会创建新切片。
//
// 注意：
//   - 目前仅支持 []interface{} 类型
//   - 后续版本将支持泛型切片
//
func (c *AppendChannel) Update(current any, update any) (any, error) {
	// 如果 current 为 nil，创建新切片
	if current == nil {
		return []any{update}, nil
	}

	// 尝试转换为切片
	currentSlice, ok := current.([]any)
	if !ok {
		// 如果不是切片，创建新切片
		return []any{update}, nil
	}

	// 追加新值
	return append(currentSlice, update), nil
}

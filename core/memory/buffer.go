package memory

import (
	"context"
	"sync"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// BufferMemory 是简单的缓冲记忆实现。
//
// BufferMemory 将所有对话历史保存在内存中。
// 适用于对话轮次不多的场景。
//
// 特点：
//   - 保存完整的对话历史
//   - 无长度限制（需要手动管理）
//   - 快速访问
//   - 线程安全
//
// 示例：
//
//	mem := memory.NewBufferMemory()
//
//	// 保存对话
//	mem.SaveContext(ctx, map[string]any{
//	    "input": "Hello",
//	}, map[string]any{
//	    "output": "Hi!",
//	})
//
//	// 加载历史
//	vars, _ := mem.LoadMemoryVariables(ctx, nil)
//	history := vars["history"].([]types.Message)
//
type BufferMemory struct {
	*BaseMemory
	messages []types.Message
	mu       sync.RWMutex
}

// NewBufferMemory 创建缓冲记忆实例。
//
// 返回：
//   - *BufferMemory: 缓冲记忆实例
//
func NewBufferMemory() *BufferMemory {
	return &BufferMemory{
		BaseMemory: NewBaseMemory(),
		messages:   make([]types.Message, 0),
	}
}

// LoadMemoryVariables 实现 Memory 接口。
func (m *BufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]any)

	if m.returnMessages {
		// 返回消息列表
		result[m.memoryKey] = m.messages
	} else {
		// 返回字符串格式
		result[m.memoryKey] = messagesToString(m.messages)
	}

	return result, nil
}

// SaveContext 实现 Memory 接口。
func (m *BufferMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inputStr, outputStr := m.extractInputOutput(inputs, outputs)

	if inputStr != "" {
		m.messages = append(m.messages, types.NewUserMessage(inputStr))
	}

	if outputStr != "" {
		m.messages = append(m.messages, types.NewAssistantMessage(outputStr))
	}

	return nil
}

// Clear 实现 Memory 接口。
func (m *BufferMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]types.Message, 0)
	return nil
}

// GetMessages 获取所有消息（用于测试和调试）。
//
// 返回：
//   - []types.Message: 消息列表的副本
//
func (m *BufferMemory) GetMessages() []types.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本以避免并发修改
	result := make([]types.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// ConversationBufferWindowMemory 是滑动窗口记忆实现。
//
// ConversationBufferWindowMemory 只保留最近的 K 轮对话。
// 适用于长对话场景，避免上下文过长。
//
// 特点：
//   - 自动限制历史长度
//   - 保持最近的对话
//   - 节省 Token 使用
//   - 线程安全
//
// 示例：
//
//	mem := memory.NewConversationBufferWindowMemory(memory.WindowMemoryConfig{
//	    K: 5, // 只保留最近 5 轮对话（10 条消息）
//	})
//
//	// 使用方式与 BufferMemory 相同
//	mem.SaveContext(ctx, inputs, outputs)
//
type ConversationBufferWindowMemory struct {
	*BaseMemory
	messages []types.Message
	k        int // 保留的对话轮次（每轮包含 input + output）
	mu       sync.RWMutex
}

// WindowMemoryConfig 是滑动窗口记忆的配置。
type WindowMemoryConfig struct {
	// K 是保留的对话轮次数（每轮包含用户输入和 AI 输出）
	// 实际保存的消息数 = K * 2
	K int
}

// NewConversationBufferWindowMemory 创建滑动窗口记忆实例。
//
// 参数：
//   - config: 窗口配置
//
// 返回：
//   - *ConversationBufferWindowMemory: 滑动窗口记忆实例
//
func NewConversationBufferWindowMemory(config WindowMemoryConfig) *ConversationBufferWindowMemory {
	k := config.K
	if k <= 0 {
		k = 5 // 默认保留 5 轮对话
	}

	return &ConversationBufferWindowMemory{
		BaseMemory: NewBaseMemory(),
		messages:   make([]types.Message, 0),
		k:          k,
	}
}

// LoadMemoryVariables 实现 Memory 接口。
func (m *ConversationBufferWindowMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]any)

	if m.returnMessages {
		result[m.memoryKey] = m.messages
	} else {
		result[m.memoryKey] = messagesToString(m.messages)
	}

	return result, nil
}

// SaveContext 实现 Memory 接口。
func (m *ConversationBufferWindowMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inputStr, outputStr := m.extractInputOutput(inputs, outputs)

	if inputStr != "" {
		m.messages = append(m.messages, types.NewUserMessage(inputStr))
	}

	if outputStr != "" {
		m.messages = append(m.messages, types.NewAssistantMessage(outputStr))
	}

	// 保持窗口大小：保留最近的 k*2 条消息
	maxMessages := m.k * 2
	if len(m.messages) > maxMessages {
		m.messages = m.messages[len(m.messages)-maxMessages:]
	}

	return nil
}

// Clear 实现 Memory 接口。
func (m *ConversationBufferWindowMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]types.Message, 0)
	return nil
}

// GetMessages 获取所有消息（用于测试和调试）。
func (m *ConversationBufferWindowMemory) GetMessages() []types.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]types.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// GetK 获取窗口大小。
func (m *ConversationBufferWindowMemory) GetK() int {
	return m.k
}

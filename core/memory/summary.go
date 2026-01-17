package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// ChatModel 是 LLM 的简化接口（用于避免循环依赖）。
type ChatModel interface {
	Invoke(ctx context.Context, messages []types.Message) (types.Message, error)
}

// ConversationSummaryMemory 是摘要记忆实现。
//
// ConversationSummaryMemory 使用 LLM 将长对话历史压缩为摘要。
// 当对话历史超过指定长度时，自动触发摘要生成。
//
// 特点：
//   - 使用 LLM 生成对话摘要
//   - 自动压缩长对话
//   - 保持关键信息
//   - 节省 Token 使用
//
// 注意：
//   - 需要提供 ChatModel 实例
//   - 摘要生成会额外调用 LLM
//   - 适用于长对话场景
//
// 示例：
//
//	mem := memory.NewConversationSummaryMemory(memory.SummaryMemoryConfig{
//	    LLM:        model,
//	    MaxTokens:  2000, // 超过此限制时生成摘要
//	})
//
//	// 使用方式与其他 Memory 相同
//	mem.SaveContext(ctx, inputs, outputs)
//
type ConversationSummaryMemory struct {
	*BaseMemory
	llm          ChatModel
	summary      string
	messages     []types.Message
	maxTokens    int
	summaryPrompt string
	mu           sync.RWMutex
}

// SummaryMemoryConfig 是摘要记忆的配置。
type SummaryMemoryConfig struct {
	// LLM 是用于生成摘要的语言模型
	LLM ChatModel

	// MaxTokens 是触发摘要的最大 Token 数（粗略估计）
	// 默认值：2000
	MaxTokens int

	// SummaryPrompt 是生成摘要的提示词模板（可选）
	// 如果为空，使用默认模板
	SummaryPrompt string
}

// 默认的摘要提示词
const defaultSummaryPrompt = `Progressively summarize the lines of conversation provided, adding onto the previous summary returning a new summary.

EXAMPLE
Current summary:
The human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good.

New lines of conversation:
Human: Why do you think artificial intelligence is a force for good?
AI: Because artificial intelligence will help humans reach their full potential.

New summary:
The human asks what the AI thinks of artificial intelligence. The AI thinks artificial intelligence is a force for good because it will help humans reach their full potential.
END OF EXAMPLE

Current summary:
%s

New lines of conversation:
%s

New summary:`

// NewConversationSummaryMemory 创建摘要记忆实例。
//
// 参数：
//   - config: 摘要记忆配置
//
// 返回：
//   - *ConversationSummaryMemory: 摘要记忆实例
//
func NewConversationSummaryMemory(config SummaryMemoryConfig) *ConversationSummaryMemory {
	maxTokens := config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2000
	}

	summaryPrompt := config.SummaryPrompt
	if summaryPrompt == "" {
		summaryPrompt = defaultSummaryPrompt
	}

	return &ConversationSummaryMemory{
		BaseMemory:    NewBaseMemory(),
		llm:           config.LLM,
		summary:       "",
		messages:      make([]types.Message, 0),
		maxTokens:     maxTokens,
		summaryPrompt: summaryPrompt,
	}
}

// LoadMemoryVariables 实现 Memory 接口。
func (m *ConversationSummaryMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]any)

	if m.returnMessages {
		// 如果有摘要，添加系统消息包含摘要
		if m.summary != "" {
			summaryMsg := types.NewSystemMessage("Previous conversation summary: " + m.summary)
			combined := append([]types.Message{summaryMsg}, m.messages...)
			result[m.memoryKey] = combined
		} else {
			result[m.memoryKey] = m.messages
		}
	} else {
		// 字符串格式
		content := ""
		if m.summary != "" {
			content = "Summary: " + m.summary + "\n\n"
		}
		content += messagesToString(m.messages)
		result[m.memoryKey] = content
	}

	return result, nil
}

// SaveContext 实现 Memory 接口。
func (m *ConversationSummaryMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inputStr, outputStr := m.extractInputOutput(inputs, outputs)

	if inputStr != "" {
		m.messages = append(m.messages, types.NewUserMessage(inputStr))
	}

	if outputStr != "" {
		m.messages = append(m.messages, types.NewAssistantMessage(outputStr))
	}

	// 检查是否需要生成摘要
	if m.shouldSummarize() {
		if err := m.summarize(ctx); err != nil {
			return fmt.Errorf("failed to generate summary: %w", err)
		}
	}

	return nil
}

// Clear 实现 Memory 接口。
func (m *ConversationSummaryMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.summary = ""
	m.messages = make([]types.Message, 0)
	return nil
}

// shouldSummarize 检查是否应该生成摘要。
func (m *ConversationSummaryMemory) shouldSummarize() bool {
	// 粗略估计 Token 数：每个字符约 0.25 个 Token
	estimatedTokens := m.estimateTokens()
	return estimatedTokens > m.maxTokens
}

// estimateTokens 粗略估计当前对话的 Token 数。
func (m *ConversationSummaryMemory) estimateTokens() int {
	totalChars := len(m.summary)
	for _, msg := range m.messages {
		totalChars += len(msg.Content)
	}
	// 粗略估计：4 个字符 = 1 个 Token
	return totalChars / 4
}

// summarize 生成对话摘要。
func (m *ConversationSummaryMemory) summarize(ctx context.Context) error {
	if m.llm == nil {
		return fmt.Errorf("LLM is required for summary generation")
	}

	// 构建新对话内容
	newConversation := messagesToString(m.messages)

	// 构建提示词
	promptText := fmt.Sprintf(m.summaryPrompt, m.summary, newConversation)

	// 调用 LLM 生成摘要
	response, err := m.llm.Invoke(ctx, []types.Message{
		types.NewUserMessage(promptText),
	})
	if err != nil {
		return err
	}

	// 更新摘要并清空消息列表
	m.summary = strings.TrimSpace(response.Content)
	m.messages = make([]types.Message, 0)

	return nil
}

// GetSummary 获取当前摘要（用于测试和调试）。
//
// 返回：
//   - string: 当前摘要
//
func (m *ConversationSummaryMemory) GetSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.summary
}

// GetMessages 获取当前未摘要的消息（用于测试和调试）。
func (m *ConversationSummaryMemory) GetMessages() []types.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]types.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

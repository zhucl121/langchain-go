package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	
	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// EntityMemory 是实体记忆实现。
//
// EntityMemory 不仅记住对话历史，还能：
//   - 自动识别和提取对话中的实体（人名、地名、组织等）
//   - 为每个实体维护专属的上下文信息
//   - 根据当前对话主题检索相关实体
//   - 智能合并和更新实体信息
//
// 这使得 AI 能够：
//   - 记住用户提到的人、地点、事物
//   - 在后续对话中引用这些实体
//   - 提供更个性化的对话体验
//
type EntityMemory struct {
	*BaseMemory
	
	// llm 用于提取实体的语言模型
	llm chat.ChatModel
	
	// entities 实体存储 map[实体名称]实体信息
	entities map[string]*Entity
	
	// conversationHistory 对话历史
	conversationHistory []types.Message
	
	// maxHistoryLength 最大历史长度
	maxHistoryLength int
	
	// entityExtractionPrompt 实体提取提示词
	entityExtractionPrompt string
	
	mu sync.RWMutex
}

// Entity 表示一个实体及其相关信息
type Entity struct {
	// Name 实体名称
	Name string
	
	// Type 实体类型（如: person, organization, location, date, etc.）
	Type string
	
	// Context 实体相关的上下文信息列表
	Context []string
	
	// FirstMentioned 首次提及的对话轮次
	FirstMentioned int
	
	// LastMentioned 最后提及的对话轮次
	LastMentioned int
	
	// MentionCount 提及次数
	MentionCount int
}

// EntityMemoryConfig EntityMemory 配置
type EntityMemoryConfig struct {
	// LLM 用于提取实体的语言模型（必需）
	LLM chat.ChatModel
	
	// MaxHistoryLength 最大对话历史长度（默认: 20）
	MaxHistoryLength int
	
	// EntityExtractionPrompt 自定义实体提取提示词（可选）
	EntityExtractionPrompt string
	
	// ReturnMessages 是否返回消息列表（默认: true）
	ReturnMessages bool
}

// NewEntityMemory 创建实体记忆实例
func NewEntityMemory(config EntityMemoryConfig) *EntityMemory {
	if config.MaxHistoryLength <= 0 {
		config.MaxHistoryLength = 20
	}
	
	prompt := config.EntityExtractionPrompt
	if prompt == "" {
		prompt = getDefaultEntityExtractionPrompt()
	}
	
	baseMemory := NewBaseMemory()
	baseMemory.SetReturnMessages(config.ReturnMessages)
	
	return &EntityMemory{
		BaseMemory:             baseMemory,
		llm:                    config.LLM,
		entities:               make(map[string]*Entity),
		conversationHistory:    make([]types.Message, 0),
		maxHistoryLength:       config.MaxHistoryLength,
		entityExtractionPrompt: prompt,
	}
}

// LoadMemoryVariables 实现 Memory 接口
func (em *EntityMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	result := make(map[string]any)
	
	// 1. 返回基础对话历史
	if em.returnMessages {
		result[em.memoryKey] = em.conversationHistory
	} else {
		result[em.memoryKey] = messagesToString(em.conversationHistory)
	}
	
	// 2. 返回实体信息
	entityInfo := em.formatEntities()
	if entityInfo != "" {
		result["entities"] = entityInfo
	}
	
	// 3. 如果有当前输入，提取相关实体
	if inputs != nil {
		if inputVal, ok := inputs[em.inputKey]; ok {
			if inputStr, ok := inputVal.(string); ok && inputStr != "" {
				relevantEntities := em.getRelevantEntities(inputStr)
				if len(relevantEntities) > 0 {
					result["relevant_entities"] = em.formatSpecificEntities(relevantEntities)
				}
			}
		}
	}
	
	return result, nil
}

// SaveContext 实现 Memory 接口
func (em *EntityMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	inputStr, outputStr := em.extractInputOutput(inputs, outputs)
	
	// 保存对话历史
	turnNumber := len(em.conversationHistory) / 2 + 1
	
	if inputStr != "" {
		em.conversationHistory = append(em.conversationHistory, types.NewUserMessage(inputStr))
	}
	
	if outputStr != "" {
		em.conversationHistory = append(em.conversationHistory, types.NewAssistantMessage(outputStr))
	}
	
	// 限制历史长度
	if len(em.conversationHistory) > em.maxHistoryLength {
		em.conversationHistory = em.conversationHistory[len(em.conversationHistory)-em.maxHistoryLength:]
	}
	
	// 提取实体（异步，不阻塞）
	go em.extractAndUpdateEntities(context.Background(), inputStr, outputStr, turnNumber)
	
	return nil
}

// Clear 实现 Memory 接口
func (em *EntityMemory) Clear(ctx context.Context) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	em.conversationHistory = make([]types.Message, 0)
	em.entities = make(map[string]*Entity)
	
	return nil
}

// extractAndUpdateEntities 提取并更新实体
func (em *EntityMemory) extractAndUpdateEntities(ctx context.Context, input, output string, turnNumber int) {
	if em.llm == nil {
		return
	}
	
	// 合并输入和输出
	text := input
	if output != "" {
		text += " " + output
	}
	
	if text == "" {
		return
	}
	
	// 构建提示词
	prompt := fmt.Sprintf("%s\n\nText: %s\n\nPlease extract entities from the text above.", 
		em.entityExtractionPrompt, text)
	
	messages := []types.Message{
		types.NewSystemMessage("You are an entity extraction assistant."),
		types.NewUserMessage(prompt),
	}
	
	// 调用 LLM
	response, err := em.llm.Invoke(ctx, messages)
	if err != nil {
		return
	}
	
	// 解析实体
	entities := em.parseEntities(response.Content)
	
	// 更新实体存储
	em.mu.Lock()
	defer em.mu.Unlock()
	
	for _, entity := range entities {
		if existing, ok := em.entities[entity.Name]; ok {
			// 更新现有实体
			existing.Context = append(existing.Context, entity.Context...)
			existing.LastMentioned = turnNumber
			existing.MentionCount++
		} else {
			// 添加新实体
			entity.FirstMentioned = turnNumber
			entity.LastMentioned = turnNumber
			entity.MentionCount = 1
			em.entities[entity.Name] = entity
		}
	}
}

// parseEntities 解析 LLM 返回的实体
func (em *EntityMemory) parseEntities(text string) []*Entity {
	entities := make([]*Entity, 0)
	
	// 简单的解析逻辑
	// 期望格式：
	// - EntityName (type): context
	// 例如：
	// - Alice (person): works at TechCorp
	// - TechCorp (organization): a technology company
	
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "-") {
			continue
		}
		
		// 去掉开头的 "-"
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		
		// 解析格式: Name (type): context
		if idx := strings.Index(line, ":"); idx > 0 {
			left := line[:idx]
			context := strings.TrimSpace(line[idx+1:])
			
			// 提取名称和类型
			name := left
			entityType := "unknown"
			
			if typeStart := strings.Index(left, "("); typeStart > 0 {
				if typeEnd := strings.Index(left[typeStart:], ")"); typeEnd > 0 {
					name = strings.TrimSpace(left[:typeStart])
					entityType = strings.TrimSpace(left[typeStart+1 : typeStart+typeEnd])
				}
			}
			
			if name != "" {
				entities = append(entities, &Entity{
					Name:    name,
					Type:    entityType,
					Context: []string{context},
				})
			}
		}
	}
	
	return entities
}

// getRelevantEntities 获取与输入相关的实体
func (em *EntityMemory) getRelevantEntities(input string) []*Entity {
	relevant := make([]*Entity, 0)
	
	inputLower := strings.ToLower(input)
	
	for _, entity := range em.entities {
		// 简单的关键词匹配
		if strings.Contains(inputLower, strings.ToLower(entity.Name)) {
			relevant = append(relevant, entity)
		}
	}
	
	return relevant
}

// formatEntities 格式化所有实体
func (em *EntityMemory) formatEntities() string {
	if len(em.entities) == 0 {
		return ""
	}
	
	var builder strings.Builder
	builder.WriteString("Known Entities:\n")
	
	for _, entity := range em.entities {
		builder.WriteString(fmt.Sprintf("\n- %s (%s):\n", entity.Name, entity.Type))
		for _, ctx := range entity.Context {
			builder.WriteString(fmt.Sprintf("  * %s\n", ctx))
		}
		builder.WriteString(fmt.Sprintf("  Mentioned %d times (first: turn %d, last: turn %d)\n",
			entity.MentionCount, entity.FirstMentioned, entity.LastMentioned))
	}
	
	return builder.String()
}

// formatSpecificEntities 格式化特定实体列表
func (em *EntityMemory) formatSpecificEntities(entities []*Entity) string {
	if len(entities) == 0 {
		return ""
	}
	
	var builder strings.Builder
	builder.WriteString("Relevant Entities:\n")
	
	for _, entity := range entities {
		builder.WriteString(fmt.Sprintf("\n- %s (%s):\n", entity.Name, entity.Type))
		for _, ctx := range entity.Context {
			builder.WriteString(fmt.Sprintf("  * %s\n", ctx))
		}
	}
	
	return builder.String()
}

// GetEntity 获取特定实体
func (em *EntityMemory) GetEntity(name string) (*Entity, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	entity, ok := em.entities[name]
	return entity, ok
}

// GetAllEntities 获取所有实体
func (em *EntityMemory) GetAllEntities() map[string]*Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	// 返回副本
	result := make(map[string]*Entity, len(em.entities))
	for k, v := range em.entities {
		result[k] = v
	}
	
	return result
}

// GetEntityCount 获取实体数量
func (em *EntityMemory) GetEntityCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	return len(em.entities)
}

// getDefaultEntityExtractionPrompt 返回默认的实体提取提示词
func getDefaultEntityExtractionPrompt() string {
	return `Extract entities from the text and format them as follows:
- EntityName (type): brief description or context

Entity types can include:
- person: people's names
- organization: companies, institutions, groups
- location: cities, countries, places
- date: dates, times, time periods
- product: products, services
- event: events, incidents
- concept: important concepts, ideas

Only extract entities that are explicitly mentioned and provide meaningful context for each.`
}

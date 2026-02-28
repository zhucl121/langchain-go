// Package cognitive 提供三层认知记忆系统。
//
// 对标 LangMem SDK，实现完整的认知记忆架构：
//   - 语义记忆（Semantic Memory）：事实和知识（用户偏好、知识三元组）
//   - 情节记忆（Episodic Memory）：经历和经验（对话历史、Few-Shot 示例）
//   - 程序性记忆（Procedural Memory）：行为和技能（System Prompt 优化、工具使用模式）
//
// 使用示例：
//
//	mem := cognitive.NewMemoryManager(cognitive.ManagerConfig{
//	    Storage:  postgresStore,
//	    UserID:   "user-123",
//	    ThreadID: "thread-456",
//	})
//
//	// 自动提取并存储记忆
//	err := mem.AutoConsolidate(ctx, messages)
//
//	// 检索相关记忆
//	recalled, err := mem.Recall(ctx, "用户的编程偏好", cognitive.RecallOptions{K: 5})
//
//	// 使用记忆增强 Agent 上下文
//	augmentedPrompt := recalled.AugmentedContext
//
package cognitive

import (
	"context"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// ─── 核心接口 ────────────────────────────────────────────────

// CognitiveMemory 三层认知记忆统一接口。
//
// 对标 LangMem SDK 的完整记忆管理能力。
type CognitiveMemory interface {
	// ── 语义记忆 ──

	// StoreSemanticMemory 存储一条语义记忆（事实/知识）
	StoreSemanticMemory(ctx context.Context, mem *SemanticMemory) error

	// SearchSemanticMemory 语义检索（向量相似度或关键词）
	SearchSemanticMemory(ctx context.Context, query string, k int) ([]*SemanticMemory, error)

	// ── 情节记忆 ──

	// StoreEpisode 存储一段对话情节
	StoreEpisode(ctx context.Context, episode *Episode) error

	// SearchEpisodes 检索相似情节
	SearchEpisodes(ctx context.Context, query string, k int) ([]*Episode, error)

	// ── 程序性记忆 ──

	// UpdateProceduralMemory 基于反馈更新程序性记忆
	UpdateProceduralMemory(ctx context.Context, feedback *Feedback) error

	// GetProceduralMemory 获取当前程序性记忆（技能 + 偏好）
	GetProceduralMemory(ctx context.Context) (*ProceduralMemory, error)

	// ── 统一检索 ──

	// Recall 跨层记忆检索，返回增强后的上下文
	Recall(ctx context.Context, query string, opts RecallOptions) (*RecalledMemories, error)

	// ── 自动管理 ──

	// AutoConsolidate 从对话消息中自动提取、整合记忆（后台运行）
	AutoConsolidate(ctx context.Context, messages []types.Message) error
}

// ─── 语义记忆类型 ─────────────────────────────────────────────

// SemanticMemory 语义记忆（事实与知识）。
//
// 存储用户偏好、知识三元组等结构化事实。
type SemanticMemory struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	ThreadID    string            `json:"thread_id,omitempty"`

	// Content 记忆内容（自然语言）
	Content     string            `json:"content"`

	// Triplets 知识三元组（主谓宾结构）
	Triplets    []KnowledgeTriplet `json:"triplets,omitempty"`

	// Category 分类（"preference", "fact", "knowledge" 等）
	Category    string            `json:"category,omitempty"`

	// Tags 标签
	Tags        []string          `json:"tags,omitempty"`

	// Confidence 置信度 [0,1]
	Confidence  float64           `json:"confidence"`

	// Source 来源（对话 ID、文档 ID 等）
	Source      string            `json:"source,omitempty"`

	// Embedding 向量嵌入（用于语义检索）
	Embedding   []float32         `json:"embedding,omitempty"`

	// Extra 额外元数据
	Extra       map[string]any    `json:"extra,omitempty"`

	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// KnowledgeTriplet 知识三元组（主谓宾结构）。
//
// 将事实结构化表示为 (Subject, Predicate, Object) 三元组，
// 便于知识图谱构建和精确检索。
//
// 示例：("用户", "偏好", "Go 语言") → 用户偏好 Go 语言
type KnowledgeTriplet struct {
	Subject    string  `json:"subject"`
	Predicate  string  `json:"predicate"`
	Object     string  `json:"object"`
	Context    string  `json:"context,omitempty"`
	Confidence float64 `json:"confidence"`
}

// ─── 情节记忆类型 ─────────────────────────────────────────────

// Episode 情节记忆（经历与经验）。
//
// 存储完整的对话片段、Few-Shot 示例、工具使用记录等。
type Episode struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	ThreadID    string            `json:"thread_id"`

	// Messages 对话消息列表
	Messages    []types.Message   `json:"messages"`

	// Summary 情节摘要（自然语言）
	Summary     string            `json:"summary,omitempty"`

	// Outcome 结果描述（成功/失败/部分完成）
	Outcome     string            `json:"outcome,omitempty"`

	// Tags 标签（任务类型、领域等）
	Tags        []string          `json:"tags,omitempty"`

	// Embedding 摘要的向量嵌入
	Embedding   []float32         `json:"embedding,omitempty"`

	// Quality 质量评分 [0,1]（用于 Few-Shot 选择）
	Quality     float64           `json:"quality"`

	// IsFewShot 是否作为 Few-Shot 示例
	IsFewShot   bool              `json:"is_few_shot,omitempty"`

	// Extra 额外元数据
	Extra       map[string]any    `json:"extra,omitempty"`

	CreatedAt   time.Time         `json:"created_at"`
}

// ─── 程序性记忆类型 ───────────────────────────────────────────

// ProceduralMemory 程序性记忆（行为与技能）。
//
// 存储 Agent 的行为模式、System Prompt 优化结果、工具使用偏好等。
type ProceduralMemory struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`

	// SystemPromptAdditions 基于反馈生成的 System Prompt 补充
	SystemPromptAdditions []string `json:"system_prompt_additions,omitempty"`

	// ToolPreferences 工具使用偏好（toolName -> preference score）
	ToolPreferences map[string]float64 `json:"tool_preferences,omitempty"`

	// BehaviorRules 行为规则列表（从失败中学习的规则）
	BehaviorRules []BehaviorRule `json:"behavior_rules,omitempty"`

	// Personality 人格特征（影响回复风格）
	Personality string `json:"personality,omitempty"`

	UpdatedAt   time.Time         `json:"updated_at"`
}

// BehaviorRule 行为规则（从历史中学习的策略）。
type BehaviorRule struct {
	ID          string  `json:"id"`
	Rule        string  `json:"rule"`   // 规则描述
	Reason      string  `json:"reason"` // 学习来源
	Confidence  float64 `json:"confidence"`
	UsageCount  int     `json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// Feedback 反馈信息（用于更新程序性记忆）。
type Feedback struct {
	// ThreadID 反馈对应的会话 ID
	ThreadID string

	// MessageID 反馈对应的消息 ID
	MessageID string

	// Score 评分 [-1, 1]（-1=很差, 0=中性, 1=很好）
	Score float64

	// Comment 文字反馈
	Comment string

	// FailureReason 失败原因（用于提取行为规则）
	FailureReason string

	// Metadata 额外信息
	Metadata map[string]any
}

// ─── 检索类型 ─────────────────────────────────────────────────

// RecallOptions 记忆检索选项。
type RecallOptions struct {
	// K 每层记忆检索的最大条数
	K int

	// IncludeSemantic 是否检索语义记忆
	IncludeSemantic bool

	// IncludeEpisodic 是否检索情节记忆
	IncludeEpisodic bool

	// IncludeProcedural 是否检索程序性记忆
	IncludeProcedural bool

	// MinConfidence 最低置信度过滤
	MinConfidence float64

	// MaxAge 最大时间范围（0 表示不限）
	MaxAge time.Duration
}

// DefaultRecallOptions 返回默认检索选项。
func DefaultRecallOptions() RecallOptions {
	return RecallOptions{
		K:                 5,
		IncludeSemantic:   true,
		IncludeEpisodic:   true,
		IncludeProcedural: true,
		MinConfidence:     0.3,
	}
}

// RecalledMemories 检索结果，包含各层记忆和增强后的上下文。
type RecalledMemories struct {
	// SemanticFacts 检索到的语义记忆
	SemanticFacts []*SemanticMemory

	// RecentEpisodes 检索到的情节记忆
	RecentEpisodes []*Episode

	// ProceduralHints 程序性记忆中的行为提示
	ProceduralHints []string

	// AugmentedContext 整合所有记忆后生成的上下文字符串
	// 可直接注入 System Prompt 或用作 RAG 上下文
	AugmentedContext string

	// TotalRecalled 总检索条数
	TotalRecalled int
}

// ─── 存储接口 ─────────────────────────────────────────────────

// SemanticStorage 语义记忆存储接口。
type SemanticStorage interface {
	Save(ctx context.Context, mem *SemanticMemory) error
	Search(ctx context.Context, query string, k int, userID string) ([]*SemanticMemory, error)
	GetByID(ctx context.Context, id string) (*SemanticMemory, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string, limit int) ([]*SemanticMemory, error)
}

// EpisodicStorage 情节记忆存储接口。
type EpisodicStorage interface {
	Save(ctx context.Context, episode *Episode) error
	Search(ctx context.Context, query string, k int, userID string) ([]*Episode, error)
	GetByID(ctx context.Context, id string) (*Episode, error)
	Delete(ctx context.Context, id string) error
	ListByThread(ctx context.Context, threadID string) ([]*Episode, error)
}

// ProceduralStorage 程序性记忆存储接口。
type ProceduralStorage interface {
	Save(ctx context.Context, mem *ProceduralMemory) error
	Get(ctx context.Context, userID string) (*ProceduralMemory, error)
}

// ─── 提取器接口 ───────────────────────────────────────────────

// MemoryExtractor 从对话中提取记忆的接口。
//
// 可实现为：
//   - LLM 驱动的提取器（调用 LLM 分析对话）
//   - 规则驱动的提取器（正则 + 模板）
//   - 混合提取器
type MemoryExtractor interface {
	// ExtractSemantic 从对话中提取语义记忆
	ExtractSemantic(ctx context.Context, messages []types.Message) ([]*SemanticMemory, error)

	// ExtractEpisode 将对话整合为情节摘要
	ExtractEpisode(ctx context.Context, messages []types.Message, threadID string) (*Episode, error)

	// ExtractBehaviorRules 从失败的对话中提取行为规则
	ExtractBehaviorRules(ctx context.Context, messages []types.Message, feedback *Feedback) ([]BehaviorRule, error)
}

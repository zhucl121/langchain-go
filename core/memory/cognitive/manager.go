package cognitive

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/types"
)

// ManagerConfig 记忆管理器配置。
type ManagerConfig struct {
	// UserID 用户 ID（记忆的归属）
	UserID string

	// ThreadID 当前会话 ID（情节记忆的归属）
	ThreadID string

	// SemanticStorage 语义记忆存储（nil 时使用内存存储）
	SemanticStorage SemanticStorage

	// EpisodicStorage 情节记忆存储（nil 时使用内存存储）
	EpisodicStorage EpisodicStorage

	// ProceduralStorage 程序性记忆存储（nil 时使用内存存储）
	ProceduralStorage ProceduralStorage

	// Extractor 记忆提取器（nil 时使用规则提取器）
	Extractor MemoryExtractor

	// AutoConsolidateAsync 是否异步整合记忆（不阻塞主流程）
	AutoConsolidateAsync bool
}

// MemoryManager 三层认知记忆管理器。
//
// 实现 CognitiveMemory 接口，统一管理语义记忆、情节记忆和程序性记忆。
//
// 示例：
//
//	mgr := cognitive.NewMemoryManager(cognitive.ManagerConfig{
//	    UserID:   "user-123",
//	    ThreadID: "thread-456",
//	})
//
//	// 存储语义记忆
//	mgr.StoreSemanticMemory(ctx, &cognitive.SemanticMemory{
//	    Content:    "用户偏好使用 Go 语言",
//	    Category:   "preference",
//	    Confidence: 0.9,
//	})
//
//	// 检索增强上下文
//	recalled, _ := mgr.Recall(ctx, "编程语言偏好", cognitive.DefaultRecallOptions())
//	fmt.Println(recalled.AugmentedContext)
type MemoryManager struct {
	config    ManagerConfig
	semantic  SemanticStorage
	episodic  EpisodicStorage
	procedural ProceduralStorage
	extractor MemoryExtractor
	mu        sync.RWMutex
}

// NewMemoryManager 创建三层认知记忆管理器。
func NewMemoryManager(config ManagerConfig) *MemoryManager {
	semantic := config.SemanticStorage
	if semantic == nil {
		semantic = NewInMemorySemanticStorage()
	}

	episodic := config.EpisodicStorage
	if episodic == nil {
		episodic = NewInMemoryEpisodicStorage()
	}

	procedural := config.ProceduralStorage
	if procedural == nil {
		procedural = NewInMemoryProceduralStorage()
	}

	extractor := config.Extractor
	if extractor == nil {
		extractor = NewRuleBasedExtractor()
	}

	return &MemoryManager{
		config:     config,
		semantic:   semantic,
		episodic:   episodic,
		procedural: procedural,
		extractor:  extractor,
	}
}

// ── 语义记忆操作 ───────────────────────────────────────────────

// StoreSemanticMemory 存储语义记忆。
func (m *MemoryManager) StoreSemanticMemory(ctx context.Context, mem *SemanticMemory) error {
	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	if mem.UserID == "" {
		mem.UserID = m.config.UserID
	}
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()

	return m.semantic.Save(ctx, mem)
}

// SearchSemanticMemory 检索语义记忆。
func (m *MemoryManager) SearchSemanticMemory(ctx context.Context, query string, k int) ([]*SemanticMemory, error) {
	return m.semantic.Search(ctx, query, k, m.config.UserID)
}

// ── 情节记忆操作 ───────────────────────────────────────────────

// StoreEpisode 存储情节记忆。
func (m *MemoryManager) StoreEpisode(ctx context.Context, episode *Episode) error {
	if episode.ID == "" {
		episode.ID = uuid.New().String()
	}
	if episode.UserID == "" {
		episode.UserID = m.config.UserID
	}
	if episode.ThreadID == "" {
		episode.ThreadID = m.config.ThreadID
	}
	episode.CreatedAt = time.Now()

	return m.episodic.Save(ctx, episode)
}

// SearchEpisodes 检索情节记忆。
func (m *MemoryManager) SearchEpisodes(ctx context.Context, query string, k int) ([]*Episode, error) {
	return m.episodic.Search(ctx, query, k, m.config.UserID)
}

// ── 程序性记忆操作 ─────────────────────────────────────────────

// UpdateProceduralMemory 基于反馈更新程序性记忆。
func (m *MemoryManager) UpdateProceduralMemory(ctx context.Context, feedback *Feedback) error {
	mem, err := m.procedural.Get(ctx, m.config.UserID)
	if err != nil {
		return fmt.Errorf("get procedural memory: %w", err)
	}
	if mem == nil {
		mem = &ProceduralMemory{
			ID:              uuid.New().String(),
			UserID:          m.config.UserID,
			ToolPreferences: make(map[string]float64),
		}
	}

	// 提取行为规则
	if feedback.Score < -0.3 && feedback.FailureReason != "" {
		rules, err := m.extractor.ExtractBehaviorRules(ctx, nil, feedback)
		if err == nil && len(rules) > 0 {
			mem.BehaviorRules = append(mem.BehaviorRules, rules...)
		}
	}

	// 正向反馈：加强现有规则
	if feedback.Score > 0.5 {
		for i := range mem.BehaviorRules {
			mem.BehaviorRules[i].UsageCount++
			if mem.BehaviorRules[i].Confidence < 1.0 {
				mem.BehaviorRules[i].Confidence = min(1.0, mem.BehaviorRules[i].Confidence+0.05)
			}
		}
	}

	mem.UpdatedAt = time.Now()
	return m.procedural.Save(ctx, mem)
}

// GetProceduralMemory 获取程序性记忆。
func (m *MemoryManager) GetProceduralMemory(ctx context.Context) (*ProceduralMemory, error) {
	return m.procedural.Get(ctx, m.config.UserID)
}

// ── 统一检索 ───────────────────────────────────────────────────

// Recall 跨层记忆检索，返回增强上下文。
func (m *MemoryManager) Recall(ctx context.Context, query string, opts RecallOptions) (*RecalledMemories, error) {
	if opts.K <= 0 {
		opts.K = 5
	}

	result := &RecalledMemories{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []string

	// 并行检索三层记忆
	if opts.IncludeSemantic {
		wg.Add(1)
		go func() {
			defer wg.Done()
			facts, err := m.SearchSemanticMemory(ctx, query, opts.K)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("semantic: %v", err))
			} else {
				// 过滤低置信度
				for _, f := range facts {
					if f.Confidence >= opts.MinConfidence {
						result.SemanticFacts = append(result.SemanticFacts, f)
					}
				}
			}
		}()
	}

	if opts.IncludeEpisodic {
		wg.Add(1)
		go func() {
			defer wg.Done()
			episodes, err := m.SearchEpisodes(ctx, query, opts.K)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("episodic: %v", err))
			} else {
				for _, e := range episodes {
					if opts.MaxAge == 0 || time.Since(e.CreatedAt) <= opts.MaxAge {
						result.RecentEpisodes = append(result.RecentEpisodes, e)
					}
				}
			}
		}()
	}

	if opts.IncludeProcedural {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pm, err := m.GetProceduralMemory(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("procedural: %v", err))
			} else if pm != nil {
				for _, rule := range pm.BehaviorRules {
					if rule.Confidence >= opts.MinConfidence {
						result.ProceduralHints = append(result.ProceduralHints, rule.Rule)
					}
				}
				result.ProceduralHints = append(result.ProceduralHints, pm.SystemPromptAdditions...)
			}
		}()
	}

	wg.Wait()

	result.TotalRecalled = len(result.SemanticFacts) + len(result.RecentEpisodes) + len(result.ProceduralHints)

	// 生成增强上下文
	result.AugmentedContext = m.buildAugmentedContext(result)

	return result, nil
}

// buildAugmentedContext 将检索到的记忆整合为上下文字符串。
func (m *MemoryManager) buildAugmentedContext(r *RecalledMemories) string {
	var sb strings.Builder

	if len(r.SemanticFacts) > 0 {
		sb.WriteString("## 关于用户的已知信息\n")
		for _, f := range r.SemanticFacts {
			sb.WriteString("- ")
			sb.WriteString(f.Content)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(r.RecentEpisodes) > 0 {
		sb.WriteString("## 相关历史对话摘要\n")
		for _, e := range r.RecentEpisodes {
			if e.Summary != "" {
				sb.WriteString("- ")
				sb.WriteString(e.Summary)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(r.ProceduralHints) > 0 {
		sb.WriteString("## 行为指引\n")
		for _, hint := range r.ProceduralHints {
			sb.WriteString("- ")
			sb.WriteString(hint)
			sb.WriteString("\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

// ── 自动整合 ───────────────────────────────────────────────────

// AutoConsolidate 从对话中自动提取记忆并存储。
func (m *MemoryManager) AutoConsolidate(ctx context.Context, messages []types.Message) error {
	if m.config.AutoConsolidateAsync {
		go func() {
			_ = m.consolidate(context.Background(), messages)
		}()
		return nil
	}
	return m.consolidate(ctx, messages)
}

func (m *MemoryManager) consolidate(ctx context.Context, messages []types.Message) error {
	var errs []string

	// 1. 提取语义记忆
	semanticMems, err := m.extractor.ExtractSemantic(ctx, messages)
	if err != nil {
		errs = append(errs, fmt.Sprintf("extract semantic: %v", err))
	} else {
		for _, mem := range semanticMems {
			if err := m.StoreSemanticMemory(ctx, mem); err != nil {
				errs = append(errs, fmt.Sprintf("store semantic: %v", err))
			}
		}
	}

	// 2. 提取情节记忆
	episode, err := m.extractor.ExtractEpisode(ctx, messages, m.config.ThreadID)
	if err != nil {
		errs = append(errs, fmt.Sprintf("extract episode: %v", err))
	} else if episode != nil {
		if err := m.StoreEpisode(ctx, episode); err != nil {
			errs = append(errs, fmt.Sprintf("store episode: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("consolidate errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ─── 内存存储实现 ─────────────────────────────────────────────

// InMemorySemanticStorage 内存语义记忆存储。
type InMemorySemanticStorage struct {
	mu      sync.RWMutex
	entries map[string]*SemanticMemory
}

// NewInMemorySemanticStorage 创建内存语义记忆存储。
func NewInMemorySemanticStorage() *InMemorySemanticStorage {
	return &InMemorySemanticStorage{
		entries: make(map[string]*SemanticMemory),
	}
}

func (s *InMemorySemanticStorage) Save(ctx context.Context, mem *SemanticMemory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[mem.ID] = mem
	return nil
}

func (s *InMemorySemanticStorage) Search(ctx context.Context, query string, k int, userID string) ([]*SemanticMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 将查询拆分为词，只要命中任意词即返回
	queryWords := strings.Fields(strings.ToLower(query))
	var results []*SemanticMemory

	for _, mem := range s.entries {
		if userID != "" && mem.UserID != userID {
			continue
		}
		contentLower := strings.ToLower(mem.Content)
		matched := false
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				matched = true
				break
			}
		}
		if matched {
			results = append(results, mem)
		}
		if len(results) >= k {
			break
		}
	}
	return results, nil
}

func (s *InMemorySemanticStorage) GetByID(ctx context.Context, id string) (*SemanticMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mem, ok := s.entries[id]
	if !ok {
		return nil, nil
	}
	return mem, nil
}

func (s *InMemorySemanticStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
	return nil
}

func (s *InMemorySemanticStorage) List(ctx context.Context, userID string, limit int) ([]*SemanticMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*SemanticMemory
	for _, mem := range s.entries {
		if userID != "" && mem.UserID != userID {
			continue
		}
		results = append(results, mem)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results, nil
}

// InMemoryEpisodicStorage 内存情节记忆存储。
type InMemoryEpisodicStorage struct {
	mu      sync.RWMutex
	entries map[string]*Episode
}

// NewInMemoryEpisodicStorage 创建内存情节记忆存储。
func NewInMemoryEpisodicStorage() *InMemoryEpisodicStorage {
	return &InMemoryEpisodicStorage{
		entries: make(map[string]*Episode),
	}
}

func (s *InMemoryEpisodicStorage) Save(ctx context.Context, episode *Episode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[episode.ID] = episode
	return nil
}

func (s *InMemoryEpisodicStorage) Search(ctx context.Context, query string, k int, userID string) ([]*Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryWords := strings.Fields(strings.ToLower(query))
	var results []*Episode

	for _, ep := range s.entries {
		if userID != "" && ep.UserID != userID {
			continue
		}
		if ep.Summary == "" {
			continue
		}
		summaryLower := strings.ToLower(ep.Summary)
		for _, word := range queryWords {
			if strings.Contains(summaryLower, word) {
				results = append(results, ep)
				break
			}
		}
		if len(results) >= k {
			break
		}
	}
	return results, nil
}

func (s *InMemoryEpisodicStorage) GetByID(ctx context.Context, id string) (*Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ep, ok := s.entries[id]
	if !ok {
		return nil, nil
	}
	return ep, nil
}

func (s *InMemoryEpisodicStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
	return nil
}

func (s *InMemoryEpisodicStorage) ListByThread(ctx context.Context, threadID string) ([]*Episode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Episode
	for _, ep := range s.entries {
		if ep.ThreadID == threadID {
			results = append(results, ep)
		}
	}
	return results, nil
}

// InMemoryProceduralStorage 内存程序性记忆存储。
type InMemoryProceduralStorage struct {
	mu   sync.RWMutex
	data map[string]*ProceduralMemory
}

// NewInMemoryProceduralStorage 创建内存程序性记忆存储。
func NewInMemoryProceduralStorage() *InMemoryProceduralStorage {
	return &InMemoryProceduralStorage{
		data: make(map[string]*ProceduralMemory),
	}
}

func (s *InMemoryProceduralStorage) Save(ctx context.Context, mem *ProceduralMemory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[mem.UserID] = mem
	return nil
}

func (s *InMemoryProceduralStorage) Get(ctx context.Context, userID string) (*ProceduralMemory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[userID], nil
}

// ─── 规则提取器 ───────────────────────────────────────────────

// RuleBasedExtractor 基于规则的记忆提取器。
//
// 使用关键词和模式匹配从对话中提取记忆。
// 不依赖 LLM，适合低成本场景。
type RuleBasedExtractor struct{}

// NewRuleBasedExtractor 创建规则提取器。
func NewRuleBasedExtractor() *RuleBasedExtractor {
	return &RuleBasedExtractor{}
}

func (e *RuleBasedExtractor) ExtractSemantic(ctx context.Context, messages []types.Message) ([]*SemanticMemory, error) {
	var mems []*SemanticMemory

	// 简单规则：从用户消息中提取偏好声明
	preferenceKeywords := []string{"我喜欢", "我偏好", "我习惯", "我不喜欢", "请记住", "记住我"}

	for _, msg := range messages {
		if msg.Role != types.RoleUser {
			continue
		}
		for _, kw := range preferenceKeywords {
			if strings.Contains(msg.Content, kw) {
				mems = append(mems, &SemanticMemory{
					Content:    msg.Content,
					Category:   "preference",
					Confidence: 0.7,
					Source:     "conversation",
				})
				break
			}
		}
	}
	return mems, nil
}

func (e *RuleBasedExtractor) ExtractEpisode(ctx context.Context, messages []types.Message, threadID string) (*Episode, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	// 简单摘要：取第一条用户消息作为摘要
	var summary string
	for _, msg := range messages {
		if msg.Role == types.RoleUser {
			summary = msg.Content
			if len(summary) > 100 {
				summary = summary[:97] + "..."
			}
			break
		}
	}

	return &Episode{
		ThreadID: threadID,
		Messages: messages,
		Summary:  summary,
		Quality:  0.5,
	}, nil
}

func (e *RuleBasedExtractor) ExtractBehaviorRules(ctx context.Context, messages []types.Message, feedback *Feedback) ([]BehaviorRule, error) {
	if feedback == nil || feedback.FailureReason == "" {
		return nil, nil
	}

	rule := BehaviorRule{
		ID:         uuid.New().String(),
		Rule:       fmt.Sprintf("避免：%s", feedback.FailureReason),
		Reason:     "用户反馈",
		Confidence: 0.6,
		CreatedAt:  time.Now(),
	}
	return []BehaviorRule{rule}, nil
}

// min 返回两个 float64 中的较小值。
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

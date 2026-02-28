package middleware

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// ModelHook 模型调用钩子接口。
//
// ModelHook 对标 LangGraph 1.0 Pre/Post Model Hooks，允许在模型调用
// 前后注入自定义逻辑，用于：
//   - 控制 context 膨胀（历史摘要）
//   - 护栏（输入/输出过滤）
//   - HITL（Human-in-the-Loop 检查）
//   - PII 自动脱敏
//   - 速率限制
//
// 示例：
//
//	agent := agents.CreateAgent(agents.AgentConfig{
//	    Model: model,
//	    Hooks: []middleware.ModelHook{
//	        middleware.NewSummaryHook(4000),
//	        middleware.NewPIIRedactHook(middleware.DefaultPIIPatterns()),
//	    },
//	})
type ModelHook interface {
	// Name 返回 Hook 名称（用于日志和调试）
	Name() string

	// BeforeModel 在模型调用前执行。
	//
	// 可修改消息列表（如截断历史、注入系统提示）。
	// 返回修改后的消息列表。
	BeforeModel(ctx context.Context, messages []types.Message) ([]types.Message, error)

	// AfterModel 在模型调用后执行。
	//
	// 可检查或修改模型响应（如内容过滤、格式校正）。
	// 返回修改后的响应消息。
	AfterModel(ctx context.Context, response *types.Message) (*types.Message, error)
}

// HookChain 钩子链，按顺序执行多个 Hook。
type HookChain struct {
	hooks []ModelHook
}

// NewHookChain 创建钩子链。
func NewHookChain(hooks ...ModelHook) *HookChain {
	return &HookChain{hooks: hooks}
}

// Add 追加一个 Hook。
func (c *HookChain) Add(hook ModelHook) *HookChain {
	c.hooks = append(c.hooks, hook)
	return c
}

// BeforeModel 按顺序执行所有 Pre Hook。
func (c *HookChain) BeforeModel(ctx context.Context, messages []types.Message) ([]types.Message, error) {
	cur := messages
	for _, h := range c.hooks {
		var err error
		cur, err = h.BeforeModel(ctx, cur)
		if err != nil {
			return nil, fmt.Errorf("hook %s BeforeModel: %w", h.Name(), err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	return cur, nil
}

// AfterModel 按顺序执行所有 Post Hook。
func (c *HookChain) AfterModel(ctx context.Context, response *types.Message) (*types.Message, error) {
	cur := response
	for _, h := range c.hooks {
		var err error
		cur, err = h.AfterModel(ctx, cur)
		if err != nil {
			return nil, fmt.Errorf("hook %s AfterModel: %w", h.Name(), err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	return cur, nil
}

// ─── 1. SummaryHook：历史摘要 Hook ──────────────────────────────

// SummaryFunc 摘要函数类型。
// 接收原始消息列表，返回摘要后的消息列表。
type SummaryFunc func(ctx context.Context, messages []types.Message) ([]types.Message, error)

// SummaryHook 历史摘要钩子。
//
// 当消息历史超过 maxTokenEstimate 个 token 时，
// 调用 summaryFn 将旧消息压缩为摘要，防止 context 膨胀。
//
// 对标 LangGraph Pre-hook: summarizing message history。
type SummaryHook struct {
	maxTokenEstimate int
	summaryFn        SummaryFunc
	keepLast         int // 保留最近 N 条消息不摘要（保持连贯性）
}

// NewSummaryHook 创建历史摘要 Hook。
//
// 参数：
//   - maxTokenEstimate: 估算 token 上限（超过则触发摘要）
//   - summaryFn: 自定义摘要函数（nil 时使用默认截断策略）
//   - keepLast: 保留最近 N 条消息不摘要
func NewSummaryHook(maxTokenEstimate int, summaryFn SummaryFunc, keepLast int) *SummaryHook {
	if keepLast <= 0 {
		keepLast = 6
	}
	return &SummaryHook{
		maxTokenEstimate: maxTokenEstimate,
		summaryFn:        summaryFn,
		keepLast:         keepLast,
	}
}

func (h *SummaryHook) Name() string { return "summary-hook" }

func (h *SummaryHook) BeforeModel(ctx context.Context, messages []types.Message) ([]types.Message, error) {
	estimated := estimateTokens(messages)
	if estimated <= h.maxTokenEstimate {
		return messages, nil
	}

	if h.summaryFn != nil {
		return h.summaryFn(ctx, messages)
	}

	// 默认策略：保留系统消息 + 最近 keepLast 条
	return truncateMessages(messages, h.keepLast), nil
}

func (h *SummaryHook) AfterModel(_ context.Context, response *types.Message) (*types.Message, error) {
	return response, nil
}

// estimateTokens 粗略估算 token 数（字符数 / 4）。
func estimateTokens(messages []types.Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) / 4
	}
	return total
}

// truncateMessages 保留系统消息 + 最近 keepLast 条。
func truncateMessages(messages []types.Message, keepLast int) []types.Message {
	var systemMsgs []types.Message
	var otherMsgs []types.Message

	for _, m := range messages {
		if m.Role == types.RoleSystem {
			systemMsgs = append(systemMsgs, m)
		} else {
			otherMsgs = append(otherMsgs, m)
		}
	}

	start := 0
	if len(otherMsgs) > keepLast {
		start = len(otherMsgs) - keepLast
	}

	result := make([]types.Message, 0, len(systemMsgs)+len(otherMsgs[start:]))
	result = append(result, systemMsgs...)
	result = append(result, otherMsgs[start:]...)
	return result
}

// ─── 2. PIIRedactHook：PII 脱敏 Hook ────────────────────────────

// PIIPattern PII 脱敏规则。
type PIIPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
}

// DefaultPIIPatterns 返回默认的 PII 脱敏规则集（邮箱、手机、身份证、银行卡）。
func DefaultPIIPatterns() []PIIPattern {
	return []PIIPattern{
		{
			Name:        "email",
			Pattern:     regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
			Replacement: "[EMAIL]",
		},
		{
			Name:        "cn-phone",
			Pattern:     regexp.MustCompile(`(?:(?:\+|00)86)?1[3-9]\d{9}`),
			Replacement: "[PHONE]",
		},
		{
			Name:        "cn-idcard",
			Pattern:     regexp.MustCompile(`[1-9]\d{5}(?:19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}[\dXx]`),
			Replacement: "[ID-CARD]",
		},
		{
			Name:        "credit-card",
			Pattern:     regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`),
			Replacement: "[CREDIT-CARD]",
		},
	}
}

// PIIRedactHook PII 自动脱敏钩子。
//
// 在消息发送给模型前，自动将敏感信息（手机、邮箱、身份证等）替换为占位符。
// 可选地在模型响应后恢复或继续脱敏。
//
// 对标 LangGraph Post-hook: guardrails and PII redaction。
type PIIRedactHook struct {
	patterns     []PIIPattern
	redactInput  bool
	redactOutput bool
}

// NewPIIRedactHook 创建 PII 脱敏 Hook。
//
// 参数：
//   - patterns: PII 匹配规则列表（nil 时使用 DefaultPIIPatterns）
//   - redactInput: 是否脱敏输入消息
//   - redactOutput: 是否脱敏输出响应
func NewPIIRedactHook(patterns []PIIPattern, redactInput, redactOutput bool) *PIIRedactHook {
	if patterns == nil {
		patterns = DefaultPIIPatterns()
	}
	return &PIIRedactHook{
		patterns:     patterns,
		redactInput:  redactInput,
		redactOutput: redactOutput,
	}
}

func (h *PIIRedactHook) Name() string { return "pii-redact-hook" }

func (h *PIIRedactHook) BeforeModel(_ context.Context, messages []types.Message) ([]types.Message, error) {
	if !h.redactInput {
		return messages, nil
	}
	result := make([]types.Message, len(messages))
	for i, m := range messages {
		m.Content = h.redact(m.Content)
		result[i] = m
	}
	return result, nil
}

func (h *PIIRedactHook) AfterModel(_ context.Context, response *types.Message) (*types.Message, error) {
	if !h.redactOutput || response == nil {
		return response, nil
	}
	redacted := *response
	redacted.Content = h.redact(redacted.Content)
	return &redacted, nil
}

func (h *PIIRedactHook) redact(text string) string {
	for _, p := range h.patterns {
		text = p.Pattern.ReplaceAllString(text, p.Replacement)
	}
	return text
}

// ─── 3. GuardrailHook：护栏 Hook ────────────────────────────────

// GuardrailAction 护栏触发时的动作。
type GuardrailAction string

const (
	// GuardrailBlock 阻止请求（返回错误）
	GuardrailBlock GuardrailAction = "block"
	// GuardrailWarn 记录警告但继续
	GuardrailWarn GuardrailAction = "warn"
	// GuardrailReplace 替换为安全内容
	GuardrailReplace GuardrailAction = "replace"
)

// GuardrailRule 护栏规则。
type GuardrailRule struct {
	Name           string
	Description    string
	CheckInput     func(messages []types.Message) (bool, string)  // 返回 (violated, reason)
	CheckOutput    func(response *types.Message) (bool, string)
	Action         GuardrailAction
	Replacement    string // 仅 GuardrailReplace 时使用
}

// GuardrailHook 输入/输出护栏钩子。
//
// 在模型调用前后检查内容合规性，阻止或替换不安全内容。
//
// 对标 LangGraph Post-hook: guardrails。
type GuardrailHook struct {
	rules  []GuardrailRule
	logger func(ruleName, reason string)
}

// NewGuardrailHook 创建护栏 Hook。
func NewGuardrailHook(rules []GuardrailRule) *GuardrailHook {
	return &GuardrailHook{rules: rules}
}

// WithLogger 设置日志函数。
func (h *GuardrailHook) WithLogger(logger func(ruleName, reason string)) *GuardrailHook {
	h.logger = logger
	return h
}

func (h *GuardrailHook) Name() string { return "guardrail-hook" }

func (h *GuardrailHook) BeforeModel(_ context.Context, messages []types.Message) ([]types.Message, error) {
	for _, rule := range h.rules {
		if rule.CheckInput == nil {
			continue
		}
		violated, reason := rule.CheckInput(messages)
		if !violated {
			continue
		}

		if h.logger != nil {
			h.logger(rule.Name, reason)
		}

		switch rule.Action {
		case GuardrailBlock:
			return nil, fmt.Errorf("guardrail %s blocked input: %s", rule.Name, reason)
		case GuardrailWarn:
			// 继续
		case GuardrailReplace:
			return []types.Message{{
				Role:    types.RoleUser,
				Content: rule.Replacement,
			}}, nil
		}
	}
	return messages, nil
}

func (h *GuardrailHook) AfterModel(_ context.Context, response *types.Message) (*types.Message, error) {
	if response == nil {
		return nil, nil
	}
	for _, rule := range h.rules {
		if rule.CheckOutput == nil {
			continue
		}
		violated, reason := rule.CheckOutput(response)
		if !violated {
			continue
		}

		if h.logger != nil {
			h.logger(rule.Name, reason)
		}

		switch rule.Action {
		case GuardrailBlock:
			return nil, fmt.Errorf("guardrail %s blocked output: %s", rule.Name, reason)
		case GuardrailWarn:
			// 继续
		case GuardrailReplace:
			replaced := *response
			replaced.Content = rule.Replacement
			return &replaced, nil
		}
	}
	return response, nil
}

// ─── 4. RateLimitHook：速率限制 Hook ────────────────────────────

// RateLimitHook 速率限制钩子。
//
// 限制模型调用频率，防止超出 API 配额或过载。
type RateLimitHook struct {
	mu           sync.Mutex
	maxPerMinute int
	calls        []time.Time
}

// NewRateLimitHook 创建速率限制 Hook。
//
// 参数：
//   - maxPerMinute: 每分钟最大调用次数
func NewRateLimitHook(maxPerMinute int) *RateLimitHook {
	return &RateLimitHook{
		maxPerMinute: maxPerMinute,
		calls:        make([]time.Time, 0, maxPerMinute),
	}
}

func (h *RateLimitHook) Name() string { return "rate-limit-hook" }

func (h *RateLimitHook) BeforeModel(ctx context.Context, messages []types.Message) ([]types.Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	window := now.Add(-time.Minute)

	// 清理窗口外的调用记录
	valid := h.calls[:0]
	for _, t := range h.calls {
		if t.After(window) {
			valid = append(valid, t)
		}
	}
	h.calls = valid

	if len(h.calls) >= h.maxPerMinute {
		return nil, fmt.Errorf("rate limit exceeded: max %d calls per minute", h.maxPerMinute)
	}

	h.calls = append(h.calls, now)
	return messages, nil
}

func (h *RateLimitHook) AfterModel(_ context.Context, response *types.Message) (*types.Message, error) {
	return response, nil
}

// ─── 5. ContentFilterHook：内容关键词过滤 Hook ──────────────────

// ContentFilterHook 关键词内容过滤钩子。
type ContentFilterHook struct {
	blockedKeywords []string
	action          GuardrailAction
	replacement     string
}

// NewContentFilterHook 创建内容过滤 Hook。
func NewContentFilterHook(blockedKeywords []string, action GuardrailAction, replacement string) *ContentFilterHook {
	keywords := make([]string, len(blockedKeywords))
	for i, k := range blockedKeywords {
		keywords[i] = strings.ToLower(k)
	}
	return &ContentFilterHook{
		blockedKeywords: keywords,
		action:          action,
		replacement:     replacement,
	}
}

func (h *ContentFilterHook) Name() string { return "content-filter-hook" }

func (h *ContentFilterHook) BeforeModel(_ context.Context, messages []types.Message) ([]types.Message, error) {
	for _, m := range messages {
		lower := strings.ToLower(m.Content)
		for _, kw := range h.blockedKeywords {
			if strings.Contains(lower, kw) {
				switch h.action {
				case GuardrailBlock:
					return nil, fmt.Errorf("content filter: blocked keyword '%s' in input", kw)
				case GuardrailReplace:
					replaced := make([]types.Message, len(messages))
					copy(replaced, messages)
					for i := range replaced {
						replaced[i].Content = h.replacement
					}
					return replaced, nil
				}
			}
		}
	}
	return messages, nil
}

func (h *ContentFilterHook) AfterModel(_ context.Context, response *types.Message) (*types.Message, error) {
	if response == nil {
		return nil, nil
	}
	lower := strings.ToLower(response.Content)
	for _, kw := range h.blockedKeywords {
		if strings.Contains(lower, kw) {
			switch h.action {
			case GuardrailBlock:
				return nil, fmt.Errorf("content filter: blocked keyword '%s' in output", kw)
			case GuardrailReplace:
				replaced := *response
				replaced.Content = h.replacement
				return &replaced, nil
			}
		}
	}
	return response, nil
}

// ─── 便捷构造函数 ─────────────────────────────────────────────

// DefaultSummaryHook 创建使用默认截断策略的摘要 Hook。
//
// maxTokens: 估算 token 上限，超过时保留最近 6 条消息。
func DefaultSummaryHook(maxTokens int) *SummaryHook {
	return NewSummaryHook(maxTokens, nil, 6)
}

// DefaultPIIHook 创建使用默认规则的 PII 脱敏 Hook（脱敏输入，不脱敏输出）。
func DefaultPIIHook() *PIIRedactHook {
	return NewPIIRedactHook(nil, true, false)
}

// HookContextKey hook 相关 context key 类型。
type hookContextKey string

const (
	// HookContextKeyHookChain hook chain 的 context key
	HookContextKeyHookChain hookContextKey = "middleware:hook:chain"
)

// WithHookChain 将 HookChain 注入 context。
func WithHookChain(ctx context.Context, chain *HookChain) context.Context {
	return context.WithValue(ctx, HookContextKeyHookChain, chain)
}

// GetHookChainFromContext 从 context 获取 HookChain。
func GetHookChainFromContext(ctx context.Context) (*HookChain, bool) {
	chain, ok := ctx.Value(HookContextKeyHookChain).(*HookChain)
	return chain, ok
}

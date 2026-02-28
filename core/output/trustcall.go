package output

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// TokenUsage LLM Token 使用统计。
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ─── 推理轨迹 ───────────────────────────────────────────────────

// ThoughtStep 单步推理记录。
type ThoughtStep struct {
	// Step 步骤编号
	Step int `json:"step"`

	// Thought 思考内容
	Thought string `json:"thought,omitempty"`

	// Action 执行的动作（如工具调用）
	Action string `json:"action,omitempty"`

	// Result 动作结果
	Result string `json:"result,omitempty"`

	// Duration 该步骤耗时
	Duration time.Duration `json:"duration_ms,omitempty"`
}

// ToolCallRecord 工具调用记录。
type ToolCallRecord struct {
	// ToolName 工具名称
	ToolName string `json:"tool_name"`

	// Args 调用参数（JSON）
	Args string `json:"args"`

	// Result 工具返回结果
	Result string `json:"result,omitempty"`

	// Error 调用错误
	Error string `json:"error,omitempty"`

	// Duration 调用耗时
	Duration time.Duration `json:"duration_ms"`

	// Timestamp 调用时间
	Timestamp time.Time `json:"timestamp"`
}

// ReasoningTrace 推理轨迹，记录 Agent 完整的思考过程。
//
// 对标 LangChain 1.0 Standard Content Blocks 中的推理轨迹功能，
// 与 `types.ContentBlockThinking` 互补（此类更加结构化）。
//
// 使用示例：
//
//	tracer := output.NewReasoningTracer()
//
//	tracer.AddThought("首先分析用户需求")
//	result, _ := callTool("search", args)
//	tracer.AddToolCall("search", args, result, nil)
//	tracer.AddThought("根据搜索结果，用户需要...")
//
//	trace := tracer.Finish()
//	fmt.Println(trace.Summary())
type ReasoningTrace struct {
	// Steps 思考步骤
	Steps []ThoughtStep `json:"steps"`

	// ToolCalls 工具调用记录
	ToolCalls []ToolCallRecord `json:"tool_calls,omitempty"`

	// StartTime 开始时间
	StartTime time.Time `json:"start_time"`

	// EndTime 结束时间
	EndTime time.Time `json:"end_time"`

	// TokenUsage Token 使用统计
	TokenUsage *TokenUsage `json:"token_usage,omitempty"`
}

// Duration 返回总耗时。
func (r *ReasoningTrace) Duration() time.Duration {
	if r.EndTime.IsZero() {
		return time.Since(r.StartTime)
	}
	return r.EndTime.Sub(r.StartTime)
}

// Summary 返回推理摘要字符串。
func (r *ReasoningTrace) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ReasoningTrace: %d steps, %d tool calls, duration=%s",
		len(r.Steps), len(r.ToolCalls), r.Duration().Round(time.Millisecond)))
	if len(r.Steps) > 0 {
		sb.WriteString("\nSteps:")
		for _, s := range r.Steps {
			if s.Thought != "" {
				sb.WriteString(fmt.Sprintf("\n  [%d] %s", s.Step, s.Thought))
			}
		}
	}
	return sb.String()
}

// ReasoningTracer 推理轨迹记录器。
type ReasoningTracer struct {
	trace   ReasoningTrace
	stepNum int
}

// NewReasoningTracer 创建推理轨迹记录器。
func NewReasoningTracer() *ReasoningTracer {
	return &ReasoningTracer{
		trace: ReasoningTrace{
			Steps:     make([]ThoughtStep, 0),
			ToolCalls: make([]ToolCallRecord, 0),
			StartTime: time.Now(),
		},
	}
}

// AddThought 添加思考步骤。
func (t *ReasoningTracer) AddThought(thought string) *ReasoningTracer {
	t.stepNum++
	t.trace.Steps = append(t.trace.Steps, ThoughtStep{
		Step:    t.stepNum,
		Thought: thought,
	})
	return t
}

// AddAction 添加动作步骤。
func (t *ReasoningTracer) AddAction(thought, action, result string, duration time.Duration) *ReasoningTracer {
	t.stepNum++
	t.trace.Steps = append(t.trace.Steps, ThoughtStep{
		Step:     t.stepNum,
		Thought:  thought,
		Action:   action,
		Result:   result,
		Duration: duration,
	})
	return t
}

// AddToolCall 记录工具调用。
func (t *ReasoningTracer) AddToolCall(toolName, args, result string, err error, duration time.Duration) *ReasoningTracer {
	record := ToolCallRecord{
		ToolName:  toolName,
		Args:      args,
		Result:    result,
		Duration:  duration,
		Timestamp: time.Now(),
	}
	if err != nil {
		record.Error = err.Error()
	}
	t.trace.ToolCalls = append(t.trace.ToolCalls, record)
	return t
}

// WithTokenUsage 记录 Token 使用情况。
func (t *ReasoningTracer) WithTokenUsage(usage *TokenUsage) *ReasoningTracer {
	t.trace.TokenUsage = usage
	return t
}

// Finish 完成记录，返回推理轨迹。
func (t *ReasoningTracer) Finish() *ReasoningTrace {
	t.trace.EndTime = time.Now()
	result := t.trace
	return &result
}

// ToContentBlock 将推理轨迹转换为 ContentBlock（与 types 包互操作）。
func (t *ReasoningTracer) ToContentBlock() *types.ContentBlock {
	trace := t.Finish()
	var thoughts []string
	for _, s := range trace.Steps {
		if s.Thought != "" {
			thoughts = append(thoughts, fmt.Sprintf("[%d] %s", s.Step, s.Thought))
		}
	}

	block := types.NewThinkingContentBlock(strings.Join(thoughts, "\n"))
	block.WithMetadata("tool_calls_count", len(trace.ToolCalls))
	block.WithMetadata("duration_ms", trace.Duration().Milliseconds())
	return block
}

// ─── JSONPatch 结构化提取（Trustcall 对标）─────────────────────

// JSONPatchOp JSON Patch 操作类型（RFC 6902）。
type JSONPatchOp string

const (
	JSONPatchAdd     JSONPatchOp = "add"
	JSONPatchRemove  JSONPatchOp = "remove"
	JSONPatchReplace JSONPatchOp = "replace"
	JSONPatchCopy    JSONPatchOp = "copy"
	JSONPatchMove    JSONPatchOp = "move"
	JSONPatchTest    JSONPatchOp = "test"
)

// JSONPatch 单个 JSON Patch 操作（RFC 6902）。
type JSONPatch struct {
	Op    JSONPatchOp `json:"op"`
	Path  string      `json:"path"`
	Value any         `json:"value,omitempty"`
	From  string      `json:"from,omitempty"`
}

// ReliableExtractorConfig 可靠结构化提取器配置。
type ReliableExtractorConfig struct {
	// MaxRetries 最大重试次数（默认 3）
	MaxRetries int

	// UsePatch 提取失败时是否使用 JSON Patch 修复而非重新提取（默认 true）
	// JSON Patch 方式更便宜，只需让 LLM 输出差异，而非完整 JSON
	UsePatch bool

	// SystemPrompt 自定义系统提示（nil 时使用自动生成的）
	SystemPrompt *string

	// ValidateFunc 额外的自定义验证函数
	ValidateFunc func(v any) error
}

// DefaultReliableExtractorConfig 返回默认配置。
func DefaultReliableExtractorConfig() ReliableExtractorConfig {
	return ReliableExtractorConfig{
		MaxRetries: 3,
		UsePatch:   true,
	}
}

// LLMCaller LLM 调用函数类型（避免循环依赖）。
type LLMCaller func(ctx context.Context, messages []types.Message) (string, error)

// ReliableExtractor 可靠结构化数据提取器。
//
// 对标 Trustcall，使用 JSON Patch 技术在提取失败时修复 JSON，
// 而非重新调用 LLM 生成完整 JSON，大幅降低成本和延迟。
//
// 提取流程：
//  1. 调用 LLM 提取 JSON
//  2. 解析并验证 JSON 结构
//  3. 如验证失败，生成 JSON Patch 提示，让 LLM 只输出差异部分
//  4. 应用 Patch 修复原始 JSON
//  5. 重新验证，最多重试 MaxRetries 次
//
// 使用示例：
//
//	type UserProfile struct {
//	    Name  string `json:"name"  validate:"required"`
//	    Email string `json:"email" validate:"required,email"`
//	    Age   int    `json:"age"   validate:"min=0,max=150"`
//	}
//
//	extractor := output.NewReliableExtractor[UserProfile](
//	    llmCaller,
//	    output.DefaultReliableExtractorConfig(),
//	)
//
//	profile, trace, err := extractor.Extract(ctx,
//	    "用户名叫 Alice，邮箱是 alice@example.com，今年 28 岁",
//	)
type ReliableExtractor[T any] struct {
	caller  LLMCaller
	config  ReliableExtractorConfig
	schema  map[string]any
}

// NewReliableExtractor 创建可靠结构化提取器。
//
// 参数：
//   - caller: LLM 调用函数
//   - config: 提取器配置
func NewReliableExtractor[T any](caller LLMCaller, config ReliableExtractorConfig) *ReliableExtractor[T] {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	var zero T
	schema := generateJSONSchema(zero)

	return &ReliableExtractor[T]{
		caller: caller,
		config: config,
		schema: schema,
	}
}

// ExtractionResult 提取结果。
type ExtractionResult[T any] struct {
	// Value 提取到的值
	Value *T

	// RawJSON 原始 JSON 字符串
	RawJSON string

	// Attempts 实际尝试次数
	Attempts int

	// UsedPatch JSON Patch 修复是否被使用
	UsedPatch bool

	// Trace 推理轨迹
	Trace *ReasoningTrace
}

// Extract 从文本中可靠地提取结构化数据。
//
// 返回：
//   - *ExtractionResult[T]: 提取结果（包含值、原始 JSON、尝试次数等）
//   - error: 所有重试都失败时返回错误
func (e *ReliableExtractor[T]) Extract(ctx context.Context, text string) (*ExtractionResult[T], error) {
	tracer := NewReasoningTracer()
	result := &ExtractionResult[T]{}

	schemaJSON, _ := json.MarshalIndent(e.schema, "", "  ")
	systemPrompt := e.buildSystemPrompt(string(schemaJSON))

	var lastJSON string
	var lastErr error

	for attempt := 1; attempt <= e.config.MaxRetries; attempt++ {
		result.Attempts = attempt

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var prompt string
		if attempt == 1 || !e.config.UsePatch || lastJSON == "" {
			// 首次：直接提取
			prompt = fmt.Sprintf("请从以下文本中提取信息：\n\n%s", text)
			tracer.AddThought(fmt.Sprintf("第 %d 次尝试：直接提取 JSON", attempt))
		} else {
			// 后续：使用 JSON Patch 修复
			prompt = e.buildPatchPrompt(lastJSON, lastErr)
			tracer.AddThought(fmt.Sprintf("第 %d 次尝试：使用 JSON Patch 修复上次错误: %v", attempt, lastErr))
			result.UsedPatch = true
		}

		messages := []types.Message{
			{Role: types.RoleSystem, Content: systemPrompt},
			{Role: types.RoleUser, Content: prompt},
		}

		start := time.Now()
		rawOutput, err := e.caller(ctx, messages)
		duration := time.Since(start)
		tracer.AddToolCall("llm", prompt, rawOutput, err, duration)

		if err != nil {
			lastErr = fmt.Errorf("LLM call failed: %w", err)
			continue
		}

		// 提取 JSON
		jsonStr := extractJSONForTrustcall(rawOutput)
		if jsonStr == "" {
			lastErr = fmt.Errorf("no valid JSON found in LLM output")
			lastJSON = rawOutput
			continue
		}

		// 如果是 Patch 模式，应用 Patch
		if e.config.UsePatch && lastJSON != "" && attempt > 1 {
			patched, err := applyJSONPatches(lastJSON, jsonStr)
			if err != nil {
				// Patch 失败，把整个输出当作新 JSON
				jsonStr = patched
			} else {
				jsonStr = patched
			}
		}

		// 解析 JSON
		var value T
		if err := json.Unmarshal([]byte(jsonStr), &value); err != nil {
			lastErr = fmt.Errorf("JSON unmarshal failed: %w", err)
			lastJSON = jsonStr
			continue
		}

		// 自定义验证
		if e.config.ValidateFunc != nil {
			if err := e.config.ValidateFunc(&value); err != nil {
				lastErr = fmt.Errorf("validation failed: %w", err)
				lastJSON = jsonStr
				continue
			}
		}

		// 成功
		result.Value = &value
		result.RawJSON = jsonStr
		result.Trace = tracer.Finish()
		tracer.AddThought(fmt.Sprintf("提取成功，共 %d 次尝试", attempt))
		return result, nil
	}

	result.Trace = tracer.Finish()
	return result, fmt.Errorf("reliable extractor: all %d attempts failed, last error: %w",
		e.config.MaxRetries, lastErr)
}

// buildSystemPrompt 构建系统提示。
func (e *ReliableExtractor[T]) buildSystemPrompt(schemaJSON string) string {
	if e.config.SystemPrompt != nil {
		return *e.config.SystemPrompt
	}

	return fmt.Sprintf(`你是一个数据提取专家。请从用户提供的文本中提取符合以下 JSON Schema 的数据。

JSON Schema:
%s

要求：
1. 只输出合法的 JSON，不要包含任何解释文字
2. 严格遵循 Schema 中定义的字段名和类型
3. 如果某字段在文本中找不到，使用该类型的零值（字符串用 ""，数字用 0，布尔用 false）
4. 不要在 JSON 外面包裹 markdown 代码块

直接输出 JSON 对象：`, schemaJSON)
}

// buildPatchPrompt 构建 JSON Patch 修复提示。
func (e *ReliableExtractor[T]) buildPatchPrompt(lastJSON string, lastErr error) string {
	return fmt.Sprintf(`上一次提取的 JSON 存在问题：%v

当前 JSON（有问题）：
%s

请输出 RFC 6902 格式的 JSON Patch 数组来修复问题。
只需输出 Patch 操作数组，例如：
[
  {"op": "replace", "path": "/field_name", "value": "correct_value"},
  {"op": "add", "path": "/missing_field", "value": 0}
]

直接输出 JSON Patch 数组：`, lastErr, lastJSON)
}

// applyJSONPatches 应用 JSON Patch 操作。
// 这是一个简化实现，只支持 replace 和 add 操作。
func applyJSONPatches(original, patchJSON string) (string, error) {
	// 先尝试解析为 Patch 数组
	var patches []JSONPatch
	if err := json.Unmarshal([]byte(patchJSON), &patches); err != nil {
		// 如果不是 Patch 格式，当作完整 JSON 处理
		return patchJSON, nil
	}

	if len(patches) == 0 {
		return original, nil
	}

	// 解析原始 JSON
	var doc map[string]any
	if err := json.Unmarshal([]byte(original), &doc); err != nil {
		return original, fmt.Errorf("parse original JSON: %w", err)
	}

	// 应用 Patch
	for _, patch := range patches {
		// 简化实现：只处理顶层字段（path 格式 "/field"）
		if !strings.HasPrefix(patch.Path, "/") {
			continue
		}
		field := strings.TrimPrefix(patch.Path, "/")
		if field == "" || strings.Contains(field, "/") {
			continue
		}

		switch patch.Op {
		case JSONPatchReplace, JSONPatchAdd:
			doc[field] = patch.Value
		case JSONPatchRemove:
			delete(doc, field)
		}
	}

	// 序列化回 JSON
	result, err := json.Marshal(doc)
	if err != nil {
		return original, fmt.Errorf("marshal patched JSON: %w", err)
	}
	return string(result), nil
}

// generateJSONSchema 从类型生成 JSON Schema（简化版）。
func generateJSONSchema(v any) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return schema
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return schema
	}

	properties := map[string]any{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}

		fieldSchema := goTypeToJSONSchema(field.Type)
		properties[fieldName] = fieldSchema

		// validate 标签中的 required
		validateTag := field.Tag.Get("validate")
		if strings.Contains(validateTag, "required") {
			required = append(required, fieldName)
		}
	}

	schema["properties"] = properties
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// goTypeToJSONSchema 将 Go 类型转换为 JSON Schema 类型描述。
func goTypeToJSONSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "minimum": 0}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": goTypeToJSONSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object"}
	case reflect.Struct:
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "string"}
	}
}

// isValidJSONValue 检查字符串是否是合法 JSON（包括 object 和 array）。
func isValidJSONValue(s string) bool {
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}

// extractJSONForTrustcall 从文本中提取 JSON 字符串（供 trustcall 使用）。
func extractJSONForTrustcall(text string) string {
	text = strings.TrimSpace(text)

	// 尝试直接解析（包括数组）
	if isValidJSONValue(text) {
		return text
	}

	// 从 Markdown 代码块中提取
	patterns := []struct{ start, end string }{
		{"```json\n", "\n```"},
		{"```json\r\n", "\r\n```"},
		{"```\n", "\n```"},
		{"```", "```"},
	}

	for _, p := range patterns {
		if idx := strings.Index(text, p.start); idx >= 0 {
			start := idx + len(p.start)
			if end := strings.Index(text[start:], p.end); end >= 0 {
				candidate := strings.TrimSpace(text[start : start+end])
				if isValidJSONValue(candidate) {
					return candidate
				}
			}
		}
	}

	// 查找 { 开头的 JSON 对象
	if start := strings.Index(text, "{"); start >= 0 {
		for end := len(text) - 1; end > start; end-- {
			if text[end] == '}' {
				candidate := text[start : end+1]
				if isValidJSONValue(candidate) {
					return candidate
				}
			}
		}
	}

	// 查找 [ 开头的 JSON 数组（用于 Patch）
	if start := strings.Index(text, "["); start >= 0 {
		for end := len(text) - 1; end > start; end-- {
			if text[end] == ']' {
				candidate := text[start : end+1]
				if isValidJSONValue(candidate) {
					return candidate
				}
			}
		}
	}

	return ""
}

// isValidJSON 检查字符串是否是合法 JSON。
func isValidJSON(s string) bool {
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}

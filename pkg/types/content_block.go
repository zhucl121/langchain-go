package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// ContentBlockType 内容块类型
type ContentBlockType string

const (
	// ContentBlockText 文本内容块
	ContentBlockText ContentBlockType = "text"
	// ContentBlockThinking 思考过程内容块（用于 o1 等模型）
	ContentBlockThinking ContentBlockType = "thinking"
	// ContentBlockToolUse 工具使用内容块
	ContentBlockToolUse ContentBlockType = "tool_use"
	// ContentBlockToolResult 工具结果内容块
	ContentBlockToolResult ContentBlockType = "tool_result"
	// ContentBlockImage 图像内容块
	ContentBlockImage ContentBlockType = "image"
	// ContentBlockError 错误内容块
	ContentBlockError ContentBlockType = "error"
)

// ContentBlock 标准内容块。
//
// ContentBlock 是 LangChain v1.0+ 标准化的输出格式，支持：
// - 推理过程追踪（Reasoning Trace）
// - 引用来源（Citations）
// - 工具调用（Tool Calls）
// - 元数据（Metadata）
//
// 示例：
//
//	block := types.NewTextContentBlock("这是答案")
//	block.WithReasoning([]string{
//	    "步骤1: 分析问题",
//	    "步骤2: 查找资料",
//	    "步骤3: 得出结论",
//	})
//	block.WithCitation(types.Citation{
//	    Source: "doc1.pdf",
//	    Excerpt: "相关内容...",
//	    Score: 0.95,
//	})
//
type ContentBlock struct {
	// Type 内容块类型
	Type ContentBlockType `json:"type"`

	// Content 主要内容
	Content string `json:"content"`

	// Reasoning 推理步骤（用于追踪思考过程）
	Reasoning []string `json:"reasoning,omitempty"`

	// Citations 引用来源（用于 RAG 系统）
	Citations []Citation `json:"citations,omitempty"`

	// ToolCalls 工具调用列表（如果是工具使用块）
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Metadata 附加元数据
	Metadata map[string]any `json:"metadata,omitempty"`

	// Timestamp 创建时间
	Timestamp time.Time `json:"timestamp,omitempty"`

	// ID 内容块唯一标识
	ID string `json:"id,omitempty"`

	// ParentID 父内容块 ID（用于嵌套结构）
	ParentID string `json:"parent_id,omitempty"`

	// Confidence 置信度（0-1）
	Confidence *float64 `json:"confidence,omitempty"`

	// Error 错误信息（如果类型是 error）
	Error *ErrorInfo `json:"error,omitempty"`
}

// Citation 引用来源。
//
// Citation 用于记录内容的来源，支持 RAG 系统的引用追溯。
//
type Citation struct {
	// Source 来源标识（文件名、URL、文档 ID 等）
	Source string `json:"source"`

	// Excerpt 引用片段（原文摘录）
	Excerpt string `json:"excerpt,omitempty"`

	// Score 相似度分数（0-1）
	Score float64 `json:"score,omitempty"`

	// Page 页码（如果适用）
	Page *int `json:"page,omitempty"`

	// StartChar 起始字符位置
	StartChar *int `json:"start_char,omitempty"`

	// EndChar 结束字符位置
	EndChar *int `json:"end_char,omitempty"`

	// Metadata 附加元数据
	Metadata map[string]any `json:"metadata,omitempty"`

	// Title 来源标题
	Title string `json:"title,omitempty"`

	// URL 来源 URL（如果适用）
	URL string `json:"url,omitempty"`
}

// ErrorInfo 错误信息。
//
// ErrorInfo 用于标准化错误输出。
//
type ErrorInfo struct {
	// Code 错误码
	Code string `json:"code"`

	// Message 错误消息
	Message string `json:"message"`

	// Details 详细信息
	Details map[string]any `json:"details,omitempty"`

	// Recoverable 是否可恢复
	Recoverable bool `json:"recoverable"`
}

// NewTextContentBlock 创建文本内容块。
//
// 参数：
//   - content: 文本内容
//
// 返回：
//   - *ContentBlock: 内容块实例
//
func NewTextContentBlock(content string) *ContentBlock {
	return &ContentBlock{
		Type:      ContentBlockText,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// NewThinkingContentBlock 创建思考过程内容块。
//
// 思考过程块用于记录 AI 的推理过程（如 OpenAI o1 模型）。
//
// 参数：
//   - content: 思考内容
//
// 返回：
//   - *ContentBlock: 内容块实例
//
func NewThinkingContentBlock(content string) *ContentBlock {
	return &ContentBlock{
		Type:      ContentBlockThinking,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// NewToolUseContentBlock 创建工具使用内容块。
//
// 参数：
//   - toolCalls: 工具调用列表
//
// 返回：
//   - *ContentBlock: 内容块实例
//
func NewToolUseContentBlock(toolCalls []ToolCall) *ContentBlock {
	return &ContentBlock{
		Type:      ContentBlockToolUse,
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// NewToolResultContentBlock 创建工具结果内容块。
//
// 参数：
//   - content: 工具执行结果
//
// 返回：
//   - *ContentBlock: 内容块实例
//
func NewToolResultContentBlock(content string) *ContentBlock {
	return &ContentBlock{
		Type:      ContentBlockToolResult,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// NewErrorContentBlock 创建错误内容块。
//
// 参数：
//   - code: 错误码
//   - message: 错误消息
//
// 返回：
//   - *ContentBlock: 内容块实例
//
func NewErrorContentBlock(code, message string) *ContentBlock {
	return &ContentBlock{
		Type:    ContentBlockError,
		Content: message,
		Error: &ErrorInfo{
			Code:        code,
			Message:     message,
			Recoverable: false,
		},
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// WithReasoning 添加推理步骤。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - reasoning: 推理步骤列表
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) WithReasoning(reasoning []string) *ContentBlock {
	cb.Reasoning = reasoning
	return cb
}

// AddReasoning 添加单个推理步骤。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - step: 推理步骤
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) AddReasoning(step string) *ContentBlock {
	if cb.Reasoning == nil {
		cb.Reasoning = make([]string, 0)
	}
	cb.Reasoning = append(cb.Reasoning, step)
	return cb
}

// WithCitations 设置引用来源列表。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - citations: 引用来源列表
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) WithCitations(citations []Citation) *ContentBlock {
	cb.Citations = citations
	return cb
}

// AddCitation 添加单个引用来源。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - citation: 引用来源
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) AddCitation(citation Citation) *ContentBlock {
	if cb.Citations == nil {
		cb.Citations = make([]Citation, 0)
	}
	cb.Citations = append(cb.Citations, citation)
	return cb
}

// WithID 设置内容块 ID。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - id: 内容块 ID
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) WithID(id string) *ContentBlock {
	cb.ID = id
	return cb
}

// WithParentID 设置父内容块 ID。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - parentID: 父内容块 ID
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) WithParentID(parentID string) *ContentBlock {
	cb.ParentID = parentID
	return cb
}

// WithConfidence 设置置信度。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - confidence: 置信度（0-1）
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) WithConfidence(confidence float64) *ContentBlock {
	cb.Confidence = &confidence
	return cb
}

// WithMetadata 添加元数据。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - key: 元数据键
//   - value: 元数据值
//
// 返回：
//   - *ContentBlock: 自身
//
func (cb *ContentBlock) WithMetadata(key string, value any) *ContentBlock {
	if cb.Metadata == nil {
		cb.Metadata = make(map[string]any)
	}
	cb.Metadata[key] = value
	return cb
}

// ToJSON 转换为 JSON 字符串。
//
// 返回：
//   - string: JSON 字符串
//   - error: 错误
//
func (cb *ContentBlock) ToJSON() (string, error) {
	data, err := json.MarshalIndent(cb, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal content block: %w", err)
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串解析。
//
// 参数：
//   - data: JSON 字符串
//
// 返回：
//   - error: 错误
//
func (cb *ContentBlock) FromJSON(data string) error {
	if err := json.Unmarshal([]byte(data), cb); err != nil {
		return fmt.Errorf("failed to unmarshal content block: %w", err)
	}
	return nil
}

// Validate 验证内容块的有效性。
//
// 返回：
//   - error: 验证失败时返回错误
//
func (cb *ContentBlock) Validate() error {
	// 检查类型
	switch cb.Type {
	case ContentBlockText, ContentBlockThinking, ContentBlockToolUse,
		ContentBlockToolResult, ContentBlockImage, ContentBlockError:
		// 有效类型
	default:
		return fmt.Errorf("invalid content block type: %s", cb.Type)
	}

	// 检查置信度范围
	if cb.Confidence != nil {
		if *cb.Confidence < 0 || *cb.Confidence > 1 {
			return fmt.Errorf("confidence must be between 0 and 1, got: %f", *cb.Confidence)
		}
	}

	// 检查工具调用块必须有 ToolCalls
	if cb.Type == ContentBlockToolUse && len(cb.ToolCalls) == 0 {
		return fmt.Errorf("tool_use block must have at least one tool call")
	}

	// 检查错误块必须有 Error 信息
	if cb.Type == ContentBlockError && cb.Error == nil {
		return fmt.Errorf("error block must have error info")
	}

	return nil
}

// Clone 创建内容块的深拷贝。
//
// 返回：
//   - *ContentBlock: 内容块副本
//
func (cb *ContentBlock) Clone() *ContentBlock {
	clone := &ContentBlock{
		Type:      cb.Type,
		Content:   cb.Content,
		Timestamp: cb.Timestamp,
		ID:        cb.ID,
		ParentID:  cb.ParentID,
	}

	// 深拷贝 Reasoning
	if cb.Reasoning != nil {
		clone.Reasoning = make([]string, len(cb.Reasoning))
		copy(clone.Reasoning, cb.Reasoning)
	}

	// 深拷贝 Citations
	if cb.Citations != nil {
		clone.Citations = make([]Citation, len(cb.Citations))
		copy(clone.Citations, cb.Citations)
	}

	// 深拷贝 ToolCalls
	if cb.ToolCalls != nil {
		clone.ToolCalls = make([]ToolCall, len(cb.ToolCalls))
		copy(clone.ToolCalls, cb.ToolCalls)
	}

	// 深拷贝 Metadata
	if cb.Metadata != nil {
		clone.Metadata = make(map[string]any, len(cb.Metadata))
		for k, v := range cb.Metadata {
			clone.Metadata[k] = v
		}
	}

	// 深拷贝 Confidence
	if cb.Confidence != nil {
		conf := *cb.Confidence
		clone.Confidence = &conf
	}

	// 深拷贝 Error
	if cb.Error != nil {
		clone.Error = &ErrorInfo{
			Code:        cb.Error.Code,
			Message:     cb.Error.Message,
			Recoverable: cb.Error.Recoverable,
		}
		if cb.Error.Details != nil {
			clone.Error.Details = make(map[string]any, len(cb.Error.Details))
			for k, v := range cb.Error.Details {
				clone.Error.Details[k] = v
			}
		}
	}

	return clone
}

// String 实现 Stringer 接口，用于调试输出。
func (cb *ContentBlock) String() string {
	contentPreview := cb.Content
	if len(contentPreview) > 100 {
		contentPreview = contentPreview[:100] + "..."
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Type:%s", cb.Type))
	parts = append(parts, fmt.Sprintf("Content:%q", contentPreview))

	if len(cb.Reasoning) > 0 {
		parts = append(parts, fmt.Sprintf("Reasoning:%d steps", len(cb.Reasoning)))
	}

	if len(cb.Citations) > 0 {
		parts = append(parts, fmt.Sprintf("Citations:%d", len(cb.Citations)))
	}

	if len(cb.ToolCalls) > 0 {
		parts = append(parts, fmt.Sprintf("ToolCalls:%d", len(cb.ToolCalls)))
	}

	if cb.Confidence != nil {
		parts = append(parts, fmt.Sprintf("Confidence:%.2f", *cb.Confidence))
	}

	result := "ContentBlock{"
	for i, part := range parts {
		if i > 0 {
			result += ", "
		}
		result += part
	}
	result += "}"

	return result
}

// ContentBlockList 内容块列表。
//
// ContentBlockList 提供便捷方法管理多个内容块。
//
type ContentBlockList struct {
	Blocks []*ContentBlock `json:"blocks"`
}

// NewContentBlockList 创建内容块列表。
//
// 返回：
//   - *ContentBlockList: 内容块列表实例
//
func NewContentBlockList() *ContentBlockList {
	return &ContentBlockList{
		Blocks: make([]*ContentBlock, 0),
	}
}

// Add 添加内容块。
//
// 返回 self，支持链式调用。
//
// 参数：
//   - block: 内容块
//
// 返回：
//   - *ContentBlockList: 自身
//
func (cbl *ContentBlockList) Add(block *ContentBlock) *ContentBlockList {
	cbl.Blocks = append(cbl.Blocks, block)
	return cbl
}

// GetByType 按类型获取内容块。
//
// 参数：
//   - blockType: 内容块类型
//
// 返回：
//   - []*ContentBlock: 匹配的内容块列表
//
func (cbl *ContentBlockList) GetByType(blockType ContentBlockType) []*ContentBlock {
	result := make([]*ContentBlock, 0)
	for _, block := range cbl.Blocks {
		if block.Type == blockType {
			result = append(result, block)
		}
	}
	return result
}

// GetByID 按 ID 获取内容块。
//
// 参数：
//   - id: 内容块 ID
//
// 返回：
//   - *ContentBlock: 内容块，如果不存在返回 nil
//
func (cbl *ContentBlockList) GetByID(id string) *ContentBlock {
	for _, block := range cbl.Blocks {
		if block.ID == id {
			return block
		}
	}
	return nil
}

// GetTextContent 获取所有文本内容（拼接）。
//
// 返回：
//   - string: 拼接后的文本内容
//
func (cbl *ContentBlockList) GetTextContent() string {
	var result string
	for i, block := range cbl.Blocks {
		if block.Type == ContentBlockText || block.Type == ContentBlockThinking {
			if i > 0 && result != "" {
				result += "\n"
			}
			result += block.Content
		}
	}
	return result
}

// GetAllCitations 获取所有引用来源。
//
// 返回：
//   - []Citation: 所有引用来源
//
func (cbl *ContentBlockList) GetAllCitations() []Citation {
	citations := make([]Citation, 0)
	for _, block := range cbl.Blocks {
		if len(block.Citations) > 0 {
			citations = append(citations, block.Citations...)
		}
	}
	return citations
}

// ToJSON 转换为 JSON 字符串。
//
// 返回：
//   - string: JSON 字符串
//   - error: 错误
//
func (cbl *ContentBlockList) ToJSON() (string, error) {
	data, err := json.MarshalIndent(cbl, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal content block list: %w", err)
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串解析。
//
// 参数：
//   - data: JSON 字符串
//
// 返回：
//   - error: 错误
//
func (cbl *ContentBlockList) FromJSON(data string) error {
	if err := json.Unmarshal([]byte(data), cbl); err != nil {
		return fmt.Errorf("failed to unmarshal content block list: %w", err)
	}
	return nil
}

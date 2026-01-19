package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewTextContentBlock(t *testing.T) {
	content := "这是一个测试内容"
	block := NewTextContentBlock(content)

	if block.Type != ContentBlockText {
		t.Errorf("expected type %s, got %s", ContentBlockText, block.Type)
	}

	if block.Content != content {
		t.Errorf("expected content %q, got %q", content, block.Content)
	}

	if block.Metadata == nil {
		t.Error("metadata should be initialized")
	}

	if block.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
}

func TestNewThinkingContentBlock(t *testing.T) {
	content := "我正在思考这个问题..."
	block := NewThinkingContentBlock(content)

	if block.Type != ContentBlockThinking {
		t.Errorf("expected type %s, got %s", ContentBlockThinking, block.Type)
	}

	if block.Content != content {
		t.Errorf("expected content %q, got %q", content, block.Content)
	}
}

func TestNewToolUseContentBlock(t *testing.T) {
	toolCalls := []ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: FunctionCall{
				Name:      "search",
				Arguments: `{"query": "test"}`,
			},
		},
	}

	block := NewToolUseContentBlock(toolCalls)

	if block.Type != ContentBlockToolUse {
		t.Errorf("expected type %s, got %s", ContentBlockToolUse, block.Type)
	}

	if len(block.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(block.ToolCalls))
	}

	if block.ToolCalls[0].ID != "call_1" {
		t.Errorf("expected tool call ID 'call_1', got %q", block.ToolCalls[0].ID)
	}
}

func TestNewToolResultContentBlock(t *testing.T) {
	result := "搜索结果：找到3条记录"
	block := NewToolResultContentBlock(result)

	if block.Type != ContentBlockToolResult {
		t.Errorf("expected type %s, got %s", ContentBlockToolResult, block.Type)
	}

	if block.Content != result {
		t.Errorf("expected content %q, got %q", result, block.Content)
	}
}

func TestNewErrorContentBlock(t *testing.T) {
	code := "INVALID_INPUT"
	message := "输入参数无效"
	block := NewErrorContentBlock(code, message)

	if block.Type != ContentBlockError {
		t.Errorf("expected type %s, got %s", ContentBlockError, block.Type)
	}

	if block.Error == nil {
		t.Fatal("error info should be set")
	}

	if block.Error.Code != code {
		t.Errorf("expected error code %q, got %q", code, block.Error.Code)
	}

	if block.Error.Message != message {
		t.Errorf("expected error message %q, got %q", message, block.Error.Message)
	}
}

func TestContentBlock_WithReasoning(t *testing.T) {
	block := NewTextContentBlock("答案")
	reasoning := []string{
		"步骤1: 分析问题",
		"步骤2: 查找资料",
		"步骤3: 得出结论",
	}

	result := block.WithReasoning(reasoning)

	// 检查是否返回自身（链式调用）
	if result != block {
		t.Error("should return self for chaining")
	}

	if len(block.Reasoning) != 3 {
		t.Errorf("expected 3 reasoning steps, got %d", len(block.Reasoning))
	}

	if block.Reasoning[0] != reasoning[0] {
		t.Errorf("expected first reasoning %q, got %q", reasoning[0], block.Reasoning[0])
	}
}

func TestContentBlock_AddReasoning(t *testing.T) {
	block := NewTextContentBlock("答案")

	block.AddReasoning("步骤1")
	block.AddReasoning("步骤2")

	if len(block.Reasoning) != 2 {
		t.Errorf("expected 2 reasoning steps, got %d", len(block.Reasoning))
	}

	if block.Reasoning[1] != "步骤2" {
		t.Errorf("expected second reasoning '步骤2', got %q", block.Reasoning[1])
	}
}

func TestContentBlock_WithCitations(t *testing.T) {
	block := NewTextContentBlock("答案")
	citations := []Citation{
		{
			Source:  "doc1.pdf",
			Excerpt: "相关内容片段",
			Score:   0.95,
		},
		{
			Source: "doc2.pdf",
			Score:  0.88,
		},
	}

	block.WithCitations(citations)

	if len(block.Citations) != 2 {
		t.Errorf("expected 2 citations, got %d", len(block.Citations))
	}

	if block.Citations[0].Source != "doc1.pdf" {
		t.Errorf("expected source 'doc1.pdf', got %q", block.Citations[0].Source)
	}

	if block.Citations[0].Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", block.Citations[0].Score)
	}
}

func TestContentBlock_AddCitation(t *testing.T) {
	block := NewTextContentBlock("答案")

	citation := Citation{
		Source:  "test.pdf",
		Excerpt: "测试内容",
		Score:   0.9,
		Page:    intPtr(5),
	}

	block.AddCitation(citation)

	if len(block.Citations) != 1 {
		t.Errorf("expected 1 citation, got %d", len(block.Citations))
	}

	if block.Citations[0].Source != "test.pdf" {
		t.Errorf("expected source 'test.pdf', got %q", block.Citations[0].Source)
	}

	if *block.Citations[0].Page != 5 {
		t.Errorf("expected page 5, got %d", *block.Citations[0].Page)
	}
}

func TestContentBlock_WithConfidence(t *testing.T) {
	block := NewTextContentBlock("答案")
	confidence := 0.85

	block.WithConfidence(confidence)

	if block.Confidence == nil {
		t.Fatal("confidence should be set")
	}

	if *block.Confidence != confidence {
		t.Errorf("expected confidence %f, got %f", confidence, *block.Confidence)
	}
}

func TestContentBlock_WithMetadata(t *testing.T) {
	block := NewTextContentBlock("答案")

	block.WithMetadata("key1", "value1")
	block.WithMetadata("key2", 42)

	if len(block.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(block.Metadata))
	}

	if block.Metadata["key1"] != "value1" {
		t.Errorf("expected metadata key1='value1', got %v", block.Metadata["key1"])
	}

	if block.Metadata["key2"] != 42 {
		t.Errorf("expected metadata key2=42, got %v", block.Metadata["key2"])
	}
}

func TestContentBlock_ChainedCalls(t *testing.T) {
	block := NewTextContentBlock("答案").
		WithReasoning([]string{"步骤1", "步骤2"}).
		AddCitation(Citation{Source: "doc1.pdf", Score: 0.9}).
		WithConfidence(0.85).
		WithMetadata("model", "gpt-4")

	if len(block.Reasoning) != 2 {
		t.Errorf("expected 2 reasoning steps, got %d", len(block.Reasoning))
	}

	if len(block.Citations) != 1 {
		t.Errorf("expected 1 citation, got %d", len(block.Citations))
	}

	if block.Confidence == nil || *block.Confidence != 0.85 {
		t.Error("confidence not set correctly")
	}

	if block.Metadata["model"] != "gpt-4" {
		t.Errorf("expected metadata model='gpt-4', got %v", block.Metadata["model"])
	}
}

func TestContentBlock_Validate(t *testing.T) {
	tests := []struct {
		name    string
		block   *ContentBlock
		wantErr bool
	}{
		{
			name:    "valid text block",
			block:   NewTextContentBlock("test"),
			wantErr: false,
		},
		{
			name: "valid block with confidence",
			block: NewTextContentBlock("test").
				WithConfidence(0.5),
			wantErr: false,
		},
		{
			name: "invalid confidence too low",
			block: NewTextContentBlock("test").
				WithConfidence(-0.1),
			wantErr: true,
		},
		{
			name: "invalid confidence too high",
			block: NewTextContentBlock("test").
				WithConfidence(1.1),
			wantErr: true,
		},
		{
			name: "invalid type",
			block: &ContentBlock{
				Type:    "invalid_type",
				Content: "test",
			},
			wantErr: true,
		},
		{
			name: "tool_use without tool calls",
			block: &ContentBlock{
				Type:    ContentBlockToolUse,
				Content: "test",
			},
			wantErr: true,
		},
		{
			name: "error block without error info",
			block: &ContentBlock{
				Type:    ContentBlockError,
				Content: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContentBlock_ToJSON(t *testing.T) {
	block := NewTextContentBlock("测试内容").
		WithReasoning([]string{"步骤1"}).
		AddCitation(Citation{Source: "test.pdf", Score: 0.9})

	jsonStr, err := block.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// 验证 JSON 有效性
	var parsed ContentBlock
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("failed to parse generated JSON: %v", err)
	}

	if parsed.Content != "测试内容" {
		t.Errorf("expected content '测试内容', got %q", parsed.Content)
	}

	if len(parsed.Reasoning) != 1 {
		t.Errorf("expected 1 reasoning, got %d", len(parsed.Reasoning))
	}

	if len(parsed.Citations) != 1 {
		t.Errorf("expected 1 citation, got %d", len(parsed.Citations))
	}
}

func TestContentBlock_FromJSON(t *testing.T) {
	jsonStr := `{
		"type": "text",
		"content": "测试",
		"reasoning": ["步骤1", "步骤2"],
		"citations": [
			{
				"source": "test.pdf",
				"score": 0.9
			}
		],
		"confidence": 0.85
	}`

	var block ContentBlock
	err := block.FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if block.Type != ContentBlockText {
		t.Errorf("expected type %s, got %s", ContentBlockText, block.Type)
	}

	if block.Content != "测试" {
		t.Errorf("expected content '测试', got %q", block.Content)
	}

	if len(block.Reasoning) != 2 {
		t.Errorf("expected 2 reasoning steps, got %d", len(block.Reasoning))
	}

	if len(block.Citations) != 1 {
		t.Errorf("expected 1 citation, got %d", len(block.Citations))
	}

	if block.Confidence == nil || *block.Confidence != 0.85 {
		t.Error("confidence not parsed correctly")
	}
}

func TestContentBlock_Clone(t *testing.T) {
	original := NewTextContentBlock("原始内容").
		WithReasoning([]string{"步骤1"}).
		AddCitation(Citation{Source: "test.pdf", Score: 0.9}).
		WithConfidence(0.85).
		WithMetadata("key", "value")

	clone := original.Clone()

	// 验证克隆内容相同
	if clone.Content != original.Content {
		t.Error("cloned content mismatch")
	}

	// 验证是深拷贝（修改克隆不影响原始）
	clone.Content = "修改后的内容"
	if original.Content == clone.Content {
		t.Error("clone should be deep copy")
	}

	// 修改克隆的 reasoning
	clone.Reasoning[0] = "新步骤"
	if original.Reasoning[0] == clone.Reasoning[0] {
		t.Error("reasoning should be deep copied")
	}

	// 修改克隆的 metadata
	clone.Metadata["key"] = "new_value"
	if original.Metadata["key"] == clone.Metadata["key"] {
		t.Error("metadata should be deep copied")
	}
}

func TestContentBlock_String(t *testing.T) {
	block := NewTextContentBlock("这是一个很长的内容").
		AddReasoning("步骤1").
		AddCitation(Citation{Source: "test.pdf"}).
		WithConfidence(0.9)

	str := block.String()

	// 检查字符串包含关键信息
	if str == "" {
		t.Error("String() should not be empty")
	}

	// 基本验证（不要求具体格式）
	t.Logf("ContentBlock.String() = %s", str)
}

// TestContentBlockList 测试内容块列表
func TestContentBlockList(t *testing.T) {
	list := NewContentBlockList()

	// 添加内容块
	block1 := NewTextContentBlock("内容1").WithID("block1")
	block2 := NewThinkingContentBlock("思考过程").WithID("block2")
	block3 := NewTextContentBlock("内容2").WithID("block3")

	list.Add(block1).Add(block2).Add(block3)

	if len(list.Blocks) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(list.Blocks))
	}
}

func TestContentBlockList_GetByType(t *testing.T) {
	list := NewContentBlockList()
	list.Add(NewTextContentBlock("text1"))
	list.Add(NewThinkingContentBlock("thinking1"))
	list.Add(NewTextContentBlock("text2"))

	textBlocks := list.GetByType(ContentBlockText)
	if len(textBlocks) != 2 {
		t.Errorf("expected 2 text blocks, got %d", len(textBlocks))
	}

	thinkingBlocks := list.GetByType(ContentBlockThinking)
	if len(thinkingBlocks) != 1 {
		t.Errorf("expected 1 thinking block, got %d", len(thinkingBlocks))
	}
}

func TestContentBlockList_GetByID(t *testing.T) {
	list := NewContentBlockList()
	block1 := NewTextContentBlock("test").WithID("block1")
	block2 := NewTextContentBlock("test2").WithID("block2")

	list.Add(block1).Add(block2)

	found := list.GetByID("block2")
	if found == nil {
		t.Fatal("should find block2")
	}

	if found.ID != "block2" {
		t.Errorf("expected ID 'block2', got %q", found.ID)
	}

	notFound := list.GetByID("nonexistent")
	if notFound != nil {
		t.Error("should return nil for nonexistent ID")
	}
}

func TestContentBlockList_GetTextContent(t *testing.T) {
	list := NewContentBlockList()
	list.Add(NewTextContentBlock("第一段"))
	list.Add(NewThinkingContentBlock("思考"))
	list.Add(NewTextContentBlock("第二段"))
	list.Add(NewToolResultContentBlock("工具结果")) // 不应包含

	text := list.GetTextContent()

	// 应该包含文本和思考内容
	expectedSubstrings := []string{"第一段", "思考", "第二段"}
	for _, substr := range expectedSubstrings {
		found := false
		for i := 0; i < len(text)-len(substr)+1; i++ {
			if text[i:i+len(substr)] == substr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected text to contain %q, got %q", substr, text)
		}
	}

	// 不应包含工具结果
	if containsString(text, "工具结果") {
		t.Error("text should not contain tool result content")
	}
}

func TestContentBlockList_GetAllCitations(t *testing.T) {
	list := NewContentBlockList()

	block1 := NewTextContentBlock("内容1").
		AddCitation(Citation{Source: "doc1.pdf"}).
		AddCitation(Citation{Source: "doc2.pdf"})

	block2 := NewTextContentBlock("内容2").
		AddCitation(Citation{Source: "doc3.pdf"})

	list.Add(block1).Add(block2)

	citations := list.GetAllCitations()
	if len(citations) != 3 {
		t.Errorf("expected 3 citations, got %d", len(citations))
	}

	sources := make([]string, len(citations))
	for i, c := range citations {
		sources[i] = c.Source
	}

	expectedSources := []string{"doc1.pdf", "doc2.pdf", "doc3.pdf"}
	for _, expected := range expectedSources {
		found := false
		for _, source := range sources {
			if source == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find source %q in citations", expected)
		}
	}
}

func TestContentBlockList_JSON(t *testing.T) {
	list := NewContentBlockList()
	list.Add(NewTextContentBlock("test1"))
	list.Add(NewTextContentBlock("test2"))

	// ToJSON
	jsonStr, err := list.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// FromJSON
	var parsed ContentBlockList
	if err := parsed.FromJSON(jsonStr); err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if len(parsed.Blocks) != 2 {
		t.Errorf("expected 2 blocks after parsing, got %d", len(parsed.Blocks))
	}
}

func TestCitation_Complete(t *testing.T) {
	page := 10
	startChar := 100
	endChar := 200

	citation := Citation{
		Source:    "document.pdf",
		Excerpt:   "这是引用的内容",
		Score:     0.95,
		Page:      &page,
		StartChar: &startChar,
		EndChar:   &endChar,
		Title:     "文档标题",
		URL:       "https://example.com/doc.pdf",
		Metadata: map[string]any{
			"author": "张三",
			"year":   2024,
		},
	}

	if citation.Source != "document.pdf" {
		t.Error("source mismatch")
	}

	if *citation.Page != 10 {
		t.Error("page mismatch")
	}

	if *citation.StartChar != 100 {
		t.Error("startChar mismatch")
	}

	if citation.Metadata["author"] != "张三" {
		t.Error("metadata author mismatch")
	}
}

func TestErrorInfo(t *testing.T) {
	errInfo := &ErrorInfo{
		Code:        "INVALID_INPUT",
		Message:     "输入无效",
		Recoverable: true,
		Details: map[string]any{
			"field": "name",
			"value": "",
		},
	}

	if errInfo.Code != "INVALID_INPUT" {
		t.Error("code mismatch")
	}

	if !errInfo.Recoverable {
		t.Error("should be recoverable")
	}

	if errInfo.Details["field"] != "name" {
		t.Error("details field mismatch")
	}
}

// 辅助函数
func intPtr(i int) *int {
	return &i
}

func containsString(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark 测试
func BenchmarkContentBlock_Clone(b *testing.B) {
	block := NewTextContentBlock("benchmark content").
		WithReasoning([]string{"step1", "step2", "step3"}).
		AddCitation(Citation{Source: "test.pdf", Score: 0.9}).
		WithMetadata("key1", "value1").
		WithMetadata("key2", "value2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = block.Clone()
	}
}

func BenchmarkContentBlock_ToJSON(b *testing.B) {
	block := NewTextContentBlock("benchmark content").
		WithReasoning([]string{"step1", "step2"}).
		AddCitation(Citation{Source: "test.pdf", Score: 0.9})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = block.ToJSON()
	}
}

func BenchmarkContentBlockList_GetTextContent(b *testing.B) {
	list := NewContentBlockList()
	for i := 0; i < 100; i++ {
		list.Add(NewTextContentBlock("content " + string(rune(i))))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = list.GetTextContent()
	}
}

// 集成测试：完整的 RAG 场景
func TestContentBlock_RAGScenario(t *testing.T) {
	// 模拟 RAG 系统的完整输出
	list := NewContentBlockList()

	// 1. 添加思考过程
	thinkingBlock := NewThinkingContentBlock("我需要查找相关文档来回答这个问题").
		WithID("thinking_1").
		WithMetadata("model", "gpt-4")
	list.Add(thinkingBlock)

	// 2. 添加工具调用
	toolUseBlock := NewToolUseContentBlock([]ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: FunctionCall{
				Name:      "vector_search",
				Arguments: `{"query": "机器学习", "k": 3}`,
			},
		},
	}).WithID("tool_use_1")
	list.Add(toolUseBlock)

	// 3. 添加工具结果
	toolResultBlock := NewToolResultContentBlock("找到3个相关文档").
		WithID("tool_result_1").
		WithParentID("tool_use_1")
	list.Add(toolResultBlock)

	// 4. 添加最终答案（带引用）
	answerBlock := NewTextContentBlock("机器学习是人工智能的一个分支...").
		WithID("answer_1").
		WithReasoning([]string{
			"分析用户问题：什么是机器学习",
			"搜索相关文档",
			"综合多个来源得出答案",
		}).
		AddCitation(Citation{
			Source:  "ml_intro.pdf",
			Excerpt: "机器学习定义...",
			Score:   0.95,
			Page:    intPtr(1),
			Title:   "机器学习入门",
		}).
		AddCitation(Citation{
			Source:  "ai_handbook.pdf",
			Excerpt: "AI 与 ML 的关系...",
			Score:   0.88,
			Page:    intPtr(23),
			Title:   "人工智能手册",
		}).
		WithConfidence(0.92).
		WithMetadata("tokens", 150).
		WithMetadata("latency_ms", 1250)
	list.Add(answerBlock)

	// 验证完整性
	if len(list.Blocks) != 4 {
		t.Errorf("expected 4 blocks, got %d", len(list.Blocks))
	}

	// 验证可以提取文本内容
	textContent := list.GetTextContent()
	if textContent == "" {
		t.Error("text content should not be empty")
	}

	// 验证可以提取所有引用
	citations := list.GetAllCitations()
	if len(citations) != 2 {
		t.Errorf("expected 2 citations, got %d", len(citations))
	}

	// 验证可以序列化为 JSON
	jsonStr, err := list.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// 验证可以反序列化
	var parsed ContentBlockList
	if err := parsed.FromJSON(jsonStr); err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if len(parsed.Blocks) != 4 {
		t.Errorf("expected 4 blocks after parsing, got %d", len(parsed.Blocks))
	}

	// 验证答案块的完整性
	answerBlockParsed := parsed.GetByID("answer_1")
	if answerBlockParsed == nil {
		t.Fatal("answer block not found after parsing")
	}

	if len(answerBlockParsed.Reasoning) != 3 {
		t.Errorf("expected 3 reasoning steps, got %d", len(answerBlockParsed.Reasoning))
	}

	if len(answerBlockParsed.Citations) != 2 {
		t.Errorf("expected 2 citations, got %d", len(answerBlockParsed.Citations))
	}

	if answerBlockParsed.Confidence == nil || *answerBlockParsed.Confidence != 0.92 {
		t.Error("confidence not preserved after parsing")
	}

	t.Log("RAG scenario test passed")
	t.Logf("Generated JSON (%d bytes):\n%s", len(jsonStr), jsonStr)
}

// 测试时间戳
func TestContentBlock_Timestamp(t *testing.T) {
	before := time.Now()
	block := NewTextContentBlock("test")
	after := time.Now()

	if block.Timestamp.Before(before) || block.Timestamp.After(after) {
		t.Error("timestamp should be set to current time")
	}
}

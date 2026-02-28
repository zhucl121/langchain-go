package output

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// ─── 推理轨迹测试 ─────────────────────────────────────────────

func TestReasoningTracer_BasicFlow(t *testing.T) {
	tracer := NewReasoningTracer()

	tracer.AddThought("分析用户需求")
	tracer.AddAction("思考下一步", "search", "找到3条结果", 50*time.Millisecond)
	tracer.AddThought("整理结果")

	trace := tracer.Finish()

	if len(trace.Steps) != 3 {
		t.Errorf("want 3 steps, got %d", len(trace.Steps))
	}
	if trace.StartTime.IsZero() {
		t.Error("start time should not be zero")
	}
	if trace.EndTime.IsZero() {
		t.Error("end time should not be zero")
	}
	if trace.Duration() <= 0 {
		t.Error("duration should be positive")
	}
	t.Logf("trace summary: %s", trace.Summary())
}

func TestReasoningTracer_ToolCall(t *testing.T) {
	tracer := NewReasoningTracer()

	tracer.AddToolCall("search", `{"q":"golang"}`, "found 10 results", nil, 100*time.Millisecond)
	tracer.AddToolCall("calculator", `{"expr":"1+1"}`, "", fmt.Errorf("invalid expr"), 5*time.Millisecond)

	trace := tracer.Finish()

	if len(trace.ToolCalls) != 2 {
		t.Errorf("want 2 tool calls, got %d", len(trace.ToolCalls))
	}
	if trace.ToolCalls[1].Error == "" {
		t.Error("second tool call should have error")
	}
}

func TestReasoningTracer_WithTokenUsage(t *testing.T) {
	tracer := NewReasoningTracer()
	tracer.WithTokenUsage(&TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	})

	trace := tracer.Finish()
	if trace.TokenUsage == nil {
		t.Fatal("token usage should not be nil")
	}
	if trace.TokenUsage.TotalTokens != 150 {
		t.Errorf("want 150 total tokens, got %d", trace.TokenUsage.TotalTokens)
	}
}

func TestReasoningTracer_ToContentBlock(t *testing.T) {
	tracer := NewReasoningTracer()
	tracer.AddThought("step 1")
	tracer.AddThought("step 2")

	block := tracer.ToContentBlock()
	if block == nil {
		t.Fatal("content block should not be nil")
	}
	if block.Type != types.ContentBlockThinking {
		t.Errorf("block type should be thinking, got %s", block.Type)
	}
	if !strings.Contains(block.Content, "step 1") {
		t.Errorf("block content should contain step 1: %s", block.Content)
	}
}

// ─── Trustcall 可靠提取测试 ───────────────────────────────────

type testUserProfile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// mockLLMCaller 模拟 LLM 调用。
func mockLLMCaller(responses ...string) LLMCaller {
	idx := 0
	return func(ctx context.Context, messages []types.Message) (string, error) {
		if idx >= len(responses) {
			return responses[len(responses)-1], nil
		}
		resp := responses[idx]
		idx++
		return resp, nil
	}
}

func TestReliableExtractor_SuccessFirstAttempt(t *testing.T) {
	caller := mockLLMCaller(`{"name":"Alice","email":"alice@example.com","age":28}`)

	extractor := NewReliableExtractor[testUserProfile](caller, DefaultReliableExtractorConfig())

	ctx := context.Background()
	result, err := extractor.Extract(ctx, "用户 Alice，邮箱 alice@example.com，28岁")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if result.Value == nil {
		t.Fatal("result value should not be nil")
	}
	if result.Value.Name != "Alice" {
		t.Errorf("want name Alice, got %s", result.Value.Name)
	}
	if result.Value.Age != 28 {
		t.Errorf("want age 28, got %d", result.Value.Age)
	}
	if result.Attempts != 1 {
		t.Errorf("should succeed in 1 attempt, got %d", result.Attempts)
	}
	if result.UsedPatch {
		t.Error("should not use patch on first success")
	}
	t.Logf("trace: %s", result.Trace.Summary())
}

func TestReliableExtractor_SuccessAfterPatchRetry(t *testing.T) {
	// 第一次返回无效 JSON（触发重试），第二次返回正确 JSON
	caller := mockLLMCaller(
		`not valid json at all`,                                           // 第一次：无效
		`{"name":"Bob","email":"bob@test.com","age":35}`,                 // 第二次：正确
	)

	config := DefaultReliableExtractorConfig()
	config.UsePatch = true
	config.MaxRetries = 3
	extractor := NewReliableExtractor[testUserProfile](caller, config)

	ctx := context.Background()
	result, err := extractor.Extract(ctx, "Bob, bob@test.com, 35岁")
	if err != nil {
		t.Fatalf("Extract failed after retry: %v", err)
	}
	if result.Value.Name != "Bob" {
		t.Errorf("want Bob, got %s", result.Value.Name)
	}
	if result.Value.Age != 35 {
		t.Errorf("want age 35, got %d", result.Value.Age)
	}
	if result.Attempts < 2 {
		t.Errorf("should use at least 2 attempts, got %d", result.Attempts)
	}
	t.Logf("used patch: %v, attempts: %d", result.UsedPatch, result.Attempts)
}

func TestReliableExtractor_ExtractFromMarkdown(t *testing.T) {
	// LLM 返回 markdown 包裹的 JSON
	caller := mockLLMCaller("```json\n{\"name\":\"Carol\",\"email\":\"carol@test.com\",\"age\":30}\n```")

	extractor := NewReliableExtractor[testUserProfile](caller, DefaultReliableExtractorConfig())

	result, err := extractor.Extract(context.Background(), "Carol")
	if err != nil {
		t.Fatalf("should extract from markdown: %v", err)
	}
	if result.Value.Name != "Carol" {
		t.Errorf("want Carol, got %s", result.Value.Name)
	}
}

func TestReliableExtractor_AllAttemptsFail(t *testing.T) {
	caller := mockLLMCaller("this is not json at all", "also not json", "still not json")

	config := DefaultReliableExtractorConfig()
	config.MaxRetries = 3
	extractor := NewReliableExtractor[testUserProfile](caller, config)

	_, err := extractor.Extract(context.Background(), "text without json")
	if err == nil {
		t.Fatal("expected error when all attempts fail")
	}
	t.Logf("expected error: %v", err)
}

func TestReliableExtractor_ContextCancel(t *testing.T) {
	caller := func(ctx context.Context, messages []types.Message) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Second):
			return `{"name":"X"}`, nil
		}
	}

	extractor := NewReliableExtractor[testUserProfile](caller, DefaultReliableExtractorConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := extractor.Extract(ctx, "test")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestReliableExtractor_CustomValidation(t *testing.T) {
	caller := mockLLMCaller(`{"name":"Dave","email":"not-an-email","age":-5}`)

	config := DefaultReliableExtractorConfig()
	config.MaxRetries = 1
	config.ValidateFunc = func(v any) error {
		profile := v.(*testUserProfile)
		if profile.Age < 0 {
			return fmt.Errorf("age must be non-negative, got %d", profile.Age)
		}
		return nil
	}
	extractor := NewReliableExtractor[testUserProfile](caller, config)

	_, err := extractor.Extract(context.Background(), "Dave")
	if err == nil {
		t.Fatal("expected validation error")
	}
	t.Logf("validation error: %v", err)
}

// ─── JSON 工具函数测试 ────────────────────────────────────────

func TestExtractJSONFromText_Plain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON bool
	}{
		{"plain json", `{"key":"val"}`, true},
		{"markdown json", "```json\n{\"key\":\"val\"}\n```", true},
		{"json in text", "Some text {\"key\":\"val\"} more text", true},
		{"no json", "this has no json", false},
		{"array", `["a","b","c"]`, true},
		{"json patch", `[{"op":"replace","path":"/a","value":"b"}]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONForTrustcall(tt.input)
			if tt.wantJSON && result == "" {
				t.Errorf("expected JSON to be extracted from: %s", tt.input)
			}
			if !tt.wantJSON && result != "" {
				t.Errorf("expected no JSON but got: %s", result)
			}
		})
	}
}

func TestApplyJSONPatches(t *testing.T) {
	original := `{"name":"Alice","age":0}`

	patches := `[{"op":"replace","path":"/name","value":"Bob"},{"op":"add","path":"/age","value":30}]`

	result, err := applyJSONPatches(original, patches)
	if err != nil {
		t.Fatalf("applyJSONPatches failed: %v", err)
	}

	if !strings.Contains(result, `"Bob"`) {
		t.Errorf("patch should replace name to Bob: %s", result)
	}
	t.Logf("patched result: %s", result)
}

func TestApplyJSONPatches_InvalidPatch_FallbackToNewJSON(t *testing.T) {
	original := `{"name":"Alice"}`
	newJSON := `{"name":"Bob","age":25}` // 不是 Patch 格式，当完整 JSON 处理

	result, _ := applyJSONPatches(original, newJSON)
	if result != newJSON {
		t.Errorf("non-patch JSON should be returned as-is: %s", result)
	}
}

func TestGenerateJSONSchema(t *testing.T) {
	type SampleStruct struct {
		Name  string  `json:"name"   validate:"required"`
		Score float64 `json:"score"`
		Tags  []string `json:"tags"`
		Active bool   `json:"active"`
	}

	schema := generateJSONSchema(SampleStruct{})

	if schema["type"] != "object" {
		t.Errorf("schema type should be object: %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema should have properties")
	}

	for _, field := range []string{"name", "score", "tags", "active"} {
		if _, ok := props[field]; !ok {
			t.Errorf("schema should have field %s", field)
		}
	}

	required, _ := schema["required"].([]string)
	found := false
	for _, r := range required {
		if r == "name" {
			found = true
		}
	}
	if !found {
		t.Error("'name' should be required (has validate:required tag)")
	}
}

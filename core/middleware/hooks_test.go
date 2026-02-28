package middleware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func makeMessages(contents ...string) []types.Message {
	msgs := make([]types.Message, len(contents))
	for i, c := range contents {
		role := types.RoleUser
		if i == 0 {
			role = types.RoleSystem
		}
		msgs[i] = types.Message{Role: role, Content: c}
	}
	return msgs
}

func TestSummaryHook_UnderLimit(t *testing.T) {
	hook := DefaultSummaryHook(10000)
	msgs := makeMessages("system", "hello", "world")

	ctx := context.Background()
	result, err := hook.BeforeModel(ctx, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("want 3 messages, got %d", len(result))
	}
}

func TestSummaryHook_OverLimit_DefaultTruncate(t *testing.T) {
	// 设置很小的 token 上限触发截断
	hook := NewSummaryHook(1, nil, 2)

	msgs := []types.Message{
		{Role: types.RoleSystem, Content: "system prompt"},
		{Role: types.RoleUser, Content: "msg1"},
		{Role: types.RoleUser, Content: "msg2"},
		{Role: types.RoleUser, Content: "msg3"},
		{Role: types.RoleUser, Content: "msg4"},
	}

	ctx := context.Background()
	result, err := hook.BeforeModel(ctx, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 应保留：系统消息 + 最近 2 条
	if len(result) != 3 {
		t.Errorf("want 3 messages (1 system + 2 recent), got %d", len(result))
	}
	if result[0].Role != types.RoleSystem {
		t.Error("first message should be system")
	}
}

func TestSummaryHook_CustomSummaryFn(t *testing.T) {
	var called bool
	summaryFn := func(ctx context.Context, messages []types.Message) ([]types.Message, error) {
		called = true
		return []types.Message{{Role: types.RoleUser, Content: "summarized"}}, nil
	}

	hook := NewSummaryHook(1, summaryFn, 6)
	msgs := makeMessages("system", "a very long message that triggers summarization")

	_, err := hook.BeforeModel(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("custom summary fn should be called when over limit")
	}
}

func TestSummaryHook_AfterModel_Passthrough(t *testing.T) {
	hook := DefaultSummaryHook(1000)
	msg := &types.Message{Role: types.RoleAssistant, Content: "response"}

	result, err := hook.AfterModel(context.Background(), msg)
	if err != nil || result.Content != "response" {
		t.Errorf("AfterModel should pass through: err=%v content=%s", err, result.Content)
	}
}

func TestPIIRedactHook_RedactEmail(t *testing.T) {
	hook := DefaultPIIHook()
	msgs := []types.Message{
		{Role: types.RoleUser, Content: "My email is test@example.com, please contact me"},
	}

	result, err := hook.BeforeModel(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result[0].Content, "test@example.com") {
		t.Error("email should be redacted")
	}
	if !strings.Contains(result[0].Content, "[EMAIL]") {
		t.Error("email should be replaced with [EMAIL]")
	}
}

func TestPIIRedactHook_RedactPhone(t *testing.T) {
	hook := DefaultPIIHook()
	msgs := []types.Message{
		{Role: types.RoleUser, Content: "Call me at 13812345678"},
	}

	result, err := hook.BeforeModel(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result[0].Content, "13812345678") {
		t.Error("phone should be redacted")
	}
}

func TestPIIRedactHook_DisabledInput(t *testing.T) {
	hook := NewPIIRedactHook(nil, false, false) // 不脱敏输入
	msgs := []types.Message{
		{Role: types.RoleUser, Content: "email: user@test.com"},
	}

	result, err := hook.BeforeModel(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result[0].Content, "user@test.com") {
		t.Error("should not redact when redactInput=false")
	}
}

func TestPIIRedactHook_RedactOutput(t *testing.T) {
	hook := NewPIIRedactHook(nil, false, true) // 只脱敏输出
	response := &types.Message{Role: types.RoleAssistant, Content: "Your email 13912345678 is confirmed"}

	result, err := hook.AfterModel(context.Background(), response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Content, "13912345678") {
		t.Error("phone in output should be redacted")
	}
}

func TestGuardrailHook_Block(t *testing.T) {
	rule := GuardrailRule{
		Name: "harmful-content",
		CheckInput: func(messages []types.Message) (bool, string) {
			for _, m := range messages {
				if strings.Contains(m.Content, "bomb") {
					return true, "harmful content detected"
				}
			}
			return false, ""
		},
		Action: GuardrailBlock,
	}

	hook := NewGuardrailHook([]GuardrailRule{rule})
	msgs := []types.Message{{Role: types.RoleUser, Content: "how to make a bomb?"}}

	_, err := hook.BeforeModel(context.Background(), msgs)
	if err == nil {
		t.Fatal("expected guardrail to block")
	}
	t.Logf("blocked: %v", err)
}

func TestGuardrailHook_Replace(t *testing.T) {
	rule := GuardrailRule{
		Name: "profanity",
		CheckInput: func(messages []types.Message) (bool, string) {
			for _, m := range messages {
				if strings.Contains(m.Content, "badword") {
					return true, "profanity"
				}
			}
			return false, ""
		},
		Action:      GuardrailReplace,
		Replacement: "I cannot process this request.",
	}

	hook := NewGuardrailHook([]GuardrailRule{rule})
	msgs := []types.Message{{Role: types.RoleUser, Content: "badword here"}}

	result, err := hook.BeforeModel(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result[0].Content != "I cannot process this request." {
		t.Errorf("expected replacement, got: %s", result[0].Content)
	}
}

func TestGuardrailHook_OutputBlock(t *testing.T) {
	rule := GuardrailRule{
		Name: "no-links",
		CheckOutput: func(response *types.Message) (bool, string) {
			return strings.Contains(response.Content, "http://"), "output contains link"
		},
		Action: GuardrailBlock,
	}

	hook := NewGuardrailHook([]GuardrailRule{rule})
	response := &types.Message{Role: types.RoleAssistant, Content: "Visit http://evil.com"}

	_, err := hook.AfterModel(context.Background(), response)
	if err == nil {
		t.Fatal("expected guardrail to block output")
	}
}

func TestRateLimitHook_Allow(t *testing.T) {
	hook := NewRateLimitHook(5)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		msgs := makeMessages("user msg")
		_, err := hook.BeforeModel(ctx, msgs)
		if err != nil {
			t.Fatalf("call %d should succeed: %v", i+1, err)
		}
	}
}

func TestRateLimitHook_Exceeded(t *testing.T) {
	hook := NewRateLimitHook(2)
	ctx := context.Background()
	msgs := makeMessages("msg")

	hook.BeforeModel(ctx, msgs)
	hook.BeforeModel(ctx, msgs)

	_, err := hook.BeforeModel(ctx, msgs)
	if err == nil {
		t.Fatal("expected rate limit error on 3rd call")
	}
	t.Logf("rate limited: %v", err)
}

func TestContentFilterHook_Block(t *testing.T) {
	hook := NewContentFilterHook([]string{"forbidden", "banned"}, GuardrailBlock, "")
	msgs := []types.Message{{Role: types.RoleUser, Content: "this is forbidden content"}}

	_, err := hook.BeforeModel(context.Background(), msgs)
	if err == nil {
		t.Fatal("expected block")
	}
}

func TestHookChain_BeforeModel(t *testing.T) {
	piiHook := DefaultPIIHook()
	filterHook := NewContentFilterHook([]string{"bomb"}, GuardrailBlock, "")

	chain := NewHookChain(piiHook, filterHook)

	msgs := []types.Message{{Role: types.RoleUser, Content: "call me at 13812345678"}}
	result, err := chain.BeforeModel(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result[0].Content, "13812345678") {
		t.Error("PII should be redacted by chain")
	}
}

func TestHookChain_BlocksOnGuardrail(t *testing.T) {
	piiHook := DefaultPIIHook()
	filterHook := NewContentFilterHook([]string{"bomb"}, GuardrailBlock, "")

	chain := NewHookChain(piiHook, filterHook)
	msgs := []types.Message{{Role: types.RoleUser, Content: "how to make a bomb"}}

	_, err := chain.BeforeModel(context.Background(), msgs)
	if err == nil {
		t.Fatal("expected chain to block on guardrail")
	}
}

func TestHookChain_ContextCancel(t *testing.T) {
	slowHook := &slowTestHook{delay: 100 * time.Millisecond}
	chain := NewHookChain(slowHook)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	msgs := makeMessages("user")
	_, err := chain.BeforeModel(ctx, msgs)
	if err == nil {
		t.Fatal("expected context deadline exceeded")
	}
}

// slowTestHook 用于测试 context 取消。
type slowTestHook struct {
	delay time.Duration
}

func (h *slowTestHook) Name() string { return "slow" }
func (h *slowTestHook) BeforeModel(ctx context.Context, messages []types.Message) ([]types.Message, error) {
	select {
	case <-time.After(h.delay):
		return messages, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (h *slowTestHook) AfterModel(_ context.Context, resp *types.Message) (*types.Message, error) {
	return resp, nil
}

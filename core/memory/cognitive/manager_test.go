package cognitive

import (
	"context"
	"strings"
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestMemoryManager_StoreAndSearchSemantic(t *testing.T) {
	mgr := NewMemoryManager(ManagerConfig{
		UserID:   "user-1",
		ThreadID: "thread-1",
	})

	ctx := context.Background()

	err := mgr.StoreSemanticMemory(ctx, &SemanticMemory{
		Content:    "用户偏好使用 Go 语言进行后端开发",
		Category:   "preference",
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("StoreSemanticMemory failed: %v", err)
	}

	err = mgr.StoreSemanticMemory(ctx, &SemanticMemory{
		Content:    "用户工作在金融行业",
		Category:   "fact",
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("StoreSemanticMemory failed: %v", err)
	}

	results, err := mgr.SearchSemanticMemory(ctx, "Go 编程", 5)
	if err != nil {
		t.Fatalf("SearchSemanticMemory failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one semantic memory result")
	}
	t.Logf("found %d semantic memories", len(results))
}

func TestMemoryManager_StoreAndSearchEpisodes(t *testing.T) {
	mgr := NewMemoryManager(ManagerConfig{
		UserID:   "user-2",
		ThreadID: "thread-2",
	})

	ctx := context.Background()

	err := mgr.StoreEpisode(ctx, &Episode{
		Summary: "用户询问了关于 Go 并发编程的问题，获得了满意解答",
		Quality: 0.9,
		Tags:    []string{"go", "concurrency"},
	})
	if err != nil {
		t.Fatalf("StoreEpisode failed: %v", err)
	}

	results, err := mgr.SearchEpisodes(ctx, "Go 并发", 5)
	if err != nil {
		t.Fatalf("SearchEpisodes failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one episode")
	}
}

func TestMemoryManager_ProceduralMemory(t *testing.T) {
	mgr := NewMemoryManager(ManagerConfig{UserID: "user-3"})
	ctx := context.Background()

	// 获取初始（空）程序性记忆
	pm, err := mgr.GetProceduralMemory(ctx)
	if err != nil {
		t.Fatalf("GetProceduralMemory failed: %v", err)
	}
	if pm != nil {
		t.Log("no procedural memory initially (expected nil or empty)")
	}

	// 负向反馈 -> 应提取行为规则
	err = mgr.UpdateProceduralMemory(ctx, &Feedback{
		ThreadID:      "thread-3",
		Score:         -0.8,
		Comment:       "回答不够准确",
		FailureReason: "没有验证用户输入的合法性",
	})
	if err != nil {
		t.Fatalf("UpdateProceduralMemory failed: %v", err)
	}

	pm, err = mgr.GetProceduralMemory(ctx)
	if err != nil {
		t.Fatalf("GetProceduralMemory after update failed: %v", err)
	}
	if pm == nil {
		t.Fatal("expected non-nil procedural memory after update")
	}
	if len(pm.BehaviorRules) == 0 {
		t.Error("expected behavior rules from negative feedback")
	}
	t.Logf("behavior rules: %v", pm.BehaviorRules)
}

func TestMemoryManager_Recall_AllLayers(t *testing.T) {
	mgr := NewMemoryManager(ManagerConfig{
		UserID:   "user-4",
		ThreadID: "thread-4",
	})
	ctx := context.Background()

	// 存储各层记忆
	_ = mgr.StoreSemanticMemory(ctx, &SemanticMemory{
		Content:    "用户是一名资深后端工程师",
		Category:   "fact",
		Confidence: 0.95,
	})
	_ = mgr.StoreEpisode(ctx, &Episode{
		Summary: "讨论了微服务架构设计方案",
		Quality: 0.8,
	})
	_ = mgr.UpdateProceduralMemory(ctx, &Feedback{
		Score:         -0.9,
		FailureReason: "忽略了系统的容错设计",
	})

	recalled, err := mgr.Recall(ctx, "架构设计", DefaultRecallOptions())
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	t.Logf("TotalRecalled: %d", recalled.TotalRecalled)
	t.Logf("SemanticFacts: %d", len(recalled.SemanticFacts))
	t.Logf("RecentEpisodes: %d", len(recalled.RecentEpisodes))
	t.Logf("ProceduralHints: %d", len(recalled.ProceduralHints))
	t.Logf("AugmentedContext:\n%s", recalled.AugmentedContext)

	if recalled.TotalRecalled == 0 {
		t.Error("expected at least some recalled memories")
	}
}

func TestMemoryManager_Recall_AugmentedContext(t *testing.T) {
	mgr := NewMemoryManager(ManagerConfig{UserID: "user-5"})
	ctx := context.Background()

	_ = mgr.StoreSemanticMemory(ctx, &SemanticMemory{
		Content:    "用户喜欢简洁的代码风格",
		Category:   "preference",
		Confidence: 0.9,
	})

	opts := DefaultRecallOptions()
	opts.IncludeEpisodic = false
	opts.IncludeProcedural = false

	recalled, err := mgr.Recall(ctx, "代码风格", opts)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if !strings.Contains(recalled.AugmentedContext, "用户喜欢简洁的代码风格") {
		t.Errorf("AugmentedContext should contain the semantic memory: %s", recalled.AugmentedContext)
	}
}

func TestMemoryManager_AutoConsolidate(t *testing.T) {
	mgr := NewMemoryManager(ManagerConfig{
		UserID:   "user-6",
		ThreadID: "thread-6",
	})
	ctx := context.Background()

	messages := []types.Message{
		{Role: types.RoleSystem, Content: "你是一个有用的助手"},
		{Role: types.RoleUser, Content: "我喜欢简洁的代码，请记住这一点"},
		{Role: types.RoleAssistant, Content: "好的，我会记住"},
		{Role: types.RoleUser, Content: "帮我写一个 HTTP 服务"},
		{Role: types.RoleAssistant, Content: "这里是代码..."},
	}

	err := mgr.AutoConsolidate(ctx, messages)
	if err != nil {
		t.Fatalf("AutoConsolidate failed: %v", err)
	}

	// 应该提取到偏好记忆
	results, err := mgr.SearchSemanticMemory(ctx, "喜欢", 5)
	if err != nil {
		t.Fatalf("search after consolidate: %v", err)
	}
	t.Logf("auto-consolidated %d semantic memories", len(results))
}

func TestRuleBasedExtractor_ExtractSemantic(t *testing.T) {
	extractor := NewRuleBasedExtractor()
	ctx := context.Background()

	messages := []types.Message{
		{Role: types.RoleUser, Content: "我喜欢使用 vim 编辑器"},
		{Role: types.RoleAssistant, Content: "好的"},
		{Role: types.RoleUser, Content: "请帮我查一下天气"},
	}

	mems, err := extractor.ExtractSemantic(ctx, messages)
	if err != nil {
		t.Fatalf("ExtractSemantic failed: %v", err)
	}
	if len(mems) == 0 {
		t.Error("expected to extract at least one semantic memory")
	}
	t.Logf("extracted: %v", mems[0].Content)
}

func TestRuleBasedExtractor_ExtractEpisode(t *testing.T) {
	extractor := NewRuleBasedExtractor()
	ctx := context.Background()

	messages := []types.Message{
		{Role: types.RoleUser, Content: "帮我写一个排序算法"},
		{Role: types.RoleAssistant, Content: "这里是快速排序..."},
	}

	ep, err := extractor.ExtractEpisode(ctx, messages, "thread-test")
	if err != nil {
		t.Fatalf("ExtractEpisode failed: %v", err)
	}
	if ep == nil {
		t.Fatal("expected non-nil episode")
	}
	if ep.Summary == "" {
		t.Error("expected non-empty summary")
	}
	t.Logf("episode summary: %s", ep.Summary)
}

func TestKnowledgeTriplet(t *testing.T) {
	triplet := KnowledgeTriplet{
		Subject:    "用户",
		Predicate:  "偏好",
		Object:     "Go 语言",
		Confidence: 0.9,
	}

	if triplet.Subject == "" || triplet.Predicate == "" || triplet.Object == "" {
		t.Error("triplet fields should not be empty")
	}
}

func TestRecallOptions_Default(t *testing.T) {
	opts := DefaultRecallOptions()
	if opts.K != 5 {
		t.Errorf("default K should be 5, got %d", opts.K)
	}
	if !opts.IncludeSemantic || !opts.IncludeEpisodic || !opts.IncludeProcedural {
		t.Error("default options should include all memory layers")
	}
}

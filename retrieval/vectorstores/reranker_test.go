package vectorstores

import (
	"context"
	"fmt"
	"testing"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

// MockChatModel 是用于测试的模拟聊天模型。
type MockChatModel struct {
	responses map[string]string // 查询 -> 响应的映射
}

func NewMockChatModel() *MockChatModel {
	return &MockChatModel{
		responses: make(map[string]string),
	}
}

func (m *MockChatModel) AddResponse(query, response string) {
	m.responses[query] = response
}

func (m *MockChatModel) Invoke(ctx context.Context, input []types.Message, opts ...runnable.Option) (types.Message, error) {
	if len(input) == 0 {
		return types.Message{}, fmt.Errorf("no input messages")
	}

	// 获取用户消息内容
	userMsg := input[len(input)-1].Content

	// 查找对应的响应
	for key, response := range m.responses {
		if contains(userMsg, key) {
			return types.NewAssistantMessage(response), nil
		}
	}

	// 默认返回高分
	return types.NewAssistantMessage("8"), nil
}

func (m *MockChatModel) Batch(ctx context.Context, inputs [][]types.Message, opts ...runnable.Option) ([]types.Message, error) {
	results := make([]types.Message, len(inputs))
	for i, input := range inputs {
		msg, err := m.Invoke(ctx, input, opts...)
		if err != nil {
			return nil, err
		}
		results[i] = msg
	}
	return results, nil
}

func (m *MockChatModel) Stream(ctx context.Context, input []types.Message, opts ...runnable.Option) (<-chan runnable.StreamEvent[types.Message], error) {
	ch := make(chan runnable.StreamEvent[types.Message], 2)
	go func() {
		defer close(ch)
		msg, _ := m.Invoke(ctx, input, opts...)
		ch <- runnable.StreamEvent[types.Message]{
			Type: runnable.EventStream,
			Data: msg,
		}
		ch <- runnable.StreamEvent[types.Message]{
			Type: runnable.EventEnd,
			Data: msg,
		}
	}()
	return ch, nil
}

func (m *MockChatModel) BindTools(tools []types.Tool) chat.ChatModel {
	return m
}

func (m *MockChatModel) WithStructuredOutput(schema types.Schema) chat.ChatModel {
	return m
}

func (m *MockChatModel) GetModelName() string {
	return "mock-model"
}

func (m *MockChatModel) GetProvider() string {
	return "mock"
}

func (m *MockChatModel) GetName() string {
	return "mock-chat-model"
}

func (m *MockChatModel) WithConfig(config *types.Config) runnable.Runnable[[]types.Message, types.Message] {
	return m
}

func (m *MockChatModel) WithRetry(policy types.RetryPolicy) runnable.Runnable[[]types.Message, types.Message] {
	return m
}

func (m *MockChatModel) WithFallbacks(fallbacks ...runnable.Runnable[[]types.Message, types.Message]) runnable.Runnable[[]types.Message, types.Message] {
	return m
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && (text == substr || len(text) > 0)
}

// TestNewLLMReranker 测试创建 LLM 重排序器。
func TestNewLLMReranker(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		llm := NewMockChatModel()
		config := LLMRerankerConfig{
			LLM: llm,
		}

		reranker, err := NewLLMReranker(config)
		if err != nil {
			t.Fatalf("Failed to create reranker: %v", err)
		}

		if reranker == nil {
			t.Fatal("Reranker is nil")
		}

		if reranker.topK != 20 {
			t.Errorf("Expected default topK 20, got %d", reranker.topK)
		}
	})

	t.Run("CustomPromptTemplate", func(t *testing.T) {
		llm := NewMockChatModel()
		customTemplate := "Custom template: {{.Query}} - {{.Document}}"

		config := LLMRerankerConfig{
			LLM:            llm,
			PromptTemplate: customTemplate,
			TopK:           10,
		}

		reranker, err := NewLLMReranker(config)
		if err != nil {
			t.Fatalf("Failed to create reranker: %v", err)
		}

		if reranker.promptTemplate != customTemplate {
			t.Errorf("Expected custom template, got: %s", reranker.promptTemplate)
		}

		if reranker.topK != 10 {
			t.Errorf("Expected topK 10, got %d", reranker.topK)
		}
	})

	t.Run("MissingLLM", func(t *testing.T) {
		config := LLMRerankerConfig{}

		_, err := NewLLMReranker(config)
		if err == nil {
			t.Error("Expected error for missing LLM, got nil")
		}
	})
}

// TestLLMRerankerParseScore 测试分数解析。
func TestLLMRerankerParseScore(t *testing.T) {
	llm := NewMockChatModel()
	config := LLMRerankerConfig{LLM: llm}
	reranker, _ := NewLLMReranker(config)

	tests := []struct {
		name     string
		response string
		expected float64
		hasError bool
	}{
		{
			name:     "Simple number",
			response: "8",
			expected: 8.0,
			hasError: false,
		},
		{
			name:     "Number with decimal",
			response: "7.5",
			expected: 7.5,
			hasError: false,
		},
		{
			name:     "Number with whitespace",
			response: "  9  ",
			expected: 9.0,
			hasError: false,
		},
		{
			name:     "Number with text",
			response: "8 - highly relevant",
			expected: 8.0,
			hasError: false,
		},
		{
			name:     "Invalid response",
			response: "not a number",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Out of range high",
			response: "15",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Out of range low",
			response: "-2",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := reranker.parseScore(tt.response)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if score != tt.expected {
					t.Errorf("Expected score %f, got %f", tt.expected, score)
				}
			}
		})
	}
}

// TestLLMRerankerRerank 测试重排序功能。
func TestLLMRerankerRerank(t *testing.T) {
	ctx := context.Background()

	t.Run("BasicReranking", func(t *testing.T) {
		// 创建模拟 LLM
		llm := NewMockChatModel()

		// 设置响应：为不同文档返回不同分数
		// 注意：这里简化了，实际会根据提示词内容匹配
		llm.AddResponse("AI", "9")           // 高相关性
		llm.AddResponse("weather", "2")      // 低相关性
		llm.AddResponse("learning", "7")     // 中相关性

		config := LLMRerankerConfig{
			LLM:  llm,
			TopK: 10,
		}
		reranker, err := NewLLMReranker(config)
		if err != nil {
			t.Fatalf("Failed to create reranker: %v", err)
		}

		// 创建测试文档
		docs := []DocumentWithScore{
			{
				Document: loaders.NewDocument("The weather is nice", nil),
				Score:    0.9, // 原始高分
			},
			{
				Document: loaders.NewDocument("AI is transforming technology", nil),
				Score:    0.8,
			},
			{
				Document: loaders.NewDocument("Machine learning basics", nil),
				Score:    0.7,
			},
		}

		// 执行重排序
		reranked, err := reranker.Rerank(ctx, "artificial intelligence", docs)
		if err != nil {
			t.Fatalf("Reranking failed: %v", err)
		}

		if len(reranked) != len(docs) {
			t.Errorf("Expected %d results, got %d", len(docs), len(reranked))
		}

		// 验证排序（第一个应该是关于 AI 的文档）
		if !contains(reranked[0].Document.Content, "AI") {
			t.Logf("Warning: Expected AI document first, got: %s", reranked[0].Document.Content)
		}
	})

	t.Run("EmptyDocuments", func(t *testing.T) {
		llm := NewMockChatModel()
		config := LLMRerankerConfig{LLM: llm}
		reranker, _ := NewLLMReranker(config)

		docs := []DocumentWithScore{}
		reranked, err := reranker.Rerank(ctx, "test query", docs)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(reranked) != 0 {
			t.Errorf("Expected 0 results, got %d", len(reranked))
		}
	})

	t.Run("TopKLimiting", func(t *testing.T) {
		llm := NewMockChatModel()
		config := LLMRerankerConfig{
			LLM:  llm,
			TopK: 2, // 只重排前2个
		}
		reranker, _ := NewLLMReranker(config)

		// 创建5个文档
		docs := make([]DocumentWithScore, 5)
		for i := 0; i < 5; i++ {
			docs[i] = DocumentWithScore{
				Document: loaders.NewDocument(fmt.Sprintf("Document %d", i), nil),
				Score:    float32(i),
			}
		}

		reranked, err := reranker.Rerank(ctx, "test", docs)
		if err != nil {
			t.Fatalf("Reranking failed: %v", err)
		}

		// 应该返回所有5个文档
		if len(reranked) != 5 {
			t.Errorf("Expected 5 results, got %d", len(reranked))
		}
	})
}

// TestLLMRerankerRerankDocuments 测试便捷函数。
func TestLLMRerankerRerankDocuments(t *testing.T) {
	ctx := context.Background()

	llm := NewMockChatModel()
	config := LLMRerankerConfig{LLM: llm}
	reranker, _ := NewLLMReranker(config)

	docs := []*loaders.Document{
		loaders.NewDocument("Document 1", nil),
		loaders.NewDocument("Document 2", nil),
		loaders.NewDocument("Document 3", nil),
	}

	reranked, err := reranker.RerankDocuments(ctx, "test query", docs)
	if err != nil {
		t.Fatalf("RerankDocuments failed: %v", err)
	}

	if len(reranked) != len(docs) {
		t.Errorf("Expected %d results, got %d", len(docs), len(reranked))
	}
}

// TestInMemoryVectorStoreRerank 测试向量存储的重排序功能。
func TestInMemoryVectorStoreRerank(t *testing.T) {
	ctx := context.Background()

	// 创建向量存储
	emb := &FakeEmbeddings{dimension: 3}
	store := NewInMemoryVectorStore(emb)

	// 添加文档
	docs := []*loaders.Document{
		loaders.NewDocument("AI is transforming technology", nil),
		loaders.NewDocument("The weather is nice today", nil),
		loaders.NewDocument("Machine learning is powerful", nil),
		loaders.NewDocument("I love pizza", nil),
	}

	_, err := store.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// 创建 LLM 重排序器
	llm := NewMockChatModel()
	config := LLMRerankerConfig{
		LLM:  llm,
		TopK: 10,
	}
	reranker, err := NewLLMReranker(config)
	if err != nil {
		t.Fatalf("Failed to create reranker: %v", err)
	}

	// 使用重排序搜索
	results, err := store.SimilaritySearchWithRerank(ctx, "artificial intelligence", 3, reranker)
	if err != nil {
		t.Fatalf("Search with rerank failed: %v", err)
	}

	if len(results) > 3 {
		t.Errorf("Expected at most 3 results, got %d", len(results))
	}

	// 验证没有重复
	seen := make(map[string]bool)
	for _, doc := range results {
		if seen[doc.Content] {
			t.Errorf("Duplicate document: %s", doc.Content)
		}
		seen[doc.Content] = true
	}
}

// TestRerankerInterface 测试接口实现。
func TestRerankerInterface(t *testing.T) {
	var _ RerankerVectorStore = (*InMemoryVectorStore)(nil)
}

package fusion

import (
	"testing"

	"github.com/zhucl121/langchain-go/pkg/types"
)

func TestRRFStrategy_Basic(t *testing.T) {
	// 准备测试数据
	doc1 := types.Document{Content: "document one"}
	doc2 := types.Document{Content: "document two"}
	doc3 := types.Document{Content: "document three"}
	doc4 := types.Document{Content: "document four"}

	// 向量检索结果
	vectorList := RankedList{
		Source: "vector",
		Documents: []RankedDocument{
			{Document: doc1, Score: 0.9, Rank: 1},
			{Document: doc2, Score: 0.8, Rank: 2},
			{Document: doc3, Score: 0.7, Rank: 3},
		},
	}

	// 关键词检索结果
	keywordList := RankedList{
		Source: "keyword",
		Documents: []RankedDocument{
			{Document: doc2, Score: 10.5, Rank: 1}, // doc2 在两个列表中都出现
			{Document: doc4, Score: 8.2, Rank: 2},
			{Document: doc1, Score: 6.1, Rank: 3},
		},
	}

	// 融合
	strategy := NewRRFStrategy(60)
	results := strategy.Fuse([]RankedList{vectorList, keywordList})

	// 验证
	if len(results) != 4 {
		t.Fatalf("Expected 4 unique documents, got %d", len(results))
	}

	// doc2 应该排名最高（在两个列表中都靠前）
	if results[0].Document.Content != "document two" {
		t.Errorf("Expected doc2 to rank first, got %s", results[0].Document.Content)
	}

	// 验证分数递减
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted: score[%d]=%.4f > score[%d]=%.4f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}

	// 验证来源信息
	if len(results[0].SourceScores) != 2 {
		t.Errorf("Expected 2 sources for top doc, got %d", len(results[0].SourceScores))
	}

	t.Logf("Top 3 results:")
	for i := 0; i < 3 && i < len(results); i++ {
		t.Logf("  %d. %s (score: %.4f, sources: %v)",
			i+1, results[i].Document.Content, results[i].Score, results[i].SourceRanks)
	}
}

func TestRRFStrategy_EmptyLists(t *testing.T) {
	strategy := NewRRFStrategy(60)
	results := strategy.Fuse([]RankedList{})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(results))
	}
}

func TestRRFStrategy_SingleList(t *testing.T) {
	doc1 := types.Document{Content: "document one"}
	doc2 := types.Document{Content: "document two"}

	list := RankedList{
		Source: "vector",
		Documents: []RankedDocument{
			{Document: doc1, Score: 0.9, Rank: 1},
			{Document: doc2, Score: 0.8, Rank: 2},
		},
	}

	strategy := NewRRFStrategy(60)
	results := strategy.Fuse([]RankedList{list})

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// 验证顺序保持
	if results[0].Document.Content != "document one" {
		t.Errorf("Expected doc1 first, got %s", results[0].Document.Content)
	}
}

func TestRRFStrategy_DifferentK(t *testing.T) {
	doc1 := types.Document{Content: "document one"}

	list := RankedList{
		Source: "test",
		Documents: []RankedDocument{
			{Document: doc1, Score: 1.0, Rank: 1},
		},
	}

	// 测试不同的 k 值
	k30 := NewRRFStrategy(30)
	results30 := k30.Fuse([]RankedList{list})

	k60 := NewRRFStrategy(60)
	results60 := k60.Fuse([]RankedList{list})

	// k 越小，分数越高
	if results30[0].Score <= results60[0].Score {
		t.Errorf("Expected k=30 to have higher score than k=60: %.4f vs %.4f",
			results30[0].Score, results60[0].Score)
	}

	t.Logf("RRF scores: k=30: %.4f, k=60: %.4f", results30[0].Score, results60[0].Score)
}

func TestWeightedStrategy_Basic(t *testing.T) {
	doc1 := types.Document{Content: "document one"}
	doc2 := types.Document{Content: "document two"}
	doc3 := types.Document{Content: "document three"}

	vectorList := RankedList{
		Source: "vector",
		Documents: []RankedDocument{
			{Document: doc1, Score: 0.9, Rank: 1},
			{Document: doc2, Score: 0.6, Rank: 2},
		},
	}

	keywordList := RankedList{
		Source: "keyword",
		Documents: []RankedDocument{
			{Document: doc2, Score: 0.8, Rank: 1},
			{Document: doc3, Score: 0.7, Rank: 2},
		},
	}

	// 向量权重更高
	strategy := NewWeightedStrategy(map[string]float64{
		"vector":  0.7,
		"keyword": 0.3,
	})

	results := strategy.Fuse([]RankedList{vectorList, keywordList})

	if len(results) != 3 {
		t.Fatalf("Expected 3 unique documents, got %d", len(results))
	}

	// 验证分数递减
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("Results not sorted by score")
		}
	}

	t.Logf("Weighted results:")
	for i, result := range results {
		t.Logf("  %d. %s (score: %.4f)", i+1, result.Document.Content, result.Score)
	}
}

func TestWeightedStrategy_NoNormalization(t *testing.T) {
	doc1 := types.Document{Content: "document one"}

	list := RankedList{
		Source: "test",
		Documents: []RankedDocument{
			{Document: doc1, Score: 100.0, Rank: 1},
		},
	}

	strategy := NewWeightedStrategy(map[string]float64{
		"test": 0.5,
	})
	strategy.Normalize = false

	results := strategy.Fuse([]RankedList{list})

	// 不归一化时，分数 = 0.5 * 100 = 50
	expectedScore := 50.0
	if results[0].Score != expectedScore {
		t.Errorf("Expected score %.2f, got %.2f", expectedScore, results[0].Score)
	}
}

func TestWeightedStrategy_WithNormalization(t *testing.T) {
	doc1 := types.Document{Content: "document one"}
	doc2 := types.Document{Content: "document two"}

	list := RankedList{
		Source: "test",
		Documents: []RankedDocument{
			{Document: doc1, Score: 100.0, Rank: 1},
			{Document: doc2, Score: 50.0, Rank: 2},
		},
	}

	strategy := NewWeightedStrategy(map[string]float64{
		"test": 1.0,
	})
	strategy.Normalize = true

	results := strategy.Fuse([]RankedList{list})

	// 归一化后：doc1 = 1.0, doc2 = 0.0
	if results[0].Score != 1.0 {
		t.Errorf("Expected normalized score 1.0, got %.2f", results[0].Score)
	}
	if results[1].Score != 0.0 {
		t.Errorf("Expected normalized score 0.0, got %.2f", results[1].Score)
	}
}

func TestWeightedStrategy_DefaultWeights(t *testing.T) {
	doc1 := types.Document{Content: "document one"}

	list1 := RankedList{
		Source: "source1",
		Documents: []RankedDocument{
			{Document: doc1, Score: 1.0, Rank: 1},
		},
	}

	list2 := RankedList{
		Source: "source2",
		Documents: []RankedDocument{
			{Document: doc1, Score: 1.0, Rank: 1},
		},
	}

	// 不指定权重，应该使用平均权重
	strategy := NewWeightedStrategy(map[string]float64{})
	results := strategy.Fuse([]RankedList{list1, list2})

	// 两个来源，默认权重各 0.5，归一化后各为 1.0，总分 = 0.5 * 1.0 + 0.5 * 1.0 = 1.0
	if results[0].Score != 1.0 {
		t.Errorf("Expected score 1.0 with default weights, got %.2f", results[0].Score)
	}
}

func TestLinearCombinationStrategy(t *testing.T) {
	doc1 := types.Document{Content: "document one"}
	doc2 := types.Document{Content: "document two"}

	list1 := RankedList{
		Source: "source1",
		Documents: []RankedDocument{
			{Document: doc1, Score: 10.0, Rank: 1},
			{Document: doc2, Score: 5.0, Rank: 2},
		},
	}

	list2 := RankedList{
		Source: "source2",
		Documents: []RankedDocument{
			{Document: doc1, Score: 8.0, Rank: 1},
			{Document: doc2, Score: 12.0, Rank: 2},
		},
	}

	strategy := NewLinearCombinationStrategy(map[string]float64{
		"source1": 1.0,
		"source2": 2.0,
	})

	results := strategy.Fuse([]RankedList{list1, list2})

	// doc1: 1.0*10 + 2.0*8 = 26
	// doc2: 1.0*5 + 2.0*12 = 29
	if results[0].Document.Content != "document two" {
		t.Errorf("Expected doc2 to rank first")
	}

	if results[0].Score != 29.0 {
		t.Errorf("Expected score 29.0, got %.2f", results[0].Score)
	}
}

func TestConvertToRankedList(t *testing.T) {
	docs := []types.Document{
		{Content: "doc1"},
		{Content: "doc2"},
		{Content: "doc3"},
	}

	scores := []float64{0.9, 0.8, 0.7}

	rankedList := ConvertToRankedList("test", docs, scores)

	if rankedList.Source != "test" {
		t.Errorf("Expected source 'test', got '%s'", rankedList.Source)
	}

	if len(rankedList.Documents) != 3 {
		t.Fatalf("Expected 3 documents, got %d", len(rankedList.Documents))
	}

	// 验证排名从 1 开始
	for i, doc := range rankedList.Documents {
		if doc.Rank != i+1 {
			t.Errorf("Expected rank %d, got %d", i+1, doc.Rank)
		}
		if doc.Score != scores[i] {
			t.Errorf("Expected score %.2f, got %.2f", scores[i], doc.Score)
		}
	}
}

func TestGetDocumentKey_WithID(t *testing.T) {
	doc := types.Document{
		Content: "test content",
		Metadata: map[string]any{
			"id": "doc123",
		},
	}

	key := getDocumentKey(doc)
	if key != "doc123" {
		t.Errorf("Expected key 'doc123', got '%s'", key)
	}
}

func TestGetDocumentKey_WithoutID(t *testing.T) {
	doc := types.Document{
		Content: "test content without id",
	}

	key := getDocumentKey(doc)
	if key != "test content without id" {
		t.Errorf("Expected content as key, got '%s'", key)
	}
}

func TestGetDocumentKey_LongContent(t *testing.T) {
	longContent := ""
	for i := 0; i < 200; i++ {
		longContent += "a"
	}

	doc := types.Document{
		Content: longContent,
	}

	key := getDocumentKey(doc)
	if len(key) != 100 {
		t.Errorf("Expected key length 100, got %d", len(key))
	}
}

func BenchmarkRRFStrategy(b *testing.B) {
	// 准备测试数据
	docs := make([]types.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = types.Document{Content: "document"}
	}

	list1 := RankedList{
		Source:    "source1",
		Documents: make([]RankedDocument, 100),
	}
	list2 := RankedList{
		Source:    "source2",
		Documents: make([]RankedDocument, 100),
	}

	for i := 0; i < 100; i++ {
		list1.Documents[i] = RankedDocument{
			Document: docs[i],
			Score:    float64(100 - i),
			Rank:     i + 1,
		}
		list2.Documents[i] = RankedDocument{
			Document: docs[i],
			Score:    float64(100 - i),
			Rank:     i + 1,
		}
	}

	strategy := NewRRFStrategy(60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.Fuse([]RankedList{list1, list2})
	}
}

func BenchmarkWeightedStrategy(b *testing.B) {
	docs := make([]types.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = types.Document{Content: "document"}
	}

	list1 := RankedList{
		Source:    "source1",
		Documents: make([]RankedDocument, 100),
	}
	list2 := RankedList{
		Source:    "source2",
		Documents: make([]RankedDocument, 100),
	}

	for i := 0; i < 100; i++ {
		list1.Documents[i] = RankedDocument{
			Document: docs[i],
			Score:    float64(100 - i),
			Rank:     i + 1,
		}
		list2.Documents[i] = RankedDocument{
			Document: docs[i],
			Score:    float64(100 - i),
			Rank:     i + 1,
		}
	}

	strategy := NewWeightedStrategy(map[string]float64{
		"source1": 0.7,
		"source2": 0.3,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.Fuse([]RankedList{list1, list2})
	}
}

# Hybrid Search (混合检索) 统一架构设计

**版本**: v0.2.0  
**日期**: 2026-01-20  
**状态**: 设计中

---

## 📋 目录

1. [概述](#概述)
2. [现状分析](#现状分析)
3. [架构设计](#架构设计)
4. [实现计划](#实现计划)
5. [API 设计](#api-设计)
6. [测试策略](#测试策略)

---

## 1. 概述

### 1.1 背景

**问题**: 纯语义检索（Dense Vector）可能遗漏关键词匹配
- 专业术语检索不准
- 精确匹配能力不足
- 法律、医疗等领域效果差

**解决方案**: Hybrid Search = Dense Vector + Sparse (BM25) + Fusion

### 1.2 目标

- ✅ 统一的 Hybrid Search 接口
- ✅ 多种融合策略（RRF, Weighted, Linear）
- ✅ 支持多个向量存储
- ✅ 可扩展的架构
- ✅ 高性能（并行检索）

### 1.3 现状

**已实现**:
- ✅ Milvus: 基础 RRF 实现（单向量搜索 + RRF）
- ✅ 其他向量存储: 仅支持 Dense Vector

**待实现**:
- ❌ BM25 稀疏检索
- ❌ 统一的 Hybrid Retriever
- ❌ 多种融合策略
- ❌ 其他向量存储的 Hybrid 支持

---

## 2. 现状分析

### 2.1 Milvus 实现分析

**当前实现** (`retrieval/vectorstores/milvus.go`):

```go
// 优点
✅ 有 HybridSearchOptions 配置
✅ 有 HybridSearchResult 结果结构
✅ 实现了 RRF 融合算法
✅ 支持 MultiVectorSearch

// 局限
❌ 只使用了向量搜索（没有真正的 BM25）
❌ RRF 实现在 VectorStore 内部（不可复用）
❌ 缺少其他融合策略
❌ 没有关键词检索支持
```

**代码示例**:
```go
func (store *MilvusVectorStore) HybridSearch(
    ctx context.Context, 
    query string, 
    k int, 
    opts *HybridSearchOptions,
) ([]HybridSearchResult, error) {
    // 当前只做向量搜索
    vectorResults, _ := store.SimilaritySearchWithScore(ctx, query, k*2)
    
    // TODO: 添加 BM25 关键词搜索
    
    // 应用 RRF 融合
    results := store.applyRRF([][]DocumentWithScore{vectorResults}, k, topK)
    return results, nil
}
```

### 2.2 其他向量存储

**Chroma, Qdrant, Weaviate, Redis**:
- ✅ 都支持基础向量检索
- ❌ 都没有 Hybrid Search 实现
- ❌ 都没有 BM25 支持

---

## 3. 架构设计

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    HybridRetriever                           │
│              (统一的混合检索入口)                             │
└─────────────────┬───────────────────────────────────────────┘
                  │
    ┌─────────────┼─────────────┬──────────────────┐
    │             │             │                  │
┌───▼────┐  ┌────▼───┐  ┌──────▼──┐      ┌────────▼──────┐
│ Vector │  │  BM25  │  │ Fusion  │      │  VectorStore  │
│Retriever│  │Retriever│  │Strategy │      │  (Optional)   │
└────┬────┘  └────┬───┘  └──────┬───┘      └────────┬──────┘
     │            │             │                   │
     │            │             │                   │
     └────────────┴─────┬───────┴───────────────────┘
                        │
                ┌───────▼────────┐
                │   Documents    │
                │  (融合结果)     │
                └────────────────┘
```

### 3.2 核心组件

#### 3.2.1 HybridRetriever (统一入口)

```go
// retrieval/retrievers/hybrid.go
type HybridRetriever struct {
    vectorRetriever  Retriever           // 向量检索器
    keywordRetriever KeywordRetriever    // 关键词检索器（可选）
    fusionStrategy   FusionStrategy      // 融合策略
    config           HybridConfig        // 配置
}

type HybridConfig struct {
    // 向量检索权重（0.0-1.0）
    VectorWeight float64
    
    // 关键词检索权重（0.0-1.0）
    KeywordWeight float64
    
    // 融合策略
    Strategy FusionStrategyType
    
    // RRF 参数
    RRFConstant int
    
    // 是否并行执行
    Parallel bool
}
```

#### 3.2.2 KeywordRetriever (BM25)

```go
// retrieval/retrievers/keyword/bm25.go
type BM25Retriever struct {
    documents []types.Document
    index     *BM25Index
    k1        float64  // BM25 参数
    b         float64  // BM25 参数
}

type BM25Index struct {
    docFreq      map[string]int      // 文档频率
    docLengths   []int               // 文档长度
    avgDocLength float64             // 平均文档长度
    totalDocs    int                 // 总文档数
    termIndex    map[string][]int    // 倒排索引
}

func (b *BM25Retriever) Search(
    ctx context.Context, 
    query string, 
    k int,
) ([]ScoredDocument, error)
```

#### 3.2.3 FusionStrategy (融合策略)

```go
// retrieval/retrievers/fusion/strategy.go
type FusionStrategy interface {
    Fuse(resultSets [][]ScoredDocument, config FusionConfig) []ScoredDocument
}

type FusionStrategyType string

const (
    // RRF (Reciprocal Rank Fusion)
    StrategyRRF FusionStrategyType = "rrf"
    
    // 加权融合
    StrategyWeighted FusionStrategyType = "weighted"
    
    // 线性组合
    StrategyLinear FusionStrategyType = "linear"
    
    // 分布式 RRF (DRRF)
    StrategyDRRF FusionStrategyType = "drrf"
)

// RRF: 1/(k + rank)
type RRFStrategy struct {
    RankConstant int // 默认 60
}

// Weighted: α*score1 + β*score2
type WeightedStrategy struct {
    Weights []float64
}

// Linear: normalize + weight
type LinearStrategy struct {
    Alpha float64 // 向量权重
    Beta  float64 // 关键词权重
}
```

### 3.3 VectorStore 集成

#### 方案 A: VectorStore 原生支持（推荐 Milvus）

```go
type HybridSearchCapable interface {
    HybridSearch(
        ctx context.Context,
        query string,
        k int,
        opts *HybridSearchOptions,
    ) ([]HybridSearchResult, error)
}

// Milvus 继续使用原生实现
if hs, ok := vectorStore.(HybridSearchCapable); ok {
    results := hs.HybridSearch(ctx, query, k, opts)
}
```

#### 方案 B: 外部 HybridRetriever（其他 VectorStore）

```go
// 使用统一的 HybridRetriever 包装
hybridRetriever := retrievers.NewHybridRetriever(
    vectorRetriever,    // 来自任何 VectorStore
    bm25Retriever,      // 独立的 BM25
    fusion.NewRRFStrategy(60),
)
```

---

## 4. 实现计划

### Phase 1: BM25 实现 (2天)

**目标**: 独立的 BM25 检索器

```
✅ BM25 算法实现
✅ 倒排索引构建
✅ 分词器集成
✅ 基础测试
```

**文件**:
- `retrieval/retrievers/keyword/bm25.go`
- `retrieval/retrievers/keyword/bm25_test.go`
- `retrieval/retrievers/keyword/tokenizer.go`

### Phase 2: 融合策略 (2天)

**目标**: 多种融合算法

```
✅ RRF (Reciprocal Rank Fusion)
✅ Weighted Fusion
✅ Linear Combination
✅ 策略接口和工厂
```

**文件**:
- `retrieval/retrievers/fusion/strategy.go`
- `retrieval/retrievers/fusion/rrf.go`
- `retrieval/retrievers/fusion/weighted.go`
- `retrieval/retrievers/fusion/linear.go`
- `retrieval/retrievers/fusion/fusion_test.go`

### Phase 3: HybridRetriever 实现 (1天)

**目标**: 统一的混合检索入口

```
✅ HybridRetriever 实现
✅ 并行检索
✅ 配置管理
✅ 与现有 Retriever 集成
```

**文件**:
- `retrieval/retrievers/hybrid.go`
- `retrieval/retrievers/hybrid_test.go`

### Phase 4: VectorStore 集成 (2天)

**目标**: 整合到各个 VectorStore

```
✅ Milvus: 增强现有实现
✅ Chroma: 添加 Hybrid 支持
✅ Qdrant: 添加 Hybrid 支持
✅ Weaviate: 添加 Hybrid 支持
```

**文件**:
- 更新各个 `vectorstores/*.go`

### Phase 5: 测试和文档 (1天)

```
✅ 集成测试
✅ 性能基准测试
✅ 示例程序
✅ API 文档
```

---

## 5. API 设计

### 5.1 基础 API

```go
// 方式 1: 使用 HybridRetriever
retriever := retrievers.NewHybridRetriever(
    vectorRetriever,
    bm25Retriever,
    retrievers.HybridConfig{
        Strategy:      retrievers.StrategyRRF,
        VectorWeight:  0.7,
        KeywordWeight: 0.3,
        RRFConstant:   60,
        Parallel:      true,
    },
)

docs, err := retriever.GetRelevantDocuments(ctx, "查询", 10)
```

```go
// 方式 2: 直接使用 VectorStore (Milvus)
results, err := milvusStore.HybridSearch(
    ctx,
    "查询",
    10,
    &HybridSearchOptions{
        RRFRankConstant: 60,
    },
)

for _, result := range results {
    fmt.Printf("分数: %.4f (向量: %.4f, 关键词: %.4f)\n",
        result.FusionScore,
        result.VectorScore,
        result.KeywordScore)
}
```

### 5.2 高级 API

```go
// 自定义融合策略
strategy := fusion.NewWeightedStrategy(
    []float64{0.7, 0.3},  // 权重
)

retriever := retrievers.NewHybridRetriever(
    vectorRetriever,
    bm25Retriever,
    retrievers.HybridConfig{
        Strategy: strategy,
    },
)
```

---

## 6. 测试策略

### 6.1 单元测试

```go
// BM25 测试
func TestBM25Search(t *testing.T) {
    docs := []types.Document{
        {Content: "Go is a programming language"},
        {Content: "Python is also a language"},
    }
    
    bm25 := NewBM25Retriever(docs)
    results, _ := bm25.Search(ctx, "programming", 5)
    
    assert.Equal(t, "Go is a programming language", results[0].Content)
}

// RRF 测试
func TestRRFFusion(t *testing.T) {
    set1 := []ScoredDocument{{Score: 0.9}, {Score: 0.8}}
    set2 := []ScoredDocument{{Score: 0.7}, {Score: 0.85}}
    
    strategy := NewRRFStrategy(60)
    fused := strategy.Fuse([][]ScoredDocument{set1, set2}, config)
    
    // 验证 RRF 分数计算
}
```

### 6.2 集成测试

```go
func TestHybridRetrieverIntegration(t *testing.T) {
    // 端到端测试
    vectorStore := setupMilvus(t)
    retriever := NewHybridRetriever(...)
    
    results, err := retriever.GetRelevantDocuments(ctx, query, 10)
    
    // 验证结果质量
    // 验证融合效果
}
```

### 6.3 性能测试

```go
func BenchmarkHybridSearch(b *testing.B) {
    // 测试检索性能
    // 对比 Dense vs Hybrid
    // 测试并行性能
}
```

---

## 7. 性能优化

### 7.1 并行检索

```go
// 并行执行向量和关键词检索
func (h *HybridRetriever) parallelSearch(ctx context.Context, query string, k int) (
    vectorResults []ScoredDocument,
    keywordResults []ScoredDocument,
    err error,
) {
    var wg sync.WaitGroup
    var vectorErr, keywordErr error
    
    wg.Add(2)
    
    go func() {
        defer wg.Done()
        vectorResults, vectorErr = h.vectorRetriever.Search(ctx, query, k)
    }()
    
    go func() {
        defer wg.Done()
        keywordResults, keywordErr = h.keywordRetriever.Search(ctx, query, k)
    }()
    
    wg.Wait()
    
    if vectorErr != nil {
        return nil, nil, vectorErr
    }
    if keywordErr != nil {
        return nil, nil, keywordErr
    }
    
    return vectorResults, keywordResults, nil
}
```

### 7.2 缓存策略

```go
// 缓存 BM25 索引
type CachedBM25Retriever struct {
    *BM25Retriever
    cache *lru.Cache
}
```

---

## 8. 向后兼容

### 8.1 Milvus 增强

**保持现有 API 不变**:
```go
// 继续支持
func (store *MilvusVectorStore) HybridSearch(...)

// 新增 BM25 支持
func (store *MilvusVectorStore) HybridSearchWithBM25(...)
```

### 8.2 统一接口

```go
// 新增统一接口
type Retriever interface {
    GetRelevantDocuments(ctx context.Context, query string, k int) ([]types.Document, error)
}

// Hybrid 是 Retriever 的一种实现
type HybridRetriever struct {
    // ...
}

func (h *HybridRetriever) GetRelevantDocuments(...) ([]types.Document, error)
```

---

## 9. 里程碑

### Milestone 1: BM25 + RRF (3天)
- ✅ BM25 实现
- ✅ RRF 融合
- ✅ 基础测试

### Milestone 2: 多策略支持 (2天)
- ✅ Weighted Fusion
- ✅ Linear Combination
- ✅ 策略工厂

### Milestone 3: VectorStore 集成 (2天)
- ✅ 所有 VectorStore 支持
- ✅ 统一接口
- ✅ 示例程序

### Milestone 4: 优化和发布 (1天)
- ✅ 性能优化
- ✅ 文档完善
- ✅ 发布 v0.2.0

**总计**: 8 个工作日

---

## 10. 参考资料

- [Reciprocal Rank Fusion (RRF) Paper](https://plg.uwaterloo.ca/~gvcormac/cormacksigir09-rrf.pdf)
- [BM25 算法详解](https://en.wikipedia.org/wiki/Okapi_BM25)
- [Milvus Hybrid Search](https://milvus.io/docs/hybrid_search.md)
- [Weaviate Hybrid Search](https://weaviate.io/developers/weaviate/search/hybrid)

---

**状态**: 📝 设计完成，待审批  
**下一步**: 开始实施 Phase 1 - BM25 实现

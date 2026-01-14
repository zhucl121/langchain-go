# Milvus Vector Store 使用指南

## 概述

Milvus 是一个开源的高性能向量数据库，专为大规模向量检索而设计。本指南介绍如何在 LangChain-Go 中使用 Milvus 作为向量存储。

## 安装 Milvus

### 使用 Docker (推荐)

```bash
# 下载 docker-compose 配置
wget https://github.com/milvus-io/milvus/releases/download/v2.3.0/milvus-standalone-docker-compose.yml -O docker-compose.yml

# 启动 Milvus
docker-compose up -d

# 检查状态
docker-compose ps
```

### 使用 Docker 单容器（快速开始）

```bash
docker run -d \
  --name milvus \
  -p 19530:19530 \
  -p 9091:9091 \
  milvusdb/milvus:latest
```

## 安装 Go SDK

```bash
go get github.com/milvus-io/milvus-sdk-go/v2
```

## 基础使用

### 创建 Milvus 向量存储

```go
package main

import (
    "context"
    "fmt"
    
    "langchain-go/retrieval/embeddings"
    "langchain-go/retrieval/vectorstores"
)

func main() {
    // 创建嵌入模型
    emb := embeddings.NewOpenAIEmbeddings(
        embeddings.OpenAIEmbeddingsConfig{
            APIKey: "sk-...",
            Model:  "text-embedding-3-small",
        },
    )
    
    // 配置 Milvus
    config := vectorstores.MilvusConfig{
        Address:              "localhost:19530",
        CollectionName:       "my_documents",
        Dimension:            1536, // text-embedding-3-small 维度
        AutoCreateCollection: true,
    }
    
    // 创建向量存储
    store, err := vectorstores.NewMilvusVectorStore(config, emb)
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    fmt.Println("Milvus 向量存储创建成功！")
}
```

### 添加文档

```go
import "langchain-go/retrieval/loaders"

func addDocuments(store *vectorstores.MilvusVectorStore) {
    ctx := context.Background()
    
    docs := []*loaders.Document{
        loaders.NewDocument(
            "Milvus is a vector database for AI applications",
            map[string]any{"source": "intro.md", "page": 1},
        ),
        loaders.NewDocument(
            "LangChain provides tools for building LLM applications",
            map[string]any{"source": "langchain.md", "page": 1},
        ),
        loaders.NewDocument(
            "Vector search enables semantic similarity matching",
            map[string]any{"source": "concepts.md", "page": 3},
        ),
    }
    
    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("添加了 %d 个文档，IDs: %v\n", len(ids), ids)
}
```

### 相似度搜索

```go
func searchDocuments(store *vectorstores.MilvusVectorStore) {
    ctx := context.Background()
    
    // 基础搜索
    results, err := store.SimilaritySearch(ctx, "vector database", 3)
    if err != nil {
        panic(err)
    }
    
    fmt.Println("搜索结果:")
    for i, doc := range results {
        fmt.Printf("%d. %s\n", i+1, doc.Content)
    }
}
```

### 带分数搜索

```go
func searchWithScore(store *vectorstores.MilvusVectorStore) {
    ctx := context.Background()
    
    results, err := store.SimilaritySearchWithScore(ctx, "AI and machine learning", 5)
    if err != nil {
        panic(err)
    }
    
    fmt.Println("带分数的搜索结果:")
    for i, result := range results {
        fmt.Printf("%d. [%.4f] %s\n", 
            i+1, 
            result.Score, 
            result.Document.Content[:50],
        )
    }
}
```

### 删除文档

```go
func deleteDocuments(store *vectorstores.MilvusVectorStore, ids []string) {
    ctx := context.Background()
    
    err := store.Delete(ctx, ids)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("删除了 %d 个文档\n", len(ids))
}
```

## 高级配置

### 自定义字段名

```go
config := vectorstores.MilvusConfig{
    Address:        "localhost:19530",
    CollectionName: "custom_collection",
    Dimension:      1536,
    
    // 自定义字段名
    IDField:       "doc_id",
    VectorField:   "embedding",
    ContentField:  "text",
    MetadataField: "meta",
    
    AutoCreateCollection: true,
}
```

### 配置索引参数

```go
import "github.com/milvus-io/milvus-sdk-go/v2/entity"

config := vectorstores.MilvusConfig{
    Address:        "localhost:19530",
    CollectionName: "optimized_collection",
    Dimension:      1536,
    
    // 索引配置
    IndexType:  entity.HNSW,    // HNSW 索引
    MetricType: entity.L2,      // L2 距离
    IndexParams: map[string]string{
        "M":              "32",   // HNSW M 参数
        "efConstruction": "512",  // 构建时的 ef
    },
    
    AutoCreateCollection: true,
}
```

### 使用不同的距离度量

```go
// 余弦相似度
config := vectorstores.MilvusConfig{
    MetricType: entity.COSINE,
    // ... 其他配置
}

// 内积
config := vectorstores.MilvusConfig{
    MetricType: entity.IP,
    // ... 其他配置
}

// L2 距离（默认）
config := vectorstores.MilvusConfig{
    MetricType: entity.L2,
    // ... 其他配置
}
```

## 完整的 RAG 示例

```go
package main

import (
    "context"
    "fmt"
    
    "langchain-go/core/chat/providers/openai"
    "langchain-go/pkg/types"
    "langchain-go/retrieval/embeddings"
    "langchain-go/retrieval/loaders"
    "langchain-go/retrieval/splitters"
    "langchain-go/retrieval/vectorstores"
)

func ragExample() {
    ctx := context.Background()
    
    // 1. 加载文档
    loader := loaders.NewDirectoryLoader("./knowledge_base").
        WithGlob("*.md").
        WithRecursive(true)
    docs, _ := loader.Load(ctx)
    fmt.Printf("加载了 %d 个文档\n", len(docs))
    
    // 2. 分割文档
    splitter := splitters.NewRecursiveCharacterTextSplitter(1000, 200)
    chunks := splitter.SplitDocuments(docs)
    fmt.Printf("分割成 %d 个块\n", len(chunks))
    
    // 3. 创建嵌入和 Milvus 存储
    emb := embeddings.NewOpenAIEmbeddings(
        embeddings.OpenAIEmbeddingsConfig{
            APIKey: "sk-...",
            Model:  "text-embedding-3-small",
        },
    )
    
    config := vectorstores.MilvusConfig{
        Address:              "localhost:19530",
        CollectionName:       "knowledge_base",
        Dimension:            1536,
        AutoCreateCollection: true,
    }
    
    store, _ := vectorstores.NewMilvusVectorStore(config, emb)
    defer store.Close()
    
    // 4. 存储文档向量
    ids, _ := store.AddDocuments(ctx, chunks)
    fmt.Printf("存储了 %d 个向量\n", len(ids))
    
    // 5. 语义搜索
    query := "什么是 Milvus 向量数据库？"
    results, _ := store.SimilaritySearchWithScore(ctx, query, 3)
    
    // 6. 构建上下文
    var context string
    fmt.Println("\n相关文档:")
    for i, result := range results {
        fmt.Printf("%d. [%.3f] %s\n", i+1, result.Score, result.Document.Content[:100])
        context += result.Document.Content + "\n\n"
    }
    
    // 7. 调用 LLM 生成答案
    chatModel := openai.New(openai.Config{APIKey: "sk-..."})
    
    prompt := fmt.Sprintf(`基于以下上下文回答问题:

上下文:
%s

问题: %s

答案:`, context, query)
    
    response, _ := chatModel.Invoke(ctx, []types.Message{
        types.NewUserMessage(prompt),
    })
    
    fmt.Println("\nAI 答案:", response.Content)
}

func main() {
    ragExample()
}
```

## 性能优化建议

### 1. 批量插入

```go
// 分批插入大量文档
func batchInsert(store *vectorstores.MilvusVectorStore, docs []*loaders.Document) {
    ctx := context.Background()
    batchSize := 100
    
    for i := 0; i < len(docs); i += batchSize {
        end := i + batchSize
        if end > len(docs) {
            end = len(docs)
        }
        
        batch := docs[i:end]
        _, err := store.AddDocuments(ctx, batch)
        if err != nil {
            fmt.Printf("批次 %d 失败: %v\n", i/batchSize, err)
        }
        
        fmt.Printf("完成批次 %d/%d\n", i/batchSize+1, (len(docs)+batchSize-1)/batchSize)
    }
}
```

### 2. 索引优化

```go
// 对于大规模数据，调整索引参数
config := vectorstores.MilvusConfig{
    IndexType:  entity.HNSW,
    MetricType: entity.L2,
    IndexParams: map[string]string{
        "M":              "64",   // 更大的 M 提升准确率
        "efConstruction": "512",  // 更大的 ef 提升构建质量
    },
}
```

### 3. 搜索优化

搜索时的 `ef` 参数（在 SDK 内部配置）影响搜索质量和速度：
- 较小的 ef (如 32): 更快但可能不够准确
- 较大的 ef (如 512): 更准确但较慢

## 监控和维护

### 获取集合统计

```go
func getStats(store *vectorstores.MilvusVectorStore) {
    ctx := context.Background()
    
    count, err := store.GetDocumentCount(ctx)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("集合中有 %d 个文档\n", count)
}
```

### 删除集合

```go
func dropCollection(store *vectorstores.MilvusVectorStore) {
    ctx := context.Background()
    
    err := store.DropCollection(ctx)
    if err != nil {
        panic(err)
    }
    
    fmt.Println("集合已删除")
}
```

## 测试

运行测试（需要 Milvus 实例）:

```bash
# 完整测试
go test ./retrieval/vectorstores -v -run TestMilvus

# 跳过集成测试
go test ./retrieval/vectorstores -v -short

# 基准测试
go test ./retrieval/vectorstores -bench=BenchmarkMilvus -run=^$
```

## 故障排查

### 连接失败

```go
// 添加超时和重试
config := vectorstores.MilvusConfig{
    Address: "localhost:19530",
    // ... 其他配置
}

// SDK 内部会处理连接重试
```

### 维度不匹配

```go
// 确保配置的维度与嵌入模型匹配
embDim := emb.GetDimension()
config := vectorstores.MilvusConfig{
    Dimension: embDim, // 使用嵌入模型的维度
}
```

## 最佳实践

1. **集合命名**: 使用有意义的集合名，避免特殊字符
2. **批量操作**: 大规模数据使用批量插入
3. **索引选择**: HNSW 适合大多数场景，IVF 适合超大规模
4. **定期备份**: Milvus 支持数据备份和恢复
5. **监控资源**: 监控 Milvus 的内存和 CPU 使用

## 参考资源

- [Milvus 官方文档](https://milvus.io/docs)
- [Milvus Go SDK](https://github.com/milvus-io/milvus-sdk-go)
- [HNSW 算法](https://arxiv.org/abs/1603.09320)

---

**最后更新**: 2026-01-14

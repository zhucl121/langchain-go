# 标准内容块 (Content Block)

## 📖 概述

标准内容块（Content Block）是 LangChain-Go v0.1.2+ 引入的统一输出格式，对标 LangChain v1.0+ 的标准化内容块设计。

### 核心特性

- ✅ **推理追踪** (Reasoning Trace) - 记录 AI 的思考过程
- ✅ **引用来源** (Citations) - 支持 RAG 系统的来源追溯
- ✅ **工具调用** (Tool Calls) - 标准化工具调用格式
- ✅ **元数据** (Metadata) - 丰富的附加信息
- ✅ **置信度** (Confidence) - 输出的可信度评分
- ✅ **类型安全** - 强类型定义，编译时检查
- ✅ **JSON 序列化** - 完全支持 JSON 序列化/反序列化

---

## 🎯 为什么需要标准内容块？

### 问题

传统的 LLM 输出格式不统一：
- 无法追溯推理过程
- 缺少引用来源信息
- 输出格式各异，难以解析
- 缺乏标准化的元数据

### 解决方案

标准内容块提供统一的输出格式：

```go
block := types.NewTextContentBlock("机器学习是...").
    WithReasoning([]string{
        "步骤1: 分析问题",
        "步骤2: 查找相关文档", 
        "步骤3: 综合得出答案",
    }).
    AddCitation(types.Citation{
        Source: "ml_textbook.pdf",
        Excerpt: "机器学习定义...",
        Score: 0.95,
    }).
    WithConfidence(0.92)
```

---

## 📦 核心类型

### ContentBlock

主要的内容块结构：

```go
type ContentBlock struct {
    Type       ContentBlockType   // 内容块类型
    Content    string             // 主要内容
    Reasoning  []string           // 推理步骤
    Citations  []Citation         // 引用来源
    ToolCalls  []ToolCall         // 工具调用
    Metadata   map[string]any     // 元数据
    Timestamp  time.Time          // 创建时间
    ID         string             // 唯一标识
    ParentID   string             // 父内容块 ID
    Confidence *float64           // 置信度 (0-1)
    Error      *ErrorInfo         // 错误信息
}
```

### ContentBlockType

支持的内容块类型：

| 类型 | 说明 | 使用场景 |
|------|------|----------|
| `ContentBlockText` | 文本内容 | 普通文本响应 |
| `ContentBlockThinking` | 思考过程 | o1 等模型的推理 |
| `ContentBlockToolUse` | 工具调用 | Agent 工具使用 |
| `ContentBlockToolResult` | 工具结果 | 工具执行结果 |
| `ContentBlockImage` | 图像内容 | 多模态输出 |
| `ContentBlockError` | 错误信息 | 错误处理 |

### Citation

引用来源结构（用于 RAG）：

```go
type Citation struct {
    Source    string             // 来源标识
    Excerpt   string             // 引用片段
    Score     float64            // 相似度分数
    Page      *int               // 页码
    StartChar *int               // 起始字符位置
    EndChar   *int               // 结束字符位置
    Title     string             // 来源标题
    URL       string             // 来源 URL
    Metadata  map[string]any     // 附加元数据
}
```

### ErrorInfo

错误信息结构：

```go
type ErrorInfo struct {
    Code        string           // 错误码
    Message     string           // 错误消息
    Details     map[string]any   // 详细信息
    Recoverable bool             // 是否可恢复
}
```

---

## 🚀 使用示例

### 1. 基础文本内容块

```go
block := types.NewTextContentBlock("这是答案").
    WithID("block_1").
    WithConfidence(0.95).
    WithMetadata("model", "gpt-4")

fmt.Println(block.Content) // "这是答案"
```

### 2. 带推理过程

```go
block := types.NewTextContentBlock("答案是 42").
    WithReasoning([]string{
        "步骤1: 理解问题",
        "步骤2: 分析数据",
        "步骤3: 得出结论",
    })

for _, step := range block.Reasoning {
    fmt.Println(step)
}
```

### 3. 带引用来源 (RAG)

```go
page := 15
block := types.NewTextContentBlock("机器学习是...").
    AddCitation(types.Citation{
        Source:  "ml_book.pdf",
        Excerpt: "机器学习定义...",
        Score:   0.95,
        Page:    &page,
        Title:   "机器学习导论",
    })

for _, citation := range block.Citations {
    fmt.Printf("来源: %s (相似度: %.2f)\n", 
        citation.Source, citation.Score)
}
```

### 4. 链式调用

```go
block := types.NewTextContentBlock("答案").
    WithReasoning([]string{"步骤1", "步骤2"}).
    AddCitation(types.Citation{Source: "doc.pdf", Score: 0.9}).
    WithConfidence(0.92).
    WithMetadata("tokens", 150)
```

### 5. 完整 RAG 场景

```go
list := types.NewContentBlockList()

// 1. 思考过程
list.Add(types.NewThinkingContentBlock("需要查找相关文档"))

// 2. 工具调用
list.Add(types.NewToolUseContentBlock([]types.ToolCall{
    {
        ID: "call_1",
        Type: "function",
        Function: types.FunctionCall{
            Name: "vector_search",
            Arguments: `{"query": "ML"}`,
        },
    },
}))

// 3. 工具结果
list.Add(types.NewToolResultContentBlock("找到3个文档"))

// 4. 最终答案（带引用）
list.Add(types.NewTextContentBlock("机器学习是...").
    WithReasoning([]string{"分析", "搜索", "综合"}).
    AddCitation(types.Citation{Source: "doc.pdf", Score: 0.95}).
    WithConfidence(0.92))

// 提取文本内容
text := list.GetTextContent()

// 提取所有引用
citations := list.GetAllCitations()
```

### 6. JSON 序列化

```go
block := types.NewTextContentBlock("测试").
    WithReasoning([]string{"步骤1"}).
    WithConfidence(0.9)

// 转换为 JSON
jsonStr, _ := block.ToJSON()
fmt.Println(jsonStr)

// 从 JSON 解析
var parsed types.ContentBlock
_ = parsed.FromJSON(jsonStr)
```

### 7. 错误处理

```go
errorBlock := types.NewErrorContentBlock(
    "RATE_LIMIT",
    "API 调用频率超限",
).WithMetadata("retry_after", 60)

errorBlock.Error.Details = map[string]any{
    "current_rate": 150,
    "max_rate": 100,
}
errorBlock.Error.Recoverable = true

if errorBlock.Error.Recoverable {
    fmt.Println("错误可恢复，稍后重试")
}
```

---

## 📋 ContentBlockList 操作

### 创建和添加

```go
list := types.NewContentBlockList()

list.Add(types.NewTextContentBlock("内容1").WithID("block1"))
list.Add(types.NewTextContentBlock("内容2").WithID("block2"))
```

### 按类型过滤

```go
textBlocks := list.GetByType(types.ContentBlockText)
thinkingBlocks := list.GetByType(types.ContentBlockThinking)
```

### 按 ID 查找

```go
block := list.GetByID("block1")
if block != nil {
    fmt.Println(block.Content)
}
```

### 提取文本内容

```go
// 拼接所有文本和思考块的内容
text := list.GetTextContent()
```

### 提取所有引用

```go
citations := list.GetAllCitations()
for _, citation := range citations {
    fmt.Printf("%s: %.2f\n", citation.Source, citation.Score)
}
```

### JSON 序列化

```go
// 序列化整个列表
jsonStr, _ := list.ToJSON()

// 反序列化
var parsed types.ContentBlockList
_ = parsed.FromJSON(jsonStr)
```

---

## 🎨 设计模式

### 1. 构建器模式（链式调用）

```go
block := types.NewTextContentBlock("内容").
    WithID("block1").
    WithReasoning([]string{"步骤1"}).
    AddCitation(types.Citation{...}).
    WithConfidence(0.9).
    WithMetadata("key", "value")
```

### 2. 不可变性

所有 `With*` 方法返回 `*ContentBlock`（指针），支持链式调用的同时保持简洁。

### 3. 类型安全

使用强类型枚举和结构体，避免运行时错误：

```go
const (
    ContentBlockText ContentBlockType = "text"
    ContentBlockThinking ContentBlockType = "thinking"
    // ...
)
```

### 4. 验证

内置验证方法：

```go
if err := block.Validate(); err != nil {
    log.Fatalf("内容块无效: %v", err)
}
```

---

## 🔄 与现有系统集成

### 与 Agent 集成

```go
// Agent 返回标准内容块
func (a *Agent) Run(ctx context.Context, input string) (*types.ContentBlock, error) {
    result := types.NewTextContentBlock(output).
        WithReasoning(a.steps).
        WithMetadata("agent_type", a.Type)
    
    return result, nil
}
```

### 与 RAG 集成

```go
// RAG 系统返回带引用的内容块
func (r *RAGSystem) Query(query string) (*types.ContentBlock, error) {
    docs := r.Retrieve(query)
    answer := r.Generate(query, docs)
    
    block := types.NewTextContentBlock(answer)
    for _, doc := range docs {
        block.AddCitation(types.Citation{
            Source: doc.Source,
            Score: doc.Score,
        })
    }
    
    return block, nil
}
```

### 与 LLM 集成

```go
// LLM 返回包含推理的内容块
func (l *LLM) Generate(ctx context.Context, prompt string) (*types.ContentBlock, error) {
    response := l.Call(prompt)
    
    block := types.NewTextContentBlock(response.Text).
        WithMetadata("model", l.ModelName).
        WithMetadata("tokens", response.Tokens).
        WithConfidence(response.Confidence)
    
    return block, nil
}
```

---

## 🧪 测试

### 单元测试

```go
func TestContentBlock(t *testing.T) {
    block := types.NewTextContentBlock("test").
        WithConfidence(0.95)
    
    if block.Type != types.ContentBlockText {
        t.Error("type mismatch")
    }
    
    if *block.Confidence != 0.95 {
        t.Error("confidence mismatch")
    }
}
```

### 集成测试

```go
func TestRAGWithContentBlock(t *testing.T) {
    // 模拟完整 RAG 流程
    list := types.NewContentBlockList()
    
    // 添加各种内容块...
    
    // 验证完整性
    citations := list.GetAllCitations()
    if len(citations) == 0 {
        t.Error("should have citations")
    }
    
    // 验证 JSON 序列化
    jsonStr, err := list.ToJSON()
    if err != nil {
        t.Fatalf("ToJSON failed: %v", err)
    }
    
    var parsed types.ContentBlockList
    if err := parsed.FromJSON(jsonStr); err != nil {
        t.Fatalf("FromJSON failed: %v", err)
    }
}
```

---

## ⚡ 性能优化

### 1. 预分配容量

```go
// 如果知道大小，预分配
block := types.NewTextContentBlock("content")
block.Reasoning = make([]string, 0, 10)  // 预分配 10 个
block.Citations = make([]types.Citation, 0, 5)  // 预分配 5 个
```

### 2. 避免不必要的拷贝

```go
// 使用指针避免大对象拷贝
func ProcessBlock(block *types.ContentBlock) {
    // 直接操作指针
    block.WithMetadata("processed", true)
}
```

### 3. 批量操作

```go
// 批量添加引用
citations := []types.Citation{
    {Source: "doc1.pdf", Score: 0.95},
    {Source: "doc2.pdf", Score: 0.88},
}
block.WithCitations(citations)  // 一次性设置
```

---

## 📊 JSON 格式示例

完整的 JSON 输出示例：

```json
{
  "type": "text",
  "content": "机器学习是人工智能的一个分支...",
  "reasoning": [
    "分析用户问题",
    "搜索相关文档",
    "综合多个来源得出答案"
  ],
  "citations": [
    {
      "source": "ml_intro.pdf",
      "excerpt": "机器学习定义...",
      "score": 0.95,
      "page": 1,
      "title": "机器学习入门"
    },
    {
      "source": "ai_handbook.pdf",
      "excerpt": "AI 与 ML 的关系...",
      "score": 0.88,
      "page": 23,
      "title": "人工智能手册",
      "url": "https://example.com/ai-handbook.pdf"
    }
  ],
  "metadata": {
    "model": "gpt-4",
    "tokens": 150,
    "latency_ms": 1250
  },
  "timestamp": "2026-01-20T00:18:36.241806+08:00",
  "id": "answer_1",
  "confidence": 0.92
}
```

---

## 🔗 相关资源

- **源码**: `pkg/types/content_block.go`
- **测试**: `pkg/types/content_block_test.go`
- **示例**: `examples/content_block_demo/content_block_demo.go`
- **设计文档**: [LangChain v1.0 Content Blocks](https://blog.langchain.com/langchain-langgraph-1dot0/)

---

## 🎓 最佳实践

### 1. 始终设置 ID

```go
block := types.NewTextContentBlock("content").
    WithID("unique_id")  // 便于追踪
```

### 2. 添加置信度

```go
block.WithConfidence(0.92)  // 帮助下游系统判断可信度
```

### 3. 记录推理过程

```go
block.WithReasoning([]string{
    "步骤1: ...",
    "步骤2: ...",
})  // 提高可解释性
```

### 4. RAG 必须添加引用

```go
block.AddCitation(types.Citation{
    Source: "source.pdf",
    Score: 0.95,
})  // 支持事实核查
```

### 5. 使用元数据记录关键信息

```go
block.WithMetadata("model", "gpt-4").
     WithMetadata("tokens", 150).
     WithMetadata("latency_ms", 1200)
```

### 6. 验证内容块

```go
if err := block.Validate(); err != nil {
    return fmt.Errorf("invalid block: %w", err)
}
```

---

## 🐛 故障排查

### 问题1: JSON 解析失败

```go
// 确保 JSON 格式正确
var block types.ContentBlock
if err := block.FromJSON(jsonStr); err != nil {
    log.Printf("解析失败: %v", err)
    // 检查 JSON 格式
}
```

### 问题2: 验证失败

```go
// 检查必填字段
if block.Type == types.ContentBlockToolUse && len(block.ToolCalls) == 0 {
    // tool_use 必须有 ToolCalls
}
```

### 问题3: 置信度超出范围

```go
// 置信度必须在 0-1 之间
confidence := 0.95
if confidence < 0 || confidence > 1 {
    return fmt.Errorf("invalid confidence: %f", confidence)
}
block.WithConfidence(confidence)
```

---

## 🚀 下一步

- [Agent Middleware 系统](./AGENT_MIDDLEWARE.md)
- [Streaming 支持](./STREAMING.md)
- [Hybrid Search](./HYBRID_SEARCH.md)

---

**版本**: v0.1.2  
**状态**: ✅ 已实现  
**最后更新**: 2026-01-20

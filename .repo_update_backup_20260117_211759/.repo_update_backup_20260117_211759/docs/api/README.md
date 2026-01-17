# API 参考文档

LangChain-Go API 的完整参考文档。

---

## 📖 API 文档

### 在线文档

- **[GoDoc](https://pkg.go.dev/langchain-go)** - 完整的 API 参考文档

---

## 🏗️ 包结构

### 核心包

```
langchain-go/
├── core/
│   ├── types/          # 基础类型（Message, Tool, Schema）
│   ├── runnable/       # Runnable 系统
│   ├── chatmodels/     # ChatModel 接口
│   ├── prompts/        # Prompt 模板
│   ├── output/         # OutputParser
│   ├── tools/          # 工具系统
│   ├── agents/         # Agent 系统
│   └── memory/         # 记忆系统
├── chatmodels/
│   ├── openai/         # OpenAI 集成
│   └── anthropic/      # Anthropic 集成
├── state/              # StateGraph 系统
├── checkpoints/        # Checkpoint 持久化
├── execute/            # 执行引擎
└── retrieval/          # RAG 系统
    ├── loaders/        # 文档加载器
    ├── splitters/      # 文本分割器
    ├── embeddings/     # 嵌入模型
    └── vectorstores/   # 向量存储
```

---

## 📚 核心包详解

### `core/types`
基础类型定义。

**主要类型：**
- `Message` - 消息类型
- `Tool` - 工具定义
- `Schema` - JSON Schema
- `Document` - 文档类型

**导入：**
```go
import "github.com/zhucl121/langchain-go/core/types"
```

### `core/runnable`
Runnable 系统，LCEL 链式组合的核心。

**主要接口：**
- `Runnable[I, O]` - 可执行接口
- `RunnableFunc[I, O]` - 函数包装器
- `RunnableChain[I, M, O]` - 链式组合

**导入：**
```go
import "github.com/zhucl121/langchain-go/core/runnable"
```

### `core/chatmodels`
对话模型接口。

**主要接口：**
- `ChatModel` - ChatModel 接口
- `StreamingChatModel` - 流式接口

**导入：**
```go
import "github.com/zhucl121/langchain-go/core/chatmodels"
```

### `core/prompts`
提示词模板系统。

**主要类型：**
- `PromptTemplate` - 简单模板
- `ChatPromptTemplate` - 对话模板
- `FewShotPromptTemplate` - Few-Shot 模板

**导入：**
```go
import "github.com/zhucl121/langchain-go/core/prompts"
```

### `core/output`
输出解析器。

**主要类型：**
- `OutputParser[T]` - 解析器接口
- `JSONParser[T]` - JSON 解析器
- `StructuredOutputParser` - 结构化解析器

**导入：**
```go
import "github.com/zhucl121/langchain-go/core/output"
```

### `core/tools`
工具系统。

**主要类型：**
- `Tool` - 工具接口
- `FunctionTool` - 函数工具
- `StructuredTool` - 结构化工具

**导入：**
```go
import "github.com/zhucl121/langchain-go/core/tools"
```

### `core/agents`
Agent 系统。

**主要类型：**
- `Agent` - Agent 接口
- `Executor` - 执行器
- `Middleware` - 中间件

**导入：**
```go
import "github.com/zhucl121/langchain-go/core/agents"
```

### `state`
StateGraph 状态图系统。

**主要类型：**
- `StateGraph[S]` - 状态图
- `CompiledGraph[S]` - 编译后的图
- `NodeFunc[S]` - 节点函数

**导入：**
```go
import "github.com/zhucl121/langchain-go/state"
```

### `checkpoints`
Checkpoint 持久化。

**主要接口：**
- `Checkpointer` - 检查点接口
- `Saver` - 保存器接口

**实现：**
- `postgres.NewSaver()` - PostgreSQL
- `sqlite.NewSaver()` - SQLite
- `memory.NewSaver()` - 内存

**导入：**
```go
import "github.com/zhucl121/langchain-go/checkpoints/postgres"
import "github.com/zhucl121/langchain-go/checkpoints/sqlite"
```

### `retrieval/loaders`
文档加载器。

**支持格式：**
- Text, Markdown, JSON, CSV
- PDF
- Word/DOCX
- HTML/Web
- Excel

**导入：**
```go
import "github.com/zhucl121/langchain-go/retrieval/loaders"
```

### `retrieval/vectorstores`
向量存储。

**支持的向量数据库：**
- InMemory
- Milvus
- Chroma
- Pinecone

**导入：**
```go
import "github.com/zhucl121/langchain-go/retrieval/vectorstores"
```

---

## 🔍 快速查找

### 按功能分类

#### 对话模型
- `chatmodels/openai` - OpenAI 模型
- `chatmodels/anthropic` - Anthropic 模型

#### 工作流编排
- `state` - StateGraph
- `checkpoints` - 持久化
- `execute` - 执行引擎

#### RAG 系统
- `retrieval/loaders` - 文档加载
- `retrieval/splitters` - 文本分割
- `retrieval/embeddings` - 嵌入模型
- `retrieval/vectorstores` - 向量存储

#### Agent 系统
- `core/agents` - Agent 核心
- `core/agents/react` - ReAct Agent
- `core/agents/planexecute` - Plan-Execute Agent

#### 工具和集成
- `core/tools` - 工具系统
- `integrations/search` - 搜索工具
- `integrations/observability` - 可观测性

---

## 📖 使用示例

### 查看包文档

```bash
# 查看包文档
go doc langchain-go/core/runnable

# 查看具体类型
go doc langchain-go/core/runnable.Runnable

# 查看方法
go doc langchain-go/core/runnable.Runnable.Invoke
```

### 在代码中使用

```go
// 导入包
import (
    "github.com/zhucl121/langchain-go/core/runnable"
    "github.com/zhucl121/langchain-go/core/types"
    "github.com/zhucl121/langchain-go/chatmodels/openai"
)

// 使用 API
model := openai.New(openai.Config{
    APIKey: "sk-...",
})

result, err := model.Invoke(ctx, []types.Message{
    types.NewUserMessage("Hello!"),
})
```

---

## 📚 相关资源

- [快速开始](../getting-started/) - 新手入门
- [使用指南](../guides/) - 详细用法
- [示例代码](../examples/) - 实用示例

---

<div align="center">

**[回到文档首页](../README.md)**

</div>

# LangChain-Go 快速开始

欢迎使用 LangChain-Go! 本指南将在 5 分钟内帮助您上手使用。

---

## 📦 安装

```bash
go get github.com/zhucl121/langchain-go
```

**系统要求**:
- Go 1.21 或更高版本
- (可选) Docker Desktop - 用于运行测试

---

## 🚀 30秒上手

### 1. 最简单的示例 - 调用 LLM

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/zhucl121/langchain-go/core/chat/providers/openai"
    "github.com/zhucl121/langchain-go/pkg/types"
)

func main() {
    // 创建 OpenAI 客户端
    model, err := openai.New(openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-3.5-turbo",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 发送消息
    response, err := model.Invoke(context.Background(), []types.Message{
        types.NewUserMessage("你好,请介绍一下你自己"),
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Content)
}
```

### 2. 使用本地模型 - Ollama

```go
import "github.com/zhucl121/langchain-go/core/chat/providers/ollama"

// 使用本地 Ollama 模型
model := ollama.New(ollama.Config{
    Model:   "llama2",
    BaseURL: "http://localhost:11434",
})
```

### 3. 创建简单 Agent

```go
import (
    "github.com/zhucl121/langchain-go/core/agents"
    "github.com/zhucl121/langchain-go/core/tools"
)

// 创建工具
calculator := tools.NewCalculatorTool()
search := tools.NewDuckDuckGoSearchTool(nil)

// 一行代码创建 Agent
agent := agents.CreateReActAgent(llm, []tools.Tool{calculator, search})

// 执行任务
result, _ := agent.Run(context.Background(), 
    "搜索今天的天气,然后计算25的平方根")
fmt.Println(result)
```

### 4. 3行代码实现 RAG

```go
import (
    "github.com/zhucl121/langchain-go/retrieval/chains"
    "github.com/zhucl121/langchain-go/retrieval/retrievers"
)

retriever := retrievers.NewVectorStoreRetriever(vectorStore)
ragChain := chains.NewRAGChain(retriever, llm)
result, _ := ragChain.Run(context.Background(), "What is LangChain?")
```

---

## 🎯 核心功能快速导航

### Agent 系统

LangChain-Go 提供 7 种 Agent 类型:

```go
// 1. ReAct Agent - 推理和行动
agent := agents.CreateReActAgent(llm, tools)

// 2. Tool Calling Agent - 函数调用
agent := agents.CreateToolCallingAgent(llm, tools)

// 3. OpenAI Functions Agent
agent := agents.CreateOpenAIFunctionsAgent(llm, tools)

// 4. Plan-Execute Agent - 计划执行
agent := agents.CreatePlanExecuteAgent(llm, tools)

// 5. Self-Ask Agent - 自问自答
agent := agents.CreateSelfAskAgent(llm, tools)

// 6. Structured Chat Agent - 结构化对话
agent := agents.CreateStructuredChatAgent(llm, tools)

// 7. Conversational Agent - 对话型
agent := agents.CreateConversationalAgent(llm, tools, memory)
```

### Multi-Agent 协作

创建多 Agent 系统处理复杂任务:

```go
// 创建协调策略
strategy := agents.NewSequentialStrategy(llm)
coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

// 创建多 Agent 系统
system := agents.NewMultiAgentSystem(coordinator, nil)

// 添加专用 Agent
researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
writer := agents.NewWriterAgent("writer", llm, nil)

system.AddAgent("researcher", researcher)
system.AddAgent("writer", writer)

// 执行复杂任务
result, _ := system.Run(context.Background(), 
    "研究Go语言的最新特性,然后写一篇技术文章")
```

### 工具生态

38 个内置工具,开箱即用:

```go
// 基础工具
tools.NewCalculatorTool()
tools.NewGetTimeTool()
tools.NewGetDateTool()

// 搜索工具
tools.NewDuckDuckGoSearchTool(nil)
tools.NewGoogleSearchTool(&googleConfig)

// 文件工具
tools.NewReadFileTool()
tools.NewWriteFileTool()

// 多模态工具
tools.NewImageAnalysisTool(config)
tools.NewSpeechToTextTool(config)
tools.NewTextToSpeechTool(config)

// 获取所有工具
allTools := tools.GetBuiltinTools()
```

### RAG 能力

完整的 RAG 工作流:

```go
// 1. 加载文档
loader := loaders.NewPDFLoader("document.pdf")
documents, _ := loader.Load()

// 2. 分割文本
splitter := splitters.NewCharacterSplitter(1000, 200)
chunks := splitter.SplitDocuments(documents)

// 3. 创建向量存储
embeddings := embeddings.NewOpenAIEmbeddings(config)
vectorStore := vectorstores.NewMilvusVectorStore(config, embeddings)
vectorStore.AddDocuments(chunks)

// 4. 创建 RAG Chain
retriever := retrievers.NewVectorStoreRetriever(vectorStore)
ragChain := chains.NewRAGChain(retriever, llm)

// 5. 查询
answer, _ := ragChain.Run(context.Background(), "你的问题")
```

---

## 🎓 学习路径

### 初学者 (30分钟)

1. **安装和配置** (5分钟)
   - 安装 LangChain-Go
   - 获取 API Key (OpenAI/Anthropic)

2. **第一个 Agent** (10分钟)
   - 运行 `examples/agent_simple_demo.go`
   - 理解 Agent 工作原理

3. **使用工具** (15分钟)
   - 运行 `examples/search_tools_demo.go`
   - 尝试不同的内置工具

### 进阶用户 (2小时)

1. **Multi-Agent 系统** (45分钟)
   - 运行 `examples/multi_agent_demo.go`
   - 创建自定义 Agent

2. **RAG 应用** (45分钟)
   - 运行 `examples/pdf_loader_demo.go`
   - 实现文档问答系统

3. **多模态应用** (30分钟)
   - 运行 `examples/multimodal_demo.go`
   - 处理图像、音频

### 高级用户

1. **深入文档**
   - 阅读 [使用指南](docs/guides/)
   - 学习 [LangGraph](docs/guides/langgraph/)

2. **生产部署**
   - 配置 [Redis 缓存](docs/guides/redis-cache.md)
   - 集成 [可观测性](docs/advanced/performance.md)

3. **贡献代码**
   - 查看 [贡献指南](CONTRIBUTING.md)
   - 提交 Pull Request

---

## 📖 示例程序

项目包含 11 个完整示例:

```bash
cd examples

# 1. 简单 Agent
go run agent_simple_demo.go

# 2. Multi-Agent 协作
go run multi_agent_demo.go

# 3. 多模态处理
go run multimodal_demo.go

# 4. 计划执行 Agent
go run plan_execute_agent_demo.go

# 5. 搜索工具
go run search_tools_demo.go

# 6. Self-Ask Agent
go run selfask_agent_demo.go

# 7. 结构化对话
go run structured_chat_demo.go

# 8. PDF 文档加载
go run pdf_loader_demo.go

# 9. Prompt Hub
go run prompt_hub_demo.go

# 10. Redis 缓存
go run redis_cache_demo.go

# 11. 高级搜索
go run advanced_search_demo.go
```

**注意**: 运行示例前需要设置环境变量:

```bash
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"  # 可选
```

---

## 🔧 常见任务

### 更换 LLM 提供商

```go
// OpenAI
import "github.com/zhucl121/langchain-go/core/chat/providers/openai"
model := openai.New(openai.Config{APIKey: "...", Model: "gpt-4"})

// Claude
import "github.com/zhucl121/langchain-go/core/chat/providers/anthropic"
model := anthropic.New(anthropic.Config{APIKey: "...", Model: "claude-3-sonnet-20240229"})

// Ollama (本地)
import "github.com/zhucl121/langchain-go/core/chat/providers/ollama"
model := ollama.New(ollama.Config{Model: "llama2", BaseURL: "http://localhost:11434"})
```

### 自定义工具

```go
import "github.com/zhucl121/langchain-go/core/tools"

customTool := tools.NewFunctionTool(tools.FunctionToolConfig{
    Name:        "my_custom_tool",
    Description: "这是我的自定义工具",
    Fn: func(ctx context.Context, input map[string]any) (any, error) {
        // 你的工具逻辑
        return "result", nil
    },
})
```

### 添加记忆

```go
import "github.com/zhucl121/langchain-go/core/memory"

// 创建记忆
memory := memory.NewBufferMemory()

// 在 Agent 中使用
agent := agents.CreateConversationalAgent(llm, tools, memory)
```

### 启用缓存

```go
import "github.com/zhucl121/langchain-go/core/cache"

// 配置 Redis 缓存
config := cache.DefaultRedisCacheConfig()
config.Password = "your-password"
redisCache, _ := cache.NewRedisCache(config)

// 创建 LLM 缓存
llmCache := cache.NewLLMCache(redisCache)

// 在 LLM 调用中使用缓存可节省 50-90% 成本
```

---

## 💡 使用技巧

### 1. 流式输出

```go
// Agent 支持流式输出
executor := agents.NewSimplifiedAgentExecutor(agent, tools)
executor.Stream = true

result, _ := executor.Run(ctx, "your task")
```

### 2. 并行工具执行

```go
// 工具会自动并行执行,提升 3x 性能
executor := tools.NewToolExecutor(tools, nil)
executor.MaxParallel = 5  // 最多并行 5 个工具
```

### 3. 错误处理和重试

```go
// 自动重试配置
import "github.com/zhucl121/langchain-go/pkg/types"

retryPolicy := types.RetryPolicy{
    MaxRetries: 3,
    Backoff:    types.ExponentialBackoff,
}

// Agent 会自动使用重试策略
```

---

## 📚 更多资源

- 📘 [完整文档](docs/) - 详细使用指南
- 📗 [API 参考](https://pkg.go.dev/github.com/zhucl121/langchain-go) - GoDoc 文档
- 📕 [示例代码](examples/) - 11 个完整示例
- 📙 [变更日志](CHANGELOG.md) - 版本更新记录
- 💡 [贡献指南](CONTRIBUTING.md) - 如何贡献

---

## ❓ 遇到问题?

1. **查看文档**: [docs/](docs/)
2. **运行示例**: [examples/](examples/)
3. **查看测试**: 测试文件是最好的使用示例
4. **提交 Issue**: [GitHub Issues](https://github.com/zhucl121/langchain-go/issues)
5. **加入讨论**: [GitHub Discussions](https://github.com/zhucl121/langchain-go/discussions)

---

## 🎯 下一步

- ✅ 运行几个示例程序,熟悉基本用法
- ✅ 阅读 [使用指南](docs/guides/),深入了解核心功能
- ✅ 构建你的第一个 AI 应用
- ✅ 给项目一个 ⭐ Star,支持开发!

---

**祝使用愉快! 🚀**

如有问题,欢迎随时提问或查阅文档。

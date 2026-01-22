# LangChain-Go

[![Go Version](https://img.shields.io/github/go-mod/go-version/zhucl121/langchain-go)](https://github.com/zhucl121/langchain-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhucl121/langchain-go)](https://goreportcard.com/report/github.com/zhucl121/langchain-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/zhucl121/langchain-go)](https://pkg.go.dev/github.com/zhucl121/langchain-go)

🎯 **生产就绪的 Go AI 开发框架**

LangChain-Go 是 [LangChain](https://github.com/langchain-ai/langchain) 和 [LangGraph](https://github.com/langchain-ai/langgraph) 的完整 Go 语言实现，针对 Go 生态优化，提供高性能、类型安全的 AI 应用开发体验。

## ✨ 核心特性

- 🤖 **7种Agent类型** - ReAct、ToolCalling、Conversational、PlanExecute、OpenAI Functions、SelfAsk、StructuredChat
- 🔗 **MCP协议** - 与 Claude Desktop 互操作，Go 生态首个实现 🔥 v0.6.1 NEW!
- 🤝 **A2A协议** - 跨语言、跨系统 Agent 标准化协作 🔥 v0.6.1 NEW!
- 🌐 **协议桥接** - MCP ↔ A2A 无缝互操作 🔥 v0.6.1 NEW!
- 🤝 **Multi-Agent协作** - 完整的多Agent协作系统，支持顺序、并行、层次化执行策略
- 🛠️ **38个内置工具** - 计算、搜索、文件、数据、HTTP、多模态（图像、音频、视频）
- 🚀 **3行代码RAG** - 简化的RAG Chain API，从150行代码降至3行
- 🧠 **学习型检索** - 自动收集反馈、质量评估、参数优化、A/B测试
- 📊 **GraphRAG** - 知识图谱增强检索，支持 Neo4j, NebulaGraph
- 🗄️ **5个向量存储** - Milvus, Chroma, Qdrant, Weaviate, Redis，支持混合搜索
- 📚 **8个文档加载器** - 支持 GitHub, Confluence, PostgreSQL 等多种数据源
- 🌐 **6个LLM提供商** - OpenAI, Anthropic, Gemini, Bedrock, Azure, Ollama
- ⚡ **分布式部署** - 集群管理、负载均衡、分布式缓存、故障转移
- 🏢 **企业级安全** - RBAC、多租户、审计日志、数据安全 v0.6.0
- 💾 **生产级特性** - Redis缓存、自动重试、状态持久化、可观测性、Prometheus指标
- 📦 **完整文档** - 65+文档页面，中英文双语，含25个示例程序

## 🚀 快速开始

### 安装

```bash
go get github.com/zhucl121/langchain-go
```

### 支持的 LLM 提供商

LangChain-Go 支持主流 LLM 提供商，开箱即用：

- ✅ **OpenAI** - GPT-3.5, GPT-4, GPT-4 Turbo, GPT-4o
- ✅ **Anthropic** - Claude 3 (Opus, Sonnet, Haiku)
- ✅ **Google Gemini** - Gemini Pro, Gemini 1.5 Pro/Flash（100万+ tokens上下文）⭐ NEW!
- ✅ **AWS Bedrock** - Claude, Titan, Llama, Cohere（企业级托管）⭐ NEW!
- ✅ **Azure OpenAI** - 企业级 GPT 模型（私有部署）⭐ NEW!
- ✅ **Ollama** - 本地运行开源模型（Llama 2, Mistral, CodeLlama 等）

```go
// OpenAI
import "github.com/zhucl121/langchain-go/core/chat/providers/openai"
model := openai.New(openai.Config{APIKey: "...", Model: "gpt-4"})

// Google Gemini
import "github.com/zhucl121/langchain-go/core/chat/providers/gemini"
model, _ := gemini.New(gemini.Config{APIKey: "...", Model: "gemini-pro"})

// AWS Bedrock
import "github.com/zhucl121/langchain-go/core/chat/providers/bedrock"
model, _ := bedrock.New(bedrock.Config{
    Region: "us-east-1", AccessKey: "...", SecretKey: "...",
    Model: "anthropic.claude-v2",
})

// Azure OpenAI
import "github.com/zhucl121/langchain-go/core/chat/providers/azure"
model, _ := azure.New(azure.Config{
    Endpoint: "https://your-resource.openai.azure.com",
    APIKey: "...", Deployment: "gpt-35-turbo",
})

// Ollama (本地模型)
import "github.com/zhucl121/langchain-go/core/chat/providers/ollama"
model := ollama.New(ollama.Config{Model: "llama2", BaseURL: "http://localhost:11434"})
```

### 30秒上手

#### 1. 简单的RAG应用（3行代码）

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/retrieval/chains"
    "github.com/zhucl121/langchain-go/retrieval/retrievers"
)

func main() {
    retriever := retrievers.NewVectorStoreRetriever(vectorStore)
    ragChain := chains.NewRAGChain(retriever, llm)
    result, _ := ragChain.Run(context.Background(), "What is LangChain?")
    println(result)
}
```

#### 2. 创建ReAct Agent

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/core/agents"
    "github.com/zhucl121/langchain-go/core/tools"
)

func main() {
    // 创建工具
    calculator := tools.NewCalculatorTool()
    search := tools.NewDuckDuckGoSearchTool(nil)
    
    // 创建Agent（1行）
    agent := agents.CreateReActAgent(llm, []tools.Tool{calculator, search})
    
    // 执行任务
    result, _ := agent.Run(context.Background(), 
        "搜索今天的天气，然后计算25的平方根")
    println(result)
}
```

#### 3. Multi-Agent协作

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/core/agents"
)

func main() {
    // 创建协调策略
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // 添加专用Agent
    researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
    writer := agents.NewWriterAgent("writer", llm, nil)
    
    system.AddAgent("researcher", researcher)
    system.AddAgent("writer", writer)
    
    // 执行复杂任务
    result, _ := system.Run(context.Background(), 
        "研究Go语言的最新特性，然后写一篇技术文章")
    println(result)
}
```

#### 4. 向量存储和文档加载 ⭐ NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores"
    "github.com/zhucl121/langchain-go/retrieval/loaders"
)

func main() {
    // Chroma 向量存储
    chromaConfig := vectorstores.ChromaConfig{
        URL:            "http://localhost:8000",
        CollectionName: "docs",
    }
    chromaStore := vectorstores.NewChromaVectorStore(chromaConfig, embedder)
    
    // Qdrant 向量存储（高性能）
    qdrantConfig := vectorstores.QdrantConfig{
        URL:            "http://localhost:6333",
        CollectionName: "docs",
        VectorSize:     384,
    }
    qdrantStore := vectorstores.NewQdrantVectorStore(qdrantConfig, embedder)
    
    // GitHub 文档加载器
    githubConfig := loaders.GitHubLoaderConfig{
        Owner:  "langchain-ai",
        Repo:   "langchain",
        Branch: "main",
        FileExtensions: []string{".md"},
    }
    githubLoader, _ := loaders.NewGitHubLoader(githubConfig)
    docs, _ := githubLoader.LoadDirectory(context.Background(), "docs")
    
    // Confluence 文档加载器
    confluenceConfig := loaders.ConfluenceLoaderConfig{
        URL:      "https://your-domain.atlassian.net/wiki",
        Username: "user@example.com",
        APIToken: "your-api-token",
    }
    confluenceLoader, _ := loaders.NewConfluenceLoader(confluenceConfig)
    docs, _ = confluenceLoader.LoadSpace(context.Background(), "SPACE_KEY")
    
    // PostgreSQL 数据库加载器
    pgConfig := loaders.PostgreSQLLoaderConfig{
        Host:     "localhost",
        Port:     5432,
        Database: "mydb",
        User:     "postgres",
        Password: "password",
    }
    pgLoader, _ := loaders.NewPostgreSQLLoader(pgConfig)
    defer pgLoader.Close()
    docs, _ = pgLoader.LoadTable(context.Background(), "documents", "content", "title")
}
```

#### 5. 学习型检索系统 🔥 v0.4.2 NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/retrieval/learning/feedback"
    "github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
    "github.com/zhucl121/langchain-go/retrieval/learning/optimization"
)

func main() {
    // 1. 收集用户反馈
    storage := feedback.NewMemoryStorage()
    collector := feedback.NewCollector(storage)
    
    collector.RecordQuery(ctx, query)
    collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
        Type: feedback.FeedbackTypeRating,
        Rating: 5,
    })
    
    // 2. 评估检索质量
    evaluator := evaluation.NewEvaluator(collector)
    metrics, _ := evaluator.EvaluateQuery(ctx, queryFeedback)
    fmt.Printf("NDCG: %.3f, MRR: %.3f\n", metrics.NDCG, metrics.MRR)
    
    // 3. 自动优化参数
    optimizer := optimization.NewOptimizer(evaluator, collector, config)
    result, _ := optimizer.Optimize(ctx, strategyID, paramSpace, opts)
    fmt.Printf("性能提升: %.2f%%\n", result.Improvement)
    
    // 4. A/B 测试验证
    abtestManager := abtest.NewManager(storage)
    analysis, _ := abtestManager.AnalyzeExperiment(ctx, experimentID)
    fmt.Printf("获胜者: %s, p-value: %.3f\n", 
        analysis.Winner, analysis.PValue)
}
```

#### 6. MCP协议 - 与Claude Desktop互操作 🔥 v0.6.1 NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/pkg/protocols/mcp"
    "github.com/zhucl121/langchain-go/pkg/protocols/mcp/providers"
    "github.com/zhucl121/langchain-go/pkg/protocols/mcp/transport"
)

func main() {
    // 创建 MCP Server
    server := mcp.NewServer(mcp.ServerConfig{
        Name:    "my-server",
        Version: "1.0.0",
    })
    
    // 注册资源
    fsProvider := providers.NewFileSystemProvider("/data/documents")
    server.RegisterResource(&mcp.Resource{
        URI:  "file:///documents",
        Name: "Company Documents",
    }, fsProvider)
    
    // 注册工具
    server.RegisterTool(calculatorTool, calculatorHandler)
    
    // 启动（Claude Desktop 可连接）
    server.Serve(context.Background(), transport.NewStdioTransport())
}
```

#### 7. A2A协议 - Agent间标准化协作 🔥 v0.6.1 NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/pkg/protocols/a2a"
)

func main() {
    // 桥接现有 Agent
    a2aAgent := a2a.NewA2AAgentBridge(myAgent, &a2a.BridgeConfig{
        Info: &a2a.AgentInfo{
            ID:   "agent-1",
            Name: "Research Agent",
        },
        Capabilities: &a2a.AgentCapabilities{
            Capabilities: []string{"research", "search"},
        },
    })
    
    // 注册到注册中心
    registry := a2a.NewLocalRegistry()
    registry.Register(context.Background(), a2aAgent)
    
    // 智能路由和协作
    router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{
        Strategy: a2a.StrategyHybrid,
    })
    
    agent, _ := router.Route(context.Background(), task)
    response, _ := agent.SendTask(context.Background(), task)
}
```

## 📊 性能对比

| 特性 | 传统实现 | LangChain-Go |
|------|---------|-------------|
| RAG应用代码量 | 150+ 行 | **3 行** ⚡ |
| Agent创建 | 50+ 行 | **1 行** ⚡ |
| 缓存命中响应 | 3-5秒 | **30-50ns** ⚡ |
| 工具并行执行 | 不支持 | **3x提速** ⚡ |
| 成本节省 | - | **50-90%** 💰 |

## 🎯 核心功能

### 1. Agent系统

- **7种Agent类型**，覆盖各种使用场景
- **高层工厂函数**，一行代码创建Agent
- **流式输出**，实时展示Agent思考过程
- **状态持久化**，支持长时间运行任务
- **自动重试**，生产级错误处理

### 2. Multi-Agent协作

- **消息总线**，Agent间高效通信
- **3种协调策略**：顺序、并行、层次化
- **6个专用Agent**：协调器、研究员、作者、审核、分析师、规划师
- **共享状态**，协作信息透明
- **执行追踪**，完整的历史记录

### 3. 工具生态

- **38个内置工具**，开箱即用
- **工具注册中心**，动态管理工具
- **并行执行**，提升性能3倍
- **自定义工具**，简单扩展
- **多模态支持**，处理图像、音频、视频

### 4. RAG能力

- **3行代码**实现完整RAG
- **学习型检索**，自动优化检索质量 🔥 v0.4.2 NEW!
- **GraphRAG**，知识图谱增强检索
- **多种Retriever**，灵活选择
- **5个主流向量存储**：Milvus, Chroma, Qdrant, Weaviate, Redis
- **8个文档加载器**：PDF, Word, Excel, HTML, Text, GitHub, Confluence, PostgreSQL
- **文本分割器**，智能分块
- **混合搜索**，向量 + BM25

### 5. 生产特性

- **Redis缓存**，节省50-90%成本
- **自动重试**，指数退避策略
- **可观测性**，OpenTelemetry集成
- **Prometheus指标**，完整监控
- **结构化日志**，便于调试

## 📖 文档

- 📘 [快速开始](QUICK_START.md) - 5分钟快速上手
- 📗 [完整文档](docs/README.md) - 详细使用指南
- 🔗 [MCP & A2A 指南](docs/V0.6.1_USER_GUIDE.md) - 标准化协议 🔥 v0.6.1
- 📕 [Agent 指南](docs/guides/agents/README.md) - Agent 系统文档
- 📙 [Multi-Agent 系统](docs/guides/multi-agent-guide.md) - 多Agent协作
- 📚 [RAG 指南](docs/guides/rag/README.md) - RAG 系统文档
- 🧠 [Learning Retrieval 指南](docs/V0.4.2_USER_GUIDE.md) - 学习型检索
- 🏢 [企业安全指南](docs/V0.6.0_PROGRESS.md) - RBAC 和多租户 v0.6.0
- 💡 [示例代码](examples/) - 25个完整示例

## 🔧 示例程序

查看 [examples/](examples/) 目录：

**Agent & Multi-Agent**:
- `agent_simple_demo.go` - 简单Agent示例
- `multi_agent_demo.go` - Multi-Agent协作
- `plan_execute_agent_demo.go` - 计划执行Agent

**Learning Retrieval (v0.4.2)** 🔥:
- `learning_complete_demo/` - 完整学习型检索工作流
- `learning_feedback_demo/` - 用户反馈收集
- `learning_evaluation_demo/` - 检索质量评估
- `learning_optimization_demo/` - 参数自动优化
- `learning_abtest_demo/` - A/B 测试框架
- `learning_postgres_demo/` - PostgreSQL 存储

**多模态 & 工具**:
- `multimodal_demo.go` - 多模态处理
- `redis_cache_demo.go` - Redis缓存使用
- 更多...

## 🏗️ 架构

```
langchain-go/
├── core/              # 核心功能
│   ├── agents/       # Agent实现
│   ├── tools/        # 内置工具
│   ├── prompts/      # Prompt模板
│   ├── memory/       # 记忆系统
│   ├── cache/        # 缓存层
│   └── ...
├── graph/            # LangGraph实现
│   ├── node/         # 图节点
│   ├── edge/         # 图边
│   ├── checkpoint/   # 检查点
│   └── ...
├── retrieval/        # RAG相关
│   ├── chains/       # RAG Chain
│   ├── retrievers/   # Retriever
│   ├── loaders/      # 文档加载
│   └── ...
├── pkg/              # 公共包
│   ├── types/        # 类型定义
│   └── observability/# 可观测性
└── examples/         # 示例代码
```

## 🆚 对比

### vs Python LangChain

| 特性 | Python LangChain | LangChain-Go |
|------|-----------------|-------------|
| 性能 | 慢 | **快** (编译型) ⚡ |
| 类型安全 | 运行时 | **编译时** ✅ |
| 并发 | GIL限制 | **原生支持** 🚀 |
| 部署 | 依赖复杂 | **单二进制** 📦 |
| 内存占用 | 高 | **低** 💾 |
| 生态系统 | 丰富 | 精选 |

### 为什么选择Go版本？

- ✅ **高性能**：编译型语言，无GIL限制
- ✅ **类型安全**：编译时错误检查
- ✅ **并发友好**：原生goroutine支持
- ✅ **部署简单**：单一二进制文件
- ✅ **内存高效**：更低的资源占用
- ✅ **生产就绪**：内置可观测性和监控

## 📈 技术指标

- **代码量**：40,000+ 行（v0.6.1 新增 3,300+ 行）🔥
- **测试覆盖**：85%+
- **测试用例**：626+
- **协议支持**：2个（MCP, A2A）🔥 v0.6.1 NEW!
- **LLM 提供商**：6个（OpenAI, Anthropic, Gemini, Bedrock, Azure, Ollama）
- **向量存储**：5个（Milvus, Chroma, Qdrant, Weaviate, Redis）
- **文档加载器**：8个（PDF, Word, Excel, HTML, Text, GitHub, Confluence, PostgreSQL）
- **内置工具**：38个
- **Agent类型**：7种 + 6个专用Agent
- **Learning 模块**：4个（反馈、评估、优化、A/B测试）
- **企业特性**：5个（RBAC、多租户、审计、安全、鉴权）v0.6.0
- **文档页面**：65+
- **示例程序**：25个（v0.6.1 新增 4 个）🔥

## 🧪 测试

```bash
# 启动测试环境 (Redis + Milvus)
make -f Makefile.test test-env-up

# 运行所有测试
make -f Makefile.test test

# 停止测试环境
make -f Makefile.test test-env-down
```

详见 [测试指南](TESTING.md)

---

## 🤝 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详情。

### 贡献方式

- 🐛 [报告Bug](https://github.com/zhucl121/langchain-go/issues)
- 💡 [提出新功能](https://github.com/zhucl121/langchain-go/issues)
- 📝 [改进文档](https://github.com/zhucl121/langchain-go/pulls)
- 🔧 [提交代码](https://github.com/zhucl121/langchain-go/pulls)

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE)

## 🙏 致谢

- [LangChain](https://github.com/langchain-ai/langchain) - 原始设计灵感
- [LangGraph](https://github.com/langchain-ai/langgraph) - Graph实现参考
- Go社区 - 优秀的工具和库

## 📞 社区

- **GitHub**: [https://github.com/zhucl121/langchain-go](https://github.com/zhucl121/langchain-go)
- **Issues**: [Report bugs](https://github.com/zhucl121/langchain-go/issues)
- **Discussions**: [Ask questions](https://github.com/zhucl121/langchain-go/discussions)

## ⭐ Star History

如果这个项目对你有帮助，请给个 Star ⭐

---

**Made with ❤️ in Go**

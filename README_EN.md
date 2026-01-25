# LangChain-Go

[![Go Version](https://img.shields.io/github/go-mod/go-version/zhucl121/langchain-go)](https://github.com/zhucl121/langchain-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhucl121/langchain-go)](https://goreportcard.com/report/github.com/zhucl121/langchain-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/zhucl121/langchain-go)](https://pkg.go.dev/github.com/zhucl121/langchain-go)

🌍 **Language**: [中文](README.md) | English

🎯 **Production-Ready Go AI Development Framework**

LangChain-Go is a complete Go implementation of [LangChain](https://github.com/langchain-ai/langchain) and [LangGraph](https://github.com/langchain-ai/langgraph), optimized for the Go ecosystem, providing high-performance, type-safe AI application development experience.

## ✨ Core Features

- 🤖 **7 Agent Types** - ReAct, ToolCalling, Conversational, PlanExecute, OpenAI Functions, SelfAsk, StructuredChat
- 🔗 **MCP Protocol** - First Go implementation! Interoperability with Claude Desktop 🔥 v0.6.1 NEW!
- 🤝 **A2A Protocol** - Cross-language, cross-system standardized agent collaboration 🔥 v0.6.1 NEW!
- 🌐 **Protocol Bridge** - Seamless MCP ↔ A2A interoperability 🔥 v0.6.1 NEW!
- 🤝 **Multi-Agent System** - Complete multi-agent collaboration with sequential, parallel, hierarchical strategies
- 🛠️ **38 Built-in Tools** - Calculator, search, file, data, HTTP, multimodal (image, audio, video)
- 🚀 **3-Line RAG** - Simplified RAG Chain API, reduced from 150 lines to 3 lines
- 🧠 **Learning Retrieval** - Auto feedback collection, quality evaluation, parameter optimization, A/B testing
- 📊 **GraphRAG** - Knowledge graph enhanced retrieval with Neo4j, NebulaGraph support
- 🗄️ **5 Vector Stores** - Milvus, Chroma, Qdrant, Weaviate, Redis with hybrid search
- 📚 **8 Document Loaders** - GitHub, Confluence, PostgreSQL and more data sources
- 🌐 **6 LLM Providers** - OpenAI, Anthropic, Gemini, Bedrock, Azure, Ollama
- ⚡ **Distributed Deployment** - Cluster management, load balancing, distributed cache, failover
- 🏢 **Enterprise Security** - RBAC, multi-tenancy, audit logs, data security v0.6.0
- 💾 **Production Features** - Redis cache, auto-retry, state persistence, observability, Prometheus metrics
- 📦 **Complete Docs** - 65+ pages, bilingual (EN/CN), 25 examples

## 🚀 Quick Start

### Installation

```bash
go get github.com/zhucl121/langchain-go
```

### Supported LLM Providers

LangChain-Go supports mainstream LLM providers out of the box:

- ✅ **OpenAI** - GPT-3.5, GPT-4, GPT-4 Turbo, GPT-4o
- ✅ **Anthropic** - Claude 3 (Opus, Sonnet, Haiku)
- ✅ **Google Gemini** - Gemini Pro, Gemini 1.5 Pro/Flash (1M+ tokens context) ⭐ NEW!
- ✅ **AWS Bedrock** - Claude, Titan, Llama, Cohere (enterprise managed) ⭐ NEW!
- ✅ **Azure OpenAI** - Enterprise GPT models (private deployment) ⭐ NEW!
- ✅ **Ollama** - Local open-source models (Llama 2, Mistral, CodeLlama, etc.)

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

// Ollama (Local)
import "github.com/zhucl121/langchain-go/core/chat/providers/ollama"
model := ollama.New(ollama.Config{Model: "llama2", BaseURL: "http://localhost:11434"})
```

### 30 Seconds Demo

#### 1. Simple RAG (3 Lines)

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

#### 2. Create ReAct Agent

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/core/agents"
    "github.com/zhucl121/langchain-go/core/tools"
)

func main() {
    // Create tools
    calculator := tools.NewCalculatorTool()
    search := tools.NewDuckDuckGoSearchTool(nil)
    
    // Create agent (1 line)
    agent := agents.CreateReActAgent(llm, []tools.Tool{calculator, search})
    
    // Execute task
    result, _ := agent.Run(context.Background(), 
        "Search today's weather, then calculate square root of 25")
    println(result)
}
```

#### 3. Multi-Agent Collaboration

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/core/agents"
)

func main() {
    // Create coordination strategy
    strategy := agents.NewSequentialStrategy(llm)
    coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)
    system := agents.NewMultiAgentSystem(coordinator, nil)
    
    // Add specialized agents
    researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
    writer := agents.NewWriterAgent("writer", llm, nil)
    
    system.AddAgent("researcher", researcher)
    system.AddAgent("writer", writer)
    
    // Execute complex task
    result, _ := system.Run(context.Background(), 
        "Research Go language latest features and write a tech article")
    println(result)
}
```

#### 4. Vector Stores & Document Loaders ⭐ NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/retrieval/vectorstores"
    "github.com/zhucl121/langchain-go/retrieval/loaders"
)

func main() {
    // Chroma Vector Store
    chromaConfig := vectorstores.ChromaConfig{
        URL:            "http://localhost:8000",
        CollectionName: "docs",
    }
    chromaStore := vectorstores.NewChromaVectorStore(chromaConfig, embedder)
    
    // Qdrant Vector Store (High Performance)
    qdrantConfig := vectorstores.QdrantConfig{
        URL:            "http://localhost:6333",
        CollectionName: "docs",
        VectorSize:     384,
    }
    qdrantStore := vectorstores.NewQdrantVectorStore(qdrantConfig, embedder)
    
    // GitHub Document Loader
    githubConfig := loaders.GitHubLoaderConfig{
        Owner:  "langchain-ai",
        Repo:   "langchain",
        Branch: "main",
        FileExtensions: []string{".md"},
    }
    githubLoader, _ := loaders.NewGitHubLoader(githubConfig)
    docs, _ := githubLoader.LoadDirectory(context.Background(), "docs")
    
    // Confluence Document Loader
    confluenceConfig := loaders.ConfluenceLoaderConfig{
        URL:      "https://your-domain.atlassian.net/wiki",
        Username: "user@example.com",
        APIToken: "your-api-token",
    }
    confluenceLoader, _ := loaders.NewConfluenceLoader(confluenceConfig)
    docs, _ = confluenceLoader.LoadSpace(context.Background(), "SPACE_KEY")
    
    // PostgreSQL Database Loader
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

#### 5. Learning Retrieval System 🔥 v0.4.2 NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/retrieval/learning/feedback"
    "github.com/zhucl121/langchain-go/retrieval/learning/evaluation"
    "github.com/zhucl121/langchain-go/retrieval/learning/optimization"
)

func main() {
    // 1. Collect user feedback
    storage := feedback.NewMemoryStorage()
    collector := feedback.NewCollector(storage)
    
    collector.RecordQuery(ctx, query)
    collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
        Type: feedback.FeedbackTypeRating,
        Rating: 5,
    })
    
    // 2. Evaluate retrieval quality
    evaluator := evaluation.NewEvaluator(collector)
    metrics, _ := evaluator.EvaluateQuery(ctx, queryFeedback)
    fmt.Printf("NDCG: %.3f, MRR: %.3f\n", metrics.NDCG, metrics.MRR)
    
    // 3. Auto-optimize parameters
    optimizer := optimization.NewOptimizer(evaluator, collector, config)
    result, _ := optimizer.Optimize(ctx, strategyID, paramSpace, opts)
    fmt.Printf("Performance improvement: %.2f%%\n", result.Improvement)
    
    // 4. A/B testing validation
    abtestManager := abtest.NewManager(storage)
    analysis, _ := abtestManager.AnalyzeExperiment(ctx, experimentID)
    fmt.Printf("Winner: %s, p-value: %.3f\n", 
        analysis.Winner, analysis.PValue)
}
```

#### 6. MCP Protocol - Claude Desktop Interop 🔥 v0.6.1 NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/pkg/protocols/mcp"
    "github.com/zhucl121/langchain-go/pkg/protocols/mcp/providers"
    "github.com/zhucl121/langchain-go/pkg/protocols/mcp/transport"
)

func main() {
    // Create MCP Server
    server := mcp.NewServer(mcp.ServerConfig{
        Name:    "my-server",
        Version: "1.0.0",
    })
    
    // Register resources
    fsProvider := providers.NewFileSystemProvider("/data/documents")
    server.RegisterResource(&mcp.Resource{
        URI:  "file:///documents",
        Name: "Company Documents",
    }, fsProvider)
    
    // Register tools
    server.RegisterTool(calculatorTool, calculatorHandler)
    
    // Start server (Claude Desktop can connect)
    server.Serve(context.Background(), transport.NewStdioTransport())
}
```

#### 7. A2A Protocol - Standardized Agent Collaboration 🔥 v0.6.1 NEW!

```go
package main

import (
    "context"
    "github.com/zhucl121/langchain-go/pkg/protocols/a2a"
)

func main() {
    // Bridge existing agent
    a2aAgent := a2a.NewA2AAgentBridge(myAgent, &a2a.BridgeConfig{
        Info: &a2a.AgentInfo{
            ID:   "agent-1",
            Name: "Research Agent",
        },
        Capabilities: &a2a.AgentCapabilities{
            Capabilities: []string{"research", "search"},
        },
    })
    
    // Register to registry
    registry := a2a.NewLocalRegistry()
    registry.Register(context.Background(), a2aAgent)
    
    // Smart routing and collaboration
    router := a2a.NewSmartTaskRouter(registry, a2a.RouterConfig{
        Strategy: a2a.StrategyHybrid,
    })
    
    agent, _ := router.Route(context.Background(), task)
    response, _ := agent.SendTask(context.Background(), task)
}
```

## 📊 Performance Comparison

| Feature | Traditional | LangChain-Go |
|---------|------------|--------------|
| RAG App Code | 150+ lines | **3 lines** ⚡ |
| Agent Creation | 50+ lines | **1 line** ⚡ |
| Cache Hit Response | 3-5s | **30-50ns** ⚡ |
| Parallel Tool Execution | Not supported | **3x faster** ⚡ |
| Cost Savings | - | **50-90%** 💰 |

## 🎯 Core Capabilities

### 1. Agent System

- **7 Agent Types**, covering all use cases
- **High-level factory functions**, one-line agent creation
- **Streaming output**, real-time thought process
- **State persistence**, long-running task support
- **Auto-retry**, production-grade error handling

### 2. Multi-Agent Collaboration

- **Message bus**, efficient inter-agent communication
- **3 Coordination strategies**: Sequential, Parallel, Hierarchical
- **6 Specialized Agents**: Coordinator, Researcher, Writer, Reviewer, Analyst, Planner
- **Shared state**, transparent collaboration
- **Execution tracking**, complete history

### 3. Tool Ecosystem

- **38 Built-in tools**, ready to use
- **Tool registry**, dynamic tool management
- **Parallel execution**, 3x performance boost
- **Custom tools**, easy extension
- **Multimodal support**, image, audio, video processing

### 4. RAG Capabilities

- **3-line** complete RAG implementation
- **Learning retrieval**, auto-optimize quality 🔥 v0.4.2 NEW!
- **GraphRAG**, knowledge graph enhanced retrieval
- **Multiple retrievers**, flexible selection
- **5 Vector stores**: Milvus, Chroma, Qdrant, Weaviate, Redis
- **8 Document loaders**: PDF, Word, Excel, HTML, Text, GitHub, Confluence, PostgreSQL
- **Text splitters**, intelligent chunking
- **Hybrid search**, vector + BM25

### 5. Production Features

- **Redis cache**, 50-90% cost savings
- **Auto-retry**, exponential backoff
- **Observability**, OpenTelemetry integration
- **Prometheus metrics**, complete monitoring
- **Structured logging**, easy debugging

## 📖 Documentation

- 📘 [Quick Start](QUICK_START_EN.md) - Get started in 5 minutes
- 📗 [Complete Docs](docs/README.md) - Detailed usage guide
- 🔗 [MCP & A2A Guide](docs/V0.6.1_USER_GUIDE.md) - Protocol integration 🔥 v0.6.1
- 📕 [Agent Guide](docs/guides/agents/README.md) - Agent system docs
- 📙 [Multi-Agent System](docs/guides/multi-agent-guide.md) - Multi-agent collaboration
- 📚 [RAG Guide](docs/guides/rag/README.md) - RAG system docs
- 🧠 [Learning Retrieval Guide](docs/V0.4.2_USER_GUIDE.md) - Learning retrieval
- 🏢 [Enterprise Security Guide](docs/V0.6.0_PROGRESS.md) - RBAC & multi-tenancy v0.6.0
- 💡 [Examples](examples/) - 25 complete examples

## 🔧 Example Programs

Check [examples/](examples/) directory:

**Agent & Multi-Agent**:
- `agent_simple_demo.go` - Simple agent example
- `multi_agent_demo.go` - Multi-agent collaboration
- `plan_execute_agent_demo.go` - Plan-execute agent

**Learning Retrieval (v0.4.2)** 🔥:
- `learning_complete_demo/` - Complete learning retrieval workflow
- `learning_feedback_demo/` - User feedback collection
- `learning_evaluation_demo/` - Retrieval quality evaluation
- `learning_optimization_demo/` - Parameter auto-optimization
- `learning_abtest_demo/` - A/B testing framework
- `learning_postgres_demo/` - PostgreSQL storage

**Multimodal & Tools**:
- `multimodal_demo.go` - Multimodal processing
- `redis_cache_demo.go` - Redis cache usage
- More...

## 🏗️ Architecture

```
langchain-go/
├── core/              # Core functionality
│   ├── agents/       # Agent implementations
│   ├── tools/        # Built-in tools
│   ├── prompts/      # Prompt templates
│   ├── memory/       # Memory system
│   ├── cache/        # Cache layer
│   └── ...
├── graph/            # LangGraph implementation
│   ├── node/         # Graph nodes
│   ├── edge/         # Graph edges
│   ├── checkpoint/   # Checkpointing
│   └── ...
├── retrieval/        # RAG related
│   ├── chains/       # RAG Chains
│   ├── retrievers/   # Retrievers
│   ├── loaders/      # Document loaders
│   └── ...
├── pkg/              # Public packages
│   ├── types/        # Type definitions
│   └── observability/# Observability
└── examples/         # Example code
```

## 🆚 Comparison

### vs Python LangChain

| Feature | Python LangChain | LangChain-Go |
|---------|-----------------|--------------|
| Performance | Slow | **Fast** (compiled) ⚡ |
| Type Safety | Runtime | **Compile-time** ✅ |
| Concurrency | GIL limited | **Native support** 🚀 |
| Deployment | Complex deps | **Single binary** 📦 |
| Memory | High | **Low** 💾 |
| Ecosystem | Rich | Curated |

### Why Go Version?

- ✅ **High Performance**: Compiled language, no GIL limitation
- ✅ **Type Safety**: Compile-time error checking
- ✅ **Concurrency**: Native goroutine support
- ✅ **Easy Deployment**: Single binary
- ✅ **Memory Efficient**: Lower resource usage
- ✅ **Production Ready**: Built-in observability & monitoring

## 📈 Technical Metrics

- **Code Lines**: 40,000+ (v0.6.1 added 3,300+) 🔥
- **Test Coverage**: 85%+
- **Test Cases**: 626+
- **Protocol Support**: 2 (MCP, A2A) 🔥 v0.6.1 NEW!
- **LLM Providers**: 6 (OpenAI, Anthropic, Gemini, Bedrock, Azure, Ollama)
- **Vector Stores**: 5 (Milvus, Chroma, Qdrant, Weaviate, Redis)
- **Document Loaders**: 8 (PDF, Word, Excel, HTML, Text, GitHub, Confluence, PostgreSQL)
- **Built-in Tools**: 38
- **Agent Types**: 7 + 6 specialized agents
- **Learning Modules**: 4 (Feedback, Evaluation, Optimization, A/B Test)
- **Enterprise Features**: 5 (RBAC, Multi-tenancy, Audit, Security, Auth) v0.6.0
- **Doc Pages**: 65+
- **Examples**: 25 (v0.6.1 added 4) 🔥

## 🧪 Testing

```bash
# Start test environment (Redis + Milvus)
make -f Makefile.test test-env-up

# Run all tests
make -f Makefile.test test

# Stop test environment
make -f Makefile.test test-env-down
```

See [Testing Guide](TESTING_EN.md) for details

---

## 🤝 Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### How to Contribute

- 🐛 [Report Bugs](https://github.com/zhucl121/langchain-go/issues)
- 💡 [Request Features](https://github.com/zhucl121/langchain-go/issues)
- 📝 [Improve Docs](https://github.com/zhucl121/langchain-go/pulls)
- 🔧 [Submit Code](https://github.com/zhucl121/langchain-go/pulls)

## 📄 License

MIT License - see [LICENSE](LICENSE)

## 🙏 Acknowledgments

- [LangChain](https://github.com/langchain-ai/langchain) - Original design inspiration
- [LangGraph](https://github.com/langchain-ai/langgraph) - Graph implementation reference
- Go Community - Excellent tools and libraries

## 📞 Community

- **GitHub**: [https://github.com/zhucl121/langchain-go](https://github.com/zhucl121/langchain-go)
- **Issues**: [Report bugs](https://github.com/zhucl121/langchain-go/issues)
- **Discussions**: [Ask questions](https://github.com/zhucl121/langchain-go/discussions)

## ⭐ Star History

If this project helps you, please give it a Star ⭐

---

**Made with ❤️ in Go**

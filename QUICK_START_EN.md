# LangChain-Go Quick Start

🌍 **Language**: [中文](QUICK_START.md) | English

Welcome to LangChain-Go! This guide will get you started in 5 minutes.

---

## 📦 Installation

```bash
go get github.com/zhucl121/langchain-go
```

**System Requirements**:
- Go 1.21 or higher
- (Optional) Docker Desktop - for running tests

---

## 🚀 30-Second Demo

### 1. Simplest Example - Call LLM

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
    // Create OpenAI client
    model, err := openai.New(openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-3.5-turbo",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Send message
    response, err := model.Invoke(context.Background(), []types.Message{
        types.NewUserMessage("Hello, please introduce yourself"),
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Content)
}
```

### 2. Use Local Models - Ollama

```go
import "github.com/zhucl121/langchain-go/core/chat/providers/ollama"

// Use local Ollama model
model := ollama.New(ollama.Config{
    Model:   "llama2",
    BaseURL: "http://localhost:11434",
})
```

### 3. Create Simple Agent

```go
import (
    "github.com/zhucl121/langchain-go/core/agents"
    "github.com/zhucl121/langchain-go/core/tools"
)

// Create tools
calculator := tools.NewCalculatorTool()
search := tools.NewDuckDuckGoSearchTool(nil)

// Create agent in one line
agent := agents.CreateReActAgent(llm, []tools.Tool{calculator, search})

// Execute task
result, _ := agent.Run(context.Background(), 
    "Search today's weather, then calculate square root of 25")
fmt.Println(result)
```

### 4. 3-Line RAG Implementation

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

## 🎯 Core Features Quick Navigation

### Agent System

LangChain-Go provides 7 agent types:

```go
// 1. ReAct Agent - Reasoning and Acting
agent := agents.CreateReActAgent(llm, tools)

// 2. Tool Calling Agent - Function calling
agent := agents.CreateToolCallingAgent(llm, tools)

// 3. OpenAI Functions Agent
agent := agents.CreateOpenAIFunctionsAgent(llm, tools)

// 4. Plan-Execute Agent - Planning and execution
agent := agents.CreatePlanExecuteAgent(llm, tools)

// 5. Self-Ask Agent - Self-questioning
agent := agents.CreateSelfAskAgent(llm, tools)

// 6. Structured Chat Agent - Structured conversation
agent := agents.CreateStructuredChatAgent(llm, tools)

// 7. Conversational Agent - Conversational
agent := agents.CreateConversationalAgent(llm, tools, memory)
```

### Multi-Agent Collaboration

Create multi-agent system for complex tasks:

```go
// Create coordination strategy
strategy := agents.NewSequentialStrategy(llm)
coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

// Create multi-agent system
system := agents.NewMultiAgentSystem(coordinator, nil)

// Add specialized agents
researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
writer := agents.NewWriterAgent("writer", llm, nil)

system.AddAgent("researcher", researcher)
system.AddAgent("writer", writer)

// Execute complex task
result, _ := system.Run(context.Background(), 
    "Research Go language latest features and write a tech article")
```

### Tool Ecosystem

38 built-in tools, ready to use:

```go
// Basic tools
tools.NewCalculatorTool()
tools.NewGetTimeTool()
tools.NewGetDateTool()

// Search tools
tools.NewDuckDuckGoSearchTool(nil)
tools.NewGoogleSearchTool(&googleConfig)

// File tools
tools.NewReadFileTool()
tools.NewWriteFileTool()

// Multimodal tools
tools.NewImageAnalysisTool(config)
tools.NewSpeechToTextTool(config)
tools.NewTextToSpeechTool(config)

// Get all tools
allTools := tools.GetBuiltinTools()
```

### RAG Capabilities

Complete RAG workflow:

```go
// 1. Load documents
loader := loaders.NewPDFLoader("document.pdf")
documents, _ := loader.Load()

// 2. Split text
splitter := splitters.NewCharacterSplitter(1000, 200)
chunks := splitter.SplitDocuments(documents)

// 3. Create vector store
embeddings := embeddings.NewOpenAIEmbeddings(config)
vectorStore := vectorstores.NewMilvusVectorStore(config, embeddings)
vectorStore.AddDocuments(chunks)

// 4. Create RAG Chain
retriever := retrievers.NewVectorStoreRetriever(vectorStore)
ragChain := chains.NewRAGChain(retriever, llm)

// 5. Query
answer, _ := ragChain.Run(context.Background(), "Your question")
```

---

## 🎓 Learning Path

### Beginners (30 minutes)

1. **Installation and Configuration** (5 minutes)
   - Install LangChain-Go
   - Get API Key (OpenAI/Anthropic)

2. **First Agent** (10 minutes)
   - Run `examples/agent_simple_demo.go`
   - Understand how agents work

3. **Using Tools** (15 minutes)
   - Run `examples/search_tools_demo.go`
   - Try different built-in tools

### Intermediate Users (2 hours)

1. **Multi-Agent System** (45 minutes)
   - Run `examples/multi_agent_demo.go`
   - Create custom agents

2. **RAG Application** (45 minutes)
   - Run `examples/pdf_loader_demo.go`
   - Implement document Q&A system

3. **Multimodal Application** (30 minutes)
   - Run `examples/multimodal_demo.go`
   - Process images and audio

### Advanced Users

1. **Deep Dive Docs**
   - Read [Usage Guides](docs/guides/)
   - Learn [LangGraph](docs/guides/langgraph/)

2. **Production Deployment**
   - Configure [Redis Cache](docs/guides/redis-cache.md)
   - Integrate [Observability](docs/advanced/performance.md)

3. **Contributing Code**
   - Check [Contribution Guide](CONTRIBUTING.md)
   - Submit Pull Request

---

## 📖 Example Programs

The project includes 25+ complete examples:

```bash
cd examples

# 1. Simple Agent
go run agent_simple_demo.go

# 2. Multi-Agent Collaboration
go run multi_agent_demo.go

# 3. Multimodal Processing
go run multimodal_demo.go

# 4. Plan-Execute Agent
go run plan_execute_agent_demo.go

# 5. Search Tools
go run search_tools_demo.go

# 6. Self-Ask Agent
go run selfask_agent_demo.go

# 7. Structured Chat
go run structured_chat_demo.go

# 8. PDF Document Loading
go run pdf_loader_demo.go

# 9. Prompt Hub
go run prompt_hub_demo.go

# 10. Redis Cache
go run redis_cache_demo.go

# 11. Advanced Search
go run advanced_search_demo.go
```

**Note**: Set environment variables before running examples:

```bash
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"  # Optional
```

---

## 🔧 Common Tasks

### Switch LLM Provider

```go
// OpenAI
import "github.com/zhucl121/langchain-go/core/chat/providers/openai"
model := openai.New(openai.Config{APIKey: "...", Model: "gpt-4"})

// Claude
import "github.com/zhucl121/langchain-go/core/chat/providers/anthropic"
model := anthropic.New(anthropic.Config{APIKey: "...", Model: "claude-3-sonnet-20240229"})

// Ollama (Local)
import "github.com/zhucl121/langchain-go/core/chat/providers/ollama"
model := ollama.New(ollama.Config{Model: "llama2", BaseURL: "http://localhost:11434"})
```

### Custom Tools

```go
import "github.com/zhucl121/langchain-go/core/tools"

customTool := tools.NewFunctionTool(tools.FunctionToolConfig{
    Name:        "my_custom_tool",
    Description: "This is my custom tool",
    Fn: func(ctx context.Context, input map[string]any) (any, error) {
        // Your tool logic
        return "result", nil
    },
})
```

### Add Memory

```go
import "github.com/zhucl121/langchain-go/core/memory"

// Create memory
memory := memory.NewBufferMemory()

// Use in agent
agent := agents.CreateConversationalAgent(llm, tools, memory)
```

### Enable Caching

```go
import "github.com/zhucl121/langchain-go/core/cache"

// Configure Redis cache
config := cache.DefaultRedisCacheConfig()
config.Password = "your-password"
redisCache, _ := cache.NewRedisCache(config)

// Create LLM cache
llmCache := cache.NewLLMCache(redisCache)

// Using cache in LLM calls can save 50-90% costs
```

---

## 💡 Usage Tips

### 1. Streaming Output

```go
// Agents support streaming output
executor := agents.NewSimplifiedAgentExecutor(agent, tools)
executor.Stream = true

result, _ := executor.Run(ctx, "your task")
```

### 2. Parallel Tool Execution

```go
// Tools automatically execute in parallel, 3x performance boost
executor := tools.NewToolExecutor(tools, nil)
executor.MaxParallel = 5  // Max 5 parallel tools
```

### 3. Error Handling and Retry

```go
// Auto-retry configuration
import "github.com/zhucl121/langchain-go/pkg/types"

retryPolicy := types.RetryPolicy{
    MaxRetries: 3,
    Backoff:    types.ExponentialBackoff,
}

// Agent automatically uses retry policy
```

---

## 📚 More Resources

- 📘 [Complete Docs](docs/) - Detailed usage guide
- 📗 [API Reference](https://pkg.go.dev/github.com/zhucl121/langchain-go) - GoDoc documentation
- 📕 [Example Code](examples/) - 25+ complete examples
- 📙 [Changelog](CHANGELOG.md) - Version update history
- 💡 [Contributing Guide](CONTRIBUTING.md) - How to contribute

---

## ❓ Encountering Issues?

1. **Check docs**: [docs/](docs/)
2. **Run examples**: [examples/](examples/)
3. **Check tests**: Test files are the best usage examples
4. **Submit Issue**: [GitHub Issues](https://github.com/zhucl121/langchain-go/issues)
5. **Join Discussion**: [GitHub Discussions](https://github.com/zhucl121/langchain-go/discussions)

---

## 🎯 Next Steps

- ✅ Run a few example programs to familiarize yourself with basic usage
- ✅ Read [Usage Guides](docs/guides/) for in-depth understanding of core features
- ✅ Build your first AI application
- ✅ Give the project a ⭐ Star to support development!

---

**Happy coding! 🚀**

Feel free to ask questions or check the documentation anytime.

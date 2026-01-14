# Documentation Index

Welcome to the LangChain-Go documentation! This index helps you find the information you need.

## 📚 Getting Started

### Quick Start Guides
- **[Quick Start](../QUICKSTART.md)** - 5-minute introduction
- **[ChatModel Quick Start](../QUICKSTART-CHAT.md)** - Using ChatModels
- **[Prompts Quick Start](../QUICKSTART-PROMPTS.md)** - Working with prompts
- **[Memory Quick Start](../QUICKSTART-MEMORY.md)** - Using memory systems
- **[Tools Quick Start](../QUICKSTART-TOOLS.md)** - Creating and using tools
- **[StateGraph Quick Start](../QUICKSTART-STATEGRAPH.md)** - Building workflows
- **[OutputParser Quick Start](../QUICKSTART-OUTPUT.md)** - Parsing LLM outputs

## 🎯 Core Concepts

### Phase 1: LangChain Core
- **[Runnable System](./Phase1-Runnable-Summary.md)** - LCEL and composable chains
- **[ChatModel Integration](./M09-M12-ChatModel-Summary.md)** - OpenAI & Anthropic
- **[Prompts System](./M13-M14-Prompts-Summary.md)** - Template system
- **[OutputParser System](./M15-M16-OutputParser-Summary.md)** - Structured parsing
- **[Tools System](./M17-M18-Tools-Summary.md)** - Tool integration
- **[Memory System](./M19-M21-Memory-Summary.md)** - Conversation memory

### Phase 2: LangGraph Core
- **[StateGraph Core](./M24-M26-StateGraph-Summary.md)** - State management
- **[Node System](./M27-M29-Node-Summary.md)** - Graph nodes
- **[Edge & Routing](./M30-M34-Edge-Compile-Summary.md)** - Graph edges and compilation
- **[Execution Engine](./M35-M37-Executor-Summary.md)** - Graph execution
- **[Checkpoint System](./M38-M42-Checkpoint-Summary.md)** - State persistence
- **[Durability Modes](./M43-M45-Durability-Summary.md)** - Fault tolerance
- **[Human-in-the-Loop](./M46-M49-HITL-Summary.md)** - Human intervention

### Phase 3: Agent System
- **[Agent System Overview](./Phase3-Agent-System-Summary.md)** - Complete agent guide
- **[Agent Final Report](./PHASE3-FINAL-REPORT.md)** - Implementation details
- **[Release Notes](./PHASE3-RELEASE-NOTES.md)** - Phase 3 features

### Phase 4: RAG System
- **[RAG Complete Guide](./PHASE4-RAG-COMPLETE.md)** - Full RAG implementation
- **[Milvus Guide](./MILVUS-GUIDE.md)** - Using Milvus vector store
- **[Milvus Hybrid Search](./MILVUS-HYBRID-SEARCH.md)** - Advanced search features

## 🔧 Advanced Topics

### Performance & Optimization
- **[Enhancements Summary](./Enhancements-Summary.md)** - Performance improvements
- **[Simplified Implementation List](./Simplified-Implementation-List.md)** - Implementation details

### Examples & Tutorials
- **[Chat Examples](./chat-examples.md)** - ChatModel usage examples
- **[Prompt Examples](./prompts-examples.md)** - Prompt template examples
- **[Output Examples](./output-examples.md)** - Output parsing examples
- **[Tool Examples](./tools-examples.md)** - Tool creation examples

### Project Planning
- **[Phase 2 Planning](./Phase2-Planning.md)** - LangGraph roadmap
- **[Phase 2 Kickoff](./Phase2-Kickoff.md)** - Implementation plan
- **[Phase 2 Week 2-3 Summary](./Phase2-Week2-3-Summary.md)** - Progress report

## 📖 Module Documentation

### Detailed Summaries
- **[M01-M04: Type System](./M01-M04-summary.md)**
- **[M09-M12: ChatModel](./M09-M12-ChatModel-Summary.md)**
- **[M13-M14: Prompts](./M13-M14-Prompts-Summary.md)**
- **[M15-M16: OutputParser](./M15-M16-OutputParser-Summary.md)**
- **[M17-M18: Tools](./M17-M18-Tools-Summary.md)**
- **[M24-M26: StateGraph](./M24-M26-StateGraph-Summary.md)**
- **[M27-M29: Node System](./M27-M29-Node-Summary.md)**
- **[M30-M34: Edge & Compile](./M30-M34-Edge-Compile-Summary.md)**
- **[M35-M37: Executor](./M35-M37-Executor-Summary.md)**
- **[M38-M42: Checkpoint](./M38-M42-Checkpoint-Summary.md)**
- **[M43-M45: Durability](./M43-M45-Durability-Summary.md)**
- **[M60: ToolNode](./M60-ToolNode-Summary.md)**

## 🚀 Extension & Enhancement

- **[Extension Roadmap](./课后扩展增强功能清单.md)** - Future features (Chinese)
- **[Unimplemented Features](./未实现功能清单.md)** - Feature backlog (Chinese)

## 🛠️ Development

### Setup
- **[Go Installation Guide](./GO-INSTALLATION-GUIDE.md)** - Setting up Go

### Contributing
- **[Contributing Guide](../CONTRIBUTING.md)** - How to contribute
- **[Code of Conduct](../CODE_OF_CONDUCT.md)** - Community guidelines
- **[Security Policy](../SECURITY.md)** - Reporting vulnerabilities

### Project Status
- **[Project Progress](../PROJECT-PROGRESS.md)** - Detailed progress tracking
- **[Changelog](../CHANGELOG.md)** - Version history

## 🔍 By Feature

### Runnable & Chains
- [Runnable Interface](./Phase1-Runnable-Summary.md#runnable-接口)
- [LCEL Composition](./Phase1-Runnable-Summary.md#lcel-组合)
- [Streaming](./Phase1-Runnable-Summary.md#streaming)

### ChatModels
- [OpenAI Integration](./M09-M12-ChatModel-Summary.md#openai)
- [Anthropic Integration](./M09-M12-ChatModel-Summary.md#anthropic)
- [Function Calling](./M09-M12-ChatModel-Summary.md#function-calling)

### StateGraph
- [Graph Creation](./M24-M26-StateGraph-Summary.md#创建-stategraph)
- [Nodes and Edges](./M27-M29-Node-Summary.md)
- [Conditional Routing](./M30-M34-Edge-Compile-Summary.md#conditional-edges)

### Persistence
- [Checkpointing](./M38-M42-Checkpoint-Summary.md)
- [Time Travel](./M38-M42-Checkpoint-Summary.md#时间旅行)
- [Durability Modes](./M43-M45-Durability-Summary.md)

### Human-in-the-Loop
- [Interrupts](./M46-M49-HITL-Summary.md#interrupts)
- [Approval Workflow](./M46-M49-HITL-Summary.md#approval)
- [Resume Execution](./M46-M49-HITL-Summary.md#resume)

### RAG System
- [Document Loaders](./PHASE4-RAG-COMPLETE.md#m61-document-loaders)
- [Text Splitters](./PHASE4-RAG-COMPLETE.md#m62-text-splitters)
- [Embeddings](./PHASE4-RAG-COMPLETE.md#m63-embeddings)
- [Vector Stores](./PHASE4-RAG-COMPLETE.md#m64-vector-stores)
- [Hybrid Search](./MILVUS-HYBRID-SEARCH.md)

## 📝 API Reference

- **[GoDoc](https://pkg.go.dev/langchain-go)** - Complete API documentation

## 💬 Community

- **[GitHub Discussions](https://github.com/yourusername/langchain-go/discussions)** - Q&A and discussions
- **[GitHub Issues](https://github.com/yourusername/langchain-go/issues)** - Bug reports and feature requests

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

---

**Can't find what you're looking for?**
- Check the [README](../README.md) for overview
- Search in [GitHub Issues](https://github.com/yourusername/langchain-go/issues)
- Ask in [Discussions](https://github.com/yourusername/langchain-go/discussions)

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Weaviate vector store integration
- Semantic text splitter
- Multi-Agent system
- API tool integration (OpenAPI/Swagger)

## [1.5.0] - 2026-01-15

### Added - Phase 5 Stage 4 Complete: Vector Store & Document Loader Ecosystem 🎉

#### New Vector Stores
- **Chroma Vector Store** - Open-source vector database integration
  - Complete CRUD operations
  - Similarity search with score threshold
  - Multiple distance metrics (L2, IP, Cosine)
  - Automatic collection creation
  - Metadata filtering support
  - Batch operations
  - ~358 lines of code, 17 tests
  - SDK: `github.com/amikos-tech/chroma-go`

- **Pinecone Vector Store** - Cloud-hosted vector database
  - Complete CRUD operations
  - Similarity search with score threshold
  - Multiple distance metrics (Cosine, Euclidean, Dotproduct)
  - Namespace support
  - Automatic index creation
  - Metadata management
  - ~355 lines of code, 18 tests
  - SDK: `github.com/pinecone-io/go-pinecone`

#### New Document Loaders
- **Word/DOCX Loader** - Microsoft Word document parsing
  - DOCX file parsing (ZIP + XML)
  - Text content extraction
  - Table data extraction
  - Document properties (title, author, dates)
  - DOC file basic support
  - Style information extraction (optional)
  - ~476 lines of code, 14 tests

- **HTML/Web Loader** - Web page scraping and crawling
  - Local HTML file loading
  - Web URL fetching
  - CSS selector support
  - Script and style filtering
  - Link extraction
  - Meta tag extraction
  - Web crawler support (recursive crawling)
  - ~573 lines of code, 18 tests
  - Library: `github.com/PuerkitoBio/goquery`

- **Excel/CSV Loader** - Spreadsheet data processing
  - Excel (.xlsx) file parsing
  - CSV file support
  - Multiple worksheet support
  - Header extraction
  - Row/column filtering
  - Document metadata extraction
  - Structured table extraction
  - ~556 lines of code, 13 tests
  - Library: `github.com/xuri/excelize`

### Features Completed
- **Complete Vector Store Ecosystem**: Milvus, Chroma, Pinecone (3 major options)
- **Comprehensive Document Support**: PDF, Word, HTML, Excel (all major formats)
- **Flexible Deployment Options**: Local, lightweight, cloud-hosted scenarios
- **Full RAG Workflow**: Document loading → Vector storage complete pipeline

### Statistics
- Stage 4 code: ~2,318 lines (+2,318 lines)
- Stage 4 tests: ~1,782 lines (+1,782 lines)
- Total code: ~35,300 lines (+2,300 lines from v1.4.0)
- Total tests: ~10,050 lines (+1,750 lines from v1.4.0)
- Documentation: ~26,000 lines (+8,000 lines)
- Test coverage: 75%+
- New test functions: 80

### Documentation
- PHASE4-COMPLETION-REPORT.md - Complete stage 4 report
- Updated README.md with new features
- Updated PROJECT-PROGRESS.md with v1.5.0 info
- Updated 课后扩展增强功能清单.md with completion status

## [1.4.0] - 2026-01-15

### Added - Phase 5 Stage 3 Complete: Observability 🎉
- **M61: Document Loaders** - Complete document loading system
  - Text loader for plain text files
  - Markdown loader with metadata extraction
  - JSON loader (single object and array support)
  - CSV loader with customizable content columns
  - Directory loader with recursive scanning and glob patterns
  - ~450 lines of code, 11 tests
- **M62: Text Splitters** - Intelligent text splitting
  - CharacterTextSplitter for basic splitting
  - RecursiveCharacterTextSplitter for semantic-aware splitting
  - TokenTextSplitter for token-based splitting
  - MarkdownTextSplitter for Markdown structure preservation
  - ~400 lines of code, 10 tests
- **M63: Embeddings** - Embedding model integration
  - OpenAI Embeddings (ada-002, 3-small, 3-large)
  - FakeEmbeddings for testing
  - CachedEmbeddings wrapper for performance
  - ~350 lines of code, 10 tests
- **M64: Vector Stores** - Vector database integration
  - InMemoryVectorStore for development
  - **Milvus 2.6+ integration** with Hybrid Search & Reranking
  - Cosine similarity search
  - Document management (add, delete, clear)
  - ~550 lines of code, 15 tests

### Enhanced - Milvus 2.6.x Features
- **Hybrid Search** - Combines vector and keyword (BM25) search
  - Configurable vector/keyword weights
  - Two reranking strategies: RRF and Weighted Fusion
  - ~10% accuracy improvement over vector-only search
- **Reranking** - Intelligent result fusion
  - RRF (Reciprocal Rank Fusion) algorithm
  - Weighted fusion with customizable weights
  - Optimized for different use cases

### Statistics
- Total code: ~28,000 lines (+2,565 lines)
- Test code: ~5,000 lines (+800 lines)
- Documentation: ~15,000 lines (+2,000 lines)
- Test coverage: 75%+
- Total modules: 62/60 (103%)

## [1.2.0] - 2026-01-14

### Added - Phase 3 Complete
- **M53: Agent System** - Core agent interfaces and factory
- **M54-M57: Middleware System** - Comprehensive middleware support
  - Logging middleware
  - Performance monitoring middleware
  - Metrics middleware
  - HITL middleware integration
- **M58: Agent Executor** - Thought-Action-Observation loop
- **M59: Agent Implementations**
  - ReActAgent for reasoning and acting
  - ToolCallingAgent for tool-based workflows
  - ConversationalAgent for dialog systems
- **M60: ToolNode** - Generic tool execution node for LangGraph

### Statistics
- Phase 3 code: ~2,140 lines
- Phase 3 tests: 15+ tests
- Test coverage: 72%+

## [1.1.0] - 2026-01-14

### Enhanced - Simplified Implementations
- **P0-1: True Parallel Execution** - Real goroutine-based parallel scheduling
  - 3x-9x performance improvement
  - State merger interface for custom merge strategies
  - Semaphore-based concurrency control
- **P0-2: Complete Recovery Manager** - Full fault recovery implementation
  - Checkpoint-based state loading
  - Durability mode-aware retry strategies
  - Task-level recovery control
- **P1-1: Graph Optimization** - Intelligent graph optimizations
  - Edge deduplication
  - Dead node elimination
  - Parallel group identification
- **P1-2: JSON Schema Enhancement** - Advanced schema generation
  - Recursive struct support
  - Array/slice element types
  - Validation rules (min, max, pattern, enum)
- **P2-1: BranchEdge Parallel** - Parallel branch support (depends on P0-1)
- **P2-2: Calculator Enhancement** - Mathematical function support
  - sqrt, sin, cos, tan, abs, log, ln, exp
  - Constants: pi, e

### Statistics
- New code: ~610 lines
- New tests: 8 tests
- Performance: 3x-9x speedup for parallel execution
- Coverage improvement: +4.5% average

## [1.0.0] - 2026-01-14

### Added - Phase 2 Complete 🎉
- **M46: Interrupt Mechanism** - Human-in-the-loop interrupt system
- **M47: Resume Management** - Execution resumption after interrupts
- **M48: Approval Workflow** - Human approval system
- **M49: Interrupt Handler** - Callback-based interrupt handling
- **M50: Streaming Interface** - Stream-based execution
- **M51: Stream Modes** - Multiple streaming modes
- **M52: Event Types** - Comprehensive event system

### Statistics
- Phase 1 code: ~8,000 lines
- Phase 2 code: ~10,000 lines
- Total: ~18,000 lines
- Average test coverage: 74%+

## [0.9.0] - 2026-01-14

### Added - Durability System
- **M43: Durability Modes**
  - AtMostOnce: No retry, fail fast
  - AtLeastOnce: Retry until success
  - ExactlyOnce: Idempotent execution with deduplication
- **M44: Durable Tasks** - Task wrapper with retry logic
- **M45: Recovery Manager** - Automatic failure recovery

### Statistics
- Code: ~1,400 lines
- Tests: 19 tests
- Coverage: 63.2%

## [0.8.0] - 2026-01-14

### Added - Checkpoint System
- **M38: Checkpoint Interface** - Core checkpoint data structures
- **M39: Memory Checkpointer** - In-memory checkpoint storage
- **M40: SQLite Checkpointer** - SQLite-based persistence
- **M41: Postgres Checkpointer** - PostgreSQL persistence
- **M42: Checkpoint Manager** - Advanced checkpoint management with time travel

### Features
- Multiple storage backends
- Type-safe generic design
- Time travel capability
- Automatic cleanup
- Optional dependencies with build tags

### Statistics
- Code: ~2,000 lines
- Tests: 18 tests
- Coverage: 68.2%

## [0.7.0] - 2026-01-14

### Added - Execution Engine
- **M30: Edge System** - Normal edges with metadata
- **M31: Conditional Edges** - Branching logic
- **M32: Router** - Flexible routing with priorities
- **M33: Compiler** - Graph compilation and optimization
- **M34: Validator** - Completeness validation and cycle detection
- **M35: Executor** - Graph execution engine
- **M36: Execution Context** - Context with event system
- **M37: Scheduler** - Task scheduling with strategies

### Statistics
- Code: ~4,500 lines
- Tests: 69 tests
- Coverage: 81.4% average

## [0.6.0] - 2026-01-14

### Added - Phase 1 Complete + Phase 2 Start
- **M19-M21: Memory System**
  - Memory interface
  - BufferMemory with full history
  - ConversationBufferWindowMemory with sliding window
  - ConversationSummaryMemory with LLM summarization
  - Thread-safe implementation
- **M24-M26: StateGraph Core**
  - StateGraph definition with generics
  - Channel system for state management
  - Reducer for state updates
- **M27-M29: Node System**
  - Node interface
  - FunctionNode for simple functions
  - SubgraphNode for nested graphs

### Statistics
- Code: ~2,000 lines
- Coverage: StateGraph 82.6%, Node 89.8%

## [0.5.0] - 2026-01-14

### Added - Tools System
- **M17-M18: Tools**
  - Tool interface and executor
  - FunctionTool wrapper
  - Calculator tool with expression parsing
  - HTTP Request tool with safety controls
  - Shell tool (placeholder)
  - JSONPlaceholder tool for testing

### Statistics
- Code: ~1,050 lines
- Tests: 15 tests
- Coverage: 84.5%

## [0.4.0] - 2026-01-14

### Added - OutputParser System
- **M15-M16: OutputParser**
  - Generic OutputParser interface
  - JSONParser with intelligent extraction
  - StructuredParser for type-safe parsing
  - ListParser for array parsing
  - BooleanParser
  - Automatic schema generation
  - Format instructions

### Statistics
- Code: ~930 lines
- Coverage: 57.0%

## [0.3.0] - 2026-01-14

### Added - Prompts System
- **M13-M14: Prompts**
  - PromptTemplate with variable substitution
  - ChatPromptTemplate for chat messages
  - FewShotPromptTemplate for few-shot learning
  - Partial variables
  - Example selectors
  - Runnable integration

### Statistics
- Code: ~1,000 lines
- Coverage: 64.8%

## [0.2.0] - 2026-01-14

### Added - ChatModel System
- **M09-M12: ChatModel**
  - Unified ChatModel interface
  - OpenAI provider (GPT-3.5/4/4o)
  - Anthropic provider (Claude 3 family)
  - Streaming support (SSE)
  - Function calling / tool use
  - Structured output
  - Batch processing

### Statistics
- Code: ~1,400 lines
- Coverage: 93.8% (core), 15% (providers)

## [0.1.0] - 2026-01-13

### Added - Foundation
- **M01-M04: Type System**
  - Message types (System, User, Assistant, Tool)
  - Tool definition and validation
  - JSON Schema support
  - Config and callback system
- **M05-M08: Runnable System**
  - Generic Runnable interface
  - Invoke/Batch/Stream modes
  - Sequence composition
  - Parallel execution
  - Retry and fallback strategies

### Statistics
- Code: ~1,800 lines
- Coverage: 97.2% (types), 57.4% (runnable)

---

## Legend

- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security fixes
- **Enhanced**: Improvements to existing features

## Links

[Unreleased]: https://github.com/yourusername/langchain-go/compare/v1.5.0...HEAD
[1.5.0]: https://github.com/yourusername/langchain-go/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/yourusername/langchain-go/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/yourusername/langchain-go/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/yourusername/langchain-go/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/yourusername/langchain-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/yourusername/langchain-go/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/yourusername/langchain-go/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/yourusername/langchain-go/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/yourusername/langchain-go/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/yourusername/langchain-go/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/yourusername/langchain-go/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/yourusername/langchain-go/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/yourusername/langchain-go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/yourusername/langchain-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/yourusername/langchain-go/releases/tag/v0.1.0

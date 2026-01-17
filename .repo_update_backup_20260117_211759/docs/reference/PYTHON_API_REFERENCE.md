# Python LangChain API 参考 - Go 实现指南

## 📅 创建日期: 2026-01-16

本文档详细对照 Python LangChain v1.0+ 的 API,为 Go 实现提供参考。

---

## 🎯 实现原则

1. **直接借鉴最佳实践** - 不重复 Python 的弯路
2. **保持 Go 风格** - 使用 Go 惯用法
3. **类型安全优先** - 充分利用 Go 的类型系统
4. **性能优化** - 发挥 Go 的性能优势

---

## 📦 Part 1: RAG Chain 实现参考

### Python API (v1.0+)

#### 核心函数
```python
from langchain.chains import (
    create_retrieval_chain,
    create_stuff_documents_chain,
    create_history_aware_retriever
)

# 1. 基础 RAG Chain
combine_docs_chain = create_stuff_documents_chain(llm, prompt)
rag_chain = create_retrieval_chain(retriever, combine_docs_chain)

result = rag_chain.invoke({"input": "question"})
# 返回: {"input": "...", "context": [...], "answer": "..."}

# 2. 对话式 RAG
history_retriever = create_history_aware_retriever(llm, retriever, prompt)
rag_chain = create_retrieval_chain(history_retriever, combine_docs_chain)

result = rag_chain.invoke({
    "input": "question",
    "chat_history": [...]
})

# 3. 流式 RAG
async for event in rag_chain.astream_events({"input": "question"}):
    if event["kind"] == "on_chat_model_stream":
        print(event["data"]["chunk"].content, end="")
```

---

### Go 实现建议

**目录结构**:
```
langchain-go/retrieval/chains/
├── rag.go              # RAG Chain 核心
├── qa.go               # QA Chain (简化版)
├── conversational.go   # 对话式 RAG
├── types.go            # 类型定义
└── rag_test.go         # 测试
```

**核心 API**:
```go
// retrieval/chains/rag.go
package chains

import (
    "context"
    "github.com/zhucl121/langchain-go/core/chat"
    "github.com/zhucl121/langchain-go/core/prompts"
    "github.com/zhucl121/langchain-go/retrieval/retrievers"
)

// RAGChain RAG 链
type RAGChain struct {
    retriever retrievers.Retriever
    llm       chat.ChatModel
    prompt    *prompts.PromptTemplate
    config    RAGConfig
}

// RAGConfig 配置
type RAGConfig struct {
    ReturnSources  bool
    ScoreThreshold float32
    MaxContextLen  int
}

// RAGResult 结果
type RAGResult struct {
    Question    string
    Answer      string
    Context     []*loaders.Document
    Confidence  float64
    TimeElapsed time.Duration
}

// NewRAGChain 创建 RAG Chain
func NewRAGChain(retriever retrievers.Retriever, llm chat.ChatModel, opts ...Option) *RAGChain {
    chain := &RAGChain{
        retriever: retriever,
        llm:       llm,
        prompt:    prompts.DefaultRAGPrompt, // 默认模板
        config:    DefaultRAGConfig,
    }
    
    for _, opt := range opts {
        opt(chain)
    }
    
    return chain
}

// Option 配置选项
type Option func(*RAGChain)

func WithPrompt(prompt *prompts.PromptTemplate) Option {
    return func(c *RAGChain) { c.prompt = prompt }
}

func WithScoreThreshold(threshold float32) Option {
    return func(c *RAGChain) { c.config.ScoreThreshold = threshold }
}

// Run 执行 RAG
func (c *RAGChain) Run(ctx context.Context, question string) (RAGResult, error) {
    start := time.Now()
    
    // 1. 检索相关文档
    docs, err := c.retriever.GetRelevantDocuments(ctx, question)
    if err != nil {
        return RAGResult{}, fmt.Errorf("retrieval failed: %w", err)
    }
    
    // 2. 过滤低分文档
    var relevantDocs []*loaders.Document
    for _, doc := range docs {
        // 假设 doc 有 Score 字段或从 Retriever 获取
        relevantDocs = append(relevantDocs, doc)
    }
    
    // 3. 构建上下文
    contextStr := c.buildContext(relevantDocs)
    
    // 4. 格式化 prompt
    promptStr, err := c.prompt.Format(map[string]interface{}{
        "context":  contextStr,
        "question": question,
    })
    if err != nil {
        return RAGResult{}, err
    }
    
    // 5. 调用 LLM
    messages := []types.Message{
        types.NewUserMessage(promptStr),
    }
    response, err := c.llm.Invoke(ctx, messages)
    if err != nil {
        return RAGResult{}, fmt.Errorf("LLM invocation failed: %w", err)
    }
    
    // 6. 构建结果
    return RAGResult{
        Question:    question,
        Answer:      response.Content,
        Context:     relevantDocs,
        Confidence:  c.calculateConfidence(docs),
        TimeElapsed: time.Since(start),
    }, nil
}

// Stream 流式执行
func (c *RAGChain) Stream(ctx context.Context, question string) (<-chan RAGChunk, error) {
    resultChan := make(chan RAGChunk)
    
    go func() {
        defer close(resultChan)
        
        // 1. 检索 (发送 retrieval 事件)
        docs, _ := c.retriever.GetRelevantDocuments(ctx, question)
        resultChan <- RAGChunk{Type: "retrieval", Data: docs}
        
        // 2. 构建 prompt
        // ...
        
        // 3. 流式 LLM 调用
        streamChan, _ := c.llm.Stream(ctx, messages)
        for event := range streamChan {
            resultChan <- RAGChunk{
                Type: "llm_token",
                Data: event.Data.Content,
            }
        }
    }()
    
    return resultChan, nil
}

// Batch 批量执行
func (c *RAGChain) Batch(ctx context.Context, questions []string) ([]RAGResult, error) {
    results := make([]RAGResult, len(questions))
    
    // 可以并行处理
    var wg sync.WaitGroup
    for i, q := range questions {
        wg.Add(1)
        go func(idx int, question string) {
            defer wg.Done()
            result, err := c.Run(ctx, question)
            if err == nil {
                results[idx] = result
            }
        }(i, q)
    }
    wg.Wait()
    
    return results, nil
}

// buildContext 构建上下文
func (c *RAGChain) buildContext(docs []*loaders.Document) string {
    var builder strings.Builder
    for i, doc := range docs {
        builder.WriteString(fmt.Sprintf("\n[文档 %d]\n%s\n", i+1, doc.Content))
    }
    return builder.String()
}

// calculateConfidence 计算置信度
func (c *RAGChain) calculateConfidence(docs []*loaders.Document) float64 {
    // 基于检索分数计算
    // 可以使用平均分、最高分、加权平均等
    return 0.8 // 示例
}
```

**使用示例**:
```go
// 3 行代码完成 RAG
retriever := retrievers.NewVectorStoreRetriever(vectorStore)
ragChain := chains.NewRAGChain(retriever, llm)
result, _ := ragChain.Run(ctx, "What is LangChain?")

// 带配置
ragChain := chains.NewRAGChain(retriever, llm,
    chains.WithScoreThreshold(0.7),
    chains.WithPrompt(customPrompt),
)
```

---

## 📦 Part 2: Retriever 实现参考

### Python API (v1.0+)

#### 核心接口
```python
from langchain.retrievers import BaseRetriever

class BaseRetriever:
    def get_relevant_documents(self, query: str) -> List[Document]:
        """获取相关文档"""
        
    def invoke(self, input: str, config: RunnableConfig = None) -> List[Document]:
        """Runnable 接口"""
```

#### MultiQueryRetriever
```python
from langchain.retrievers import MultiQueryRetriever

retriever = MultiQueryRetriever.from_llm(
    retriever=base_retriever,
    llm=llm,
    include_original=True
)

docs = retriever.invoke("question")
```

#### EnsembleRetriever
```python
from langchain.retrievers import EnsembleRetriever

ensemble = EnsembleRetriever(
    retrievers=[bm25_retriever, vector_retriever],
    weights=[0.5, 0.5],
    c=60  # RRF constant
)

docs = ensemble.invoke("question")
```

---

### Go 实现建议

**目录结构**:
```
langchain-go/retrieval/retrievers/
├── retriever.go          # 核心接口
├── vector_store.go       # VectorStore 适配器
├── multi_query.go        # 多查询检索器
├── ensemble.go           # 集成检索器
├── compression.go        # 压缩检索器
├── types.go              # 类型定义
└── retriever_test.go     # 测试
```

**核心 API**:
```go
// retrieval/retrievers/retriever.go
package retrievers

import (
    "context"
    "github.com/zhucl121/langchain-go/loaders"
)

// Retriever 检索器接口
type Retriever interface {
    // GetRelevantDocuments 获取相关文档
    GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error)
    
    // GetRelevantDocumentsWithScore 带分数获取
    GetRelevantDocumentsWithScore(ctx context.Context, query string) ([]DocumentWithScore, error)
}

// DocumentWithScore 带分数的文档
type DocumentWithScore struct {
    Document *loaders.Document
    Score    float32
}

// BaseRetriever 基础检索器 (可选,提供通用功能)
type BaseRetriever struct {
    callbacks []Callback
    metadata  map[string]interface{}
}
```

**VectorStoreRetriever**:
```go
// retrieval/retrievers/vector_store.go
package retrievers

type VectorStoreRetriever struct {
    *BaseRetriever
    vectorStore    vectorstores.VectorStore
    searchType     SearchType
    k              int
    scoreThreshold float32
    filter         map[string]interface{}
}

type SearchType string

const (
    SearchSimilarity SearchType = "similarity"
    SearchMMR        SearchType = "mmr"
    SearchHybrid     SearchType = "hybrid"
)

// NewVectorStoreRetriever 创建向量存储检索器
func NewVectorStoreRetriever(
    store vectorstores.VectorStore,
    searchType SearchType,
    k int,
    opts ...RetrieverOption,
) *VectorStoreRetriever {
    retriever := &VectorStoreRetriever{
        BaseRetriever:  &BaseRetriever{},
        vectorStore:    store,
        searchType:     searchType,
        k:              k,
        scoreThreshold: 0.0,
    }
    
    for _, opt := range opts {
        opt(retriever)
    }
    
    return retriever
}

// GetRelevantDocuments 实现 Retriever 接口
func (r *VectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error) {
    switch r.searchType {
    case SearchSimilarity:
        return r.vectorStore.SimilaritySearch(ctx, query, r.k)
    case SearchMMR:
        return r.vectorStore.MMRSearch(ctx, query, r.k)
    case SearchHybrid:
        results, err := r.vectorStore.HybridSearch(ctx, query, r.k, nil)
        if err != nil {
            return nil, err
        }
        docs := make([]*loaders.Document, len(results))
        for i, r := range results {
            docs[i] = r.Document
        }
        return docs, nil
    default:
        return r.vectorStore.SimilaritySearch(ctx, query, r.k)
    }
}
```

**MultiQueryRetriever**:
```go
// retrieval/retrievers/multi_query.go
package retrievers

import (
    "context"
    "github.com/zhucl121/langchain-go/core/chat"
    "github.com/zhucl121/langchain-go/core/prompts"
)

type MultiQueryRetriever struct {
    *BaseRetriever
    baseRetriever   Retriever
    llm             chat.ChatModel
    prompt          *prompts.PromptTemplate
    includeOriginal bool
}

// NewMultiQueryRetriever 创建多查询检索器
func NewMultiQueryRetriever(
    baseRetriever Retriever,
    llm chat.ChatModel,
    opts ...MultiQueryOption,
) *MultiQueryRetriever {
    return &MultiQueryRetriever{
        BaseRetriever:   &BaseRetriever{},
        baseRetriever:   baseRetriever,
        llm:             llm,
        prompt:          DefaultMultiQueryPrompt,
        includeOriginal: false,
    }
}

// GetRelevantDocuments 实现接口
func (r *MultiQueryRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error) {
    // 1. 使用 LLM 生成多个查询变体
    queries, err := r.generateQueries(ctx, query)
    if err != nil {
        return nil, err
    }
    
    if r.includeOriginal {
        queries = append([]string{query}, queries...)
    }
    
    // 2. 对每个查询检索
    allDocs := make(map[string]*loaders.Document) // 去重
    for _, q := range queries {
        docs, err := r.baseRetriever.GetRelevantDocuments(ctx, q)
        if err != nil {
            continue
        }
        
        for _, doc := range docs {
            // 使用内容作为 key 去重
            key := doc.Content
            if _, exists := allDocs[key]; !exists {
                allDocs[key] = doc
            }
        }
    }
    
    // 3. 返回去重后的文档
    result := make([]*loaders.Document, 0, len(allDocs))
    for _, doc := range allDocs {
        result = append(result, doc)
    }
    
    return result, nil
}

// generateQueries 生成查询变体
func (r *MultiQueryRetriever) generateQueries(ctx context.Context, query string) ([]string, error) {
    // 使用 LLM 生成 3-5 个查询变体
    promptStr, _ := r.prompt.Format(map[string]interface{}{
        "question": query,
    })
    
    messages := []types.Message{types.NewUserMessage(promptStr)}
    response, err := r.llm.Invoke(ctx, messages)
    if err != nil {
        return nil, err
    }
    
    // 解析生成的查询列表
    queries := parseQueries(response.Content)
    return queries, nil
}

// DefaultMultiQueryPrompt 默认多查询 prompt
var DefaultMultiQueryPrompt = prompts.NewPromptTemplate(`
你是一个 AI 助手,帮助生成多个搜索查询。

用户问题: {{.question}}

请生成 3 个相关但措辞不同的搜索查询,以便从不同角度检索相关信息。
每个查询一行,不需要编号。

查询列表:
`, []string{"question"})
```

**EnsembleRetriever**:
```go
// retrieval/retrievers/ensemble.go
package retrievers

type EnsembleRetriever struct {
    *BaseRetriever
    retrievers []Retriever
    weights    []float64
    rrfK       int
}

// NewEnsembleRetriever 创建集成检索器
func NewEnsembleRetriever(retrievers []Retriever, opts ...EnsembleOption) *EnsembleRetriever {
    // 默认等权重
    weights := make([]float64, len(retrievers))
    for i := range weights {
        weights[i] = 1.0 / float64(len(retrievers))
    }
    
    return &EnsembleRetriever{
        BaseRetriever: &BaseRetriever{},
        retrievers:    retrievers,
        weights:       weights,
        rrfK:          60, // 默认 RRF k=60
    }
}

// GetRelevantDocuments 实现接口
func (r *EnsembleRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error) {
    // 1. 从所有检索器获取结果
    var allResults [][]DocumentWithScore
    for _, retriever := range r.retrievers {
        docs, err := retriever.GetRelevantDocumentsWithScore(ctx, query)
        if err != nil {
            continue
        }
        allResults = append(allResults, docs)
    }
    
    // 2. 使用 RRF 融合
    fusedResults := r.applyRRF(allResults)
    
    // 3. 转换为文档列表
    docs := make([]*loaders.Document, len(fusedResults))
    for i, result := range fusedResults {
        docs[i] = result.Document
    }
    
    return docs, nil
}

// applyRRF 应用 Reciprocal Rank Fusion
func (r *EnsembleRetriever) applyRRF(resultSets [][]DocumentWithScore) []DocumentWithScore {
    docScores := make(map[string]*scoredDoc)
    
    // 遍历每个结果集
    for setIdx, results := range resultSets {
        weight := r.weights[setIdx]
        
        for rank, docWithScore := range results {
            key := docWithScore.Document.Content
            
            if _, exists := docScores[key]; !exists {
                docScores[key] = &scoredDoc{
                    doc:   docWithScore.Document,
                    score: 0,
                }
            }
            
            // RRF 公式: weight / (k + rank)
            rrfScore := weight / float64(r.rrfK+rank+1)
            docScores[key].score += float32(rrfScore)
        }
    }
    
    // 排序并返回
    var results []DocumentWithScore
    for _, sd := range docScores {
        results = append(results, DocumentWithScore{
            Document: sd.doc,
            Score:    sd.score,
        })
    }
    
    // 按分数降序排序
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    return results
}
```

---

## 📦 Part 3: Agent 实现参考

### Python API (v1.0+)

#### 推荐方式 (LangGraph-based)
```python
from langchain.agents import create_agent

agent = create_agent(
    model=llm,
    tools=[tool1, tool2],
    system_prompt="You are a helpful assistant"
)

result = agent.invoke({"messages": [("user", "task")]})
```

#### Legacy 方式 (AgentExecutor)
```python
from langchain.agents import create_tool_calling_agent, AgentExecutor

agent = create_tool_calling_agent(llm, tools, prompt)
executor = AgentExecutor(
    agent=agent,
    tools=tools,
    max_iterations=10,
    max_execution_time=300,
    handle_parsing_errors=True,
    verbose=True
)

result = executor.invoke({"input": "task"})
```

---

### Go 实现建议

**目录结构**:
```
langchain-go/core/agents/
├── agent.go          # Agent 接口
├── factory.go        # 创建函数 (CreateAgent)
├── react.go          # ReAct Agent
├── executor.go       # Agent 执行器
├── parsers.go        # Action 解析器
├── types.go          # 类型定义
└── agent_test.go     # 测试
```

**核心 API**:
```go
// core/agents/agent.go
package agents

// Agent Agent 接口
type Agent interface {
    // Run 执行 Agent
    Run(ctx context.Context, input string) (AgentResult, error)
    
    // Stream 流式执行
    Stream(ctx context.Context, input string) (<-chan AgentEvent, error)
}

// AgentResult Agent 执行结果
type AgentResult struct {
    Input        string
    Output       string
    Steps        []AgentStep
    TotalTokens  int
    TimeElapsed  time.Duration
    FinishReason FinishReason
}

// AgentStep Agent 执行步骤
type AgentStep struct {
    Action      AgentAction
    Observation string
    Thought     string
}

// AgentAction Agent 动作
type AgentAction struct {
    Tool      string
    ToolInput map[string]interface{}
    Log       string
}

// FinishReason 完成原因
type FinishReason string

const (
    FinishReasonCompleted     FinishReason = "completed"
    FinishReasonMaxIterations FinishReason = "max_iterations"
    FinishReasonTimeout       FinishReason = "timeout"
    FinishReasonError         FinishReason = "error"
)
```

**CreateAgent 工厂函数**:
```go
// core/agents/factory.go
package agents

// AgentType Agent 类型
type AgentType string

const (
    AgentTypeReAct       AgentType = "react"
    AgentTypeToolCalling AgentType = "tool_calling"
)

// AgentConfig Agent 配置
type AgentConfig struct {
    LLM           chat.ChatModel
    Tools         []types.Tool
    SystemPrompt  string
    Prompt        *prompts.PromptTemplate
    MaxIterations int
    Timeout       time.Duration
}

// CreateAgent 创建 Agent (工厂函数)
func CreateAgent(agentType AgentType, config AgentConfig) (Agent, error) {
    switch agentType {
    case AgentTypeReAct:
        return NewReActAgent(config), nil
    case AgentTypeToolCalling:
        return NewToolCallingAgent(config), nil
    default:
        return nil, fmt.Errorf("unknown agent type: %s", agentType)
    }
}
```

**ReActAgent**:
```go
// core/agents/react.go
package agents

type ReActAgent struct {
    llm           chat.ChatModel
    tools         map[string]types.Tool
    prompt        *prompts.PromptTemplate
    parser        AgentActionParser
    maxIterations int
    timeout       time.Duration
}

// NewReActAgent 创建 ReAct Agent
func NewReActAgent(config AgentConfig) *ReActAgent {
    // 设置默认值
    if config.MaxIterations == 0 {
        config.MaxIterations = 10
    }
    if config.Timeout == 0 {
        config.Timeout = 5 * time.Minute
    }
    if config.Prompt == nil {
        config.Prompt = DefaultReActPrompt
    }
    
    // 工具转 map
    toolsMap := make(map[string]types.Tool)
    for _, tool := range config.Tools {
        toolsMap[tool.Name] = tool
    }
    
    return &ReActAgent{
        llm:           config.LLM,
        tools:         toolsMap,
        prompt:        config.Prompt,
        parser:        NewReActOutputParser(),
        maxIterations: config.MaxIterations,
        timeout:       config.Timeout,
    }
}

// Run 执行 Agent
func (a *ReActAgent) Run(ctx context.Context, input string) (AgentResult, error) {
    start := time.Now()
    
    // 设置超时
    ctx, cancel := context.WithTimeout(ctx, a.timeout)
    defer cancel()
    
    var steps []AgentStep
    var finalAnswer string
    
    // ReAct 循环: Thought -> Action -> Observation
    for i := 0; i < a.maxIterations; i++ {
        select {
        case <-ctx.Done():
            return AgentResult{
                Input:        input,
                Output:       "执行超时",
                Steps:        steps,
                TimeElapsed:  time.Since(start),
                FinishReason: FinishReasonTimeout,
            }, nil
        default:
        }
        
        // 1. 构建 prompt
        promptStr, _ := a.buildPrompt(input, steps)
        messages := []types.Message{types.NewUserMessage(promptStr)}
        
        // 2. 调用 LLM
        response, err := a.llm.Invoke(ctx, messages)
        if err != nil {
            return AgentResult{}, err
        }
        
        // 3. 解析输出
        action, isFinish, err := a.parser.Parse(response.Content)
        if err != nil {
            // 处理解析错误
            continue
        }
        
        // 4. 检查是否完成
        if isFinish {
            finalAnswer = action.Log
            break
        }
        
        // 5. 执行工具
        tool, exists := a.tools[action.Tool]
        if !exists {
            steps = append(steps, AgentStep{
                Action:      action,
                Observation: fmt.Sprintf("Error: tool '%s' not found", action.Tool),
            })
            continue
        }
        
        observation, err := a.executeTool(ctx, tool, action.ToolInput)
        if err != nil {
            observation = fmt.Sprintf("Error: %v", err)
        }
        
        // 6. 记录步骤
        steps = append(steps, AgentStep{
            Action:      action,
            Observation: observation,
            Thought:     "", // 从 response 中提取
        })
    }
    
    // 7. 构建结果
    finishReason := FinishReasonCompleted
    if len(steps) >= a.maxIterations {
        finishReason = FinishReasonMaxIterations
    }
    
    return AgentResult{
        Input:        input,
        Output:       finalAnswer,
        Steps:        steps,
        TimeElapsed:  time.Since(start),
        FinishReason: finishReason,
    }, nil
}

// buildPrompt 构建 ReAct prompt
func (a *ReActAgent) buildPrompt(input string, steps []AgentStep) (string, error) {
    // 构建工具描述
    toolsDesc := a.getToolsDescription()
    
    // 构建历史步骤
    history := a.formatSteps(steps)
    
    return a.prompt.Format(map[string]interface{}{
        "tools":    toolsDesc,
        "input":    input,
        "history":  history,
    })
}

// executeTool 执行工具
func (a *ReActAgent) executeTool(ctx context.Context, tool types.Tool, input map[string]interface{}) (string, error) {
    // 调用工具函数
    // 这里需要工具注册系统
    return "tool result", nil
}
```

**AgentExecutor**:
```go
// core/agents/executor.go
package agents

type AgentExecutor struct {
    agent         Agent
    tools         map[string]types.Tool
    maxIterations int
    timeout       time.Duration
    callbacks     []Callback
    returnIntermediate bool
}

// NewAgentExecutor 创建执行器
func NewAgentExecutor(agent Agent, tools []types.Tool, opts ...ExecutorOption) *AgentExecutor {
    toolsMap := make(map[string]types.Tool)
    for _, tool := range tools {
        toolsMap[tool.Name] = tool
    }
    
    return &AgentExecutor{
        agent:         agent,
        tools:         toolsMap,
        maxIterations: 10,
        timeout:       5 * time.Minute,
    }
}

// Run 执行 Agent (带错误处理、重试等)
func (e *AgentExecutor) Run(ctx context.Context, input string) (AgentResult, error) {
    // 添加超时
    ctx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()
    
    // 调用 Agent
    result, err := e.agent.Run(ctx, input)
    if err != nil {
        // 错误处理和重试逻辑
        return AgentResult{}, err
    }
    
    // 触发 callbacks
    for _, cb := range e.callbacks {
        cb.OnAgentFinish(result)
    }
    
    return result, nil
}
```

---

## 📦 Part 4: 内置工具参考

### Python (langchain-community)

**工具定义**:
```python
from langchain.tools import tool

@tool
def calculator(expression: str) -> str:
    """执行数学计算"""
    return str(eval(expression))

# 使用
tools = [calculator]
```

**内置工具包**:
- `TavilySearchResults`
- `SerpAPIWrapper`  
- `WikipediaQueryRun`
- `ShellTool`
- `PythonREPLTool`
- `RequestsGetTool`, `RequestsPostTool`
- `HumanInputRun`

---

### Go 实现建议

**目录结构**:
```
langchain-go/core/tools/builtin/
├── calculator.go     # 计算器
├── time.go           # 时间工具
├── web_search.go     # 网页搜索
├── file.go           # 文件操作
├── sql.go            # SQL 数据库
├── http.go           # HTTP 请求
└── builtin_test.go   # 测试
```

**实现示例**:
```go
// core/tools/builtin/calculator.go
package builtin

import (
    "fmt"
    "github.com/Knetic/govaluate"
    "github.com/zhucl121/langchain-go/pkg/types"
)

// NewCalculator 创建计算器工具
func NewCalculator() types.Tool {
    return types.Tool{
        Name:        "calculator",
        Description: "执行数学计算,支持 +, -, *, /, ^, sqrt 等运算",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "expression": {
                    Type:        "string",
                    Description: "数学表达式,例如: '(15 * 8) + 42'",
                },
            },
            Required: []string{"expression"},
        },
        Function: calculatorFunc,
    }
}

func calculatorFunc(ctx context.Context, args map[string]interface{}) (string, error) {
    expr, ok := args["expression"].(string)
    if !ok {
        return "", fmt.Errorf("invalid expression")
    }
    
    // 使用 govaluate 安全计算
    expression, err := govaluate.NewEvaluableExpression(expr)
    if err != nil {
        return "", err
    }
    
    result, err := expression.Evaluate(nil)
    if err != nil {
        return "", err
    }
    
    return fmt.Sprintf("%v", result), nil
}
```

```go
// core/tools/builtin/time.go
package builtin

func NewGetTime() types.Tool {
    return types.Tool{
        Name:        "get_time",
        Description: "获取当前时间",
        Parameters: types.Schema{
            Type: "object",
            Properties: map[string]types.Schema{
                "format": {
                    Type:        "string",
                    Description: "时间格式 (可选),默认 RFC3339",
                },
            },
        },
        Function: getTimeFunc,
    }
}

func getTimeFunc(ctx context.Context, args map[string]interface{}) (string, error) {
    format := "2006-01-02 15:04:05"
    if f, ok := args["format"].(string); ok {
        format = f
    }
    return time.Now().Format(format), nil
}
```

---

## 📦 Part 5: Prompt 模板库

### Python (LangChain Hub)

```python
from langchain import hub

# 从 Hub 拉取
rag_prompt = hub.pull("rlm/rag-prompt")
react_prompt = hub.pull("hwchase17/react")
```

---

### Go 实现建议

**目录结构**:
```
langchain-go/core/prompts/templates/
├── rag.go      # RAG 模板
├── agent.go    # Agent 模板
├── qa.go       # QA 模板
└── common.go   # 通用模板
```

**实现**:
```go
// core/prompts/templates/rag.go
package templates

import "github.com/zhucl121/langchain-go/core/prompts"

// DefaultRAGPrompt 默认 RAG prompt
var DefaultRAGPrompt = prompts.NewPromptTemplate(`
基于以下上下文回答问题。如果上下文中没有相关信息,请明确说明无法回答。

上下文:
{{.Context}}

问题: {{.Question}}

回答:`, []string{"Context", "Question"})

// ConversationalRAGPrompt 对话式 RAG prompt
var ConversationalRAGPrompt = prompts.NewPromptTemplate(`
基于以下上下文和对话历史回答问题。

对话历史:
{{.ChatHistory}}

上下文:
{{.Context}}

问题: {{.Question}}

回答:`, []string{"ChatHistory", "Context", "Question"})

// MultilingualRAGPrompt 多语言 RAG prompt
var MultilingualRAGPrompt = prompts.NewPromptTemplate(`
Based on the following context, answer the question in the same language as the question.

Context:
{{.Context}}

Question: {{.Question}}

Answer:`, []string{"Context", "Question"})
```

```go
// core/prompts/templates/agent.go
package templates

// ReActPrompt ReAct Agent prompt
var ReActPrompt = prompts.NewPromptTemplate(`
Answer the following questions as best you can. You have access to the following tools:

{{.Tools}}

Use the following format:

Question: the input question you must answer
Thought: you should always think about what to do
Action: the action to take, should be one of [{{.ToolNames}}]
Action Input: the input to the action
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question

Begin!

Question: {{.Input}}
{{.History}}
Thought:`, []string{"Tools", "ToolNames", "Input", "History"})

// PlanExecutePrompt Plan-Execute prompt
var PlanExecutePrompt = prompts.NewPromptTemplate(`
Let's first understand the problem and devise a plan to solve it.
Then, let's carry out the plan step by step.

Problem: {{.Input}}

Plan:`, []string{"Input"})
```

---

## 📊 实现优先级矩阵

| 功能 | Python 状态 | Go 当前 | 实现难度 | 优先级 | 预计工作量 |
|------|------------|---------|---------|--------|-----------|
| RAG Chain | ✅ 完整 | ❌ 无 | ⭐⭐⭐ | P0 | 2 周 |
| Retriever | ✅ 完整 | ⚠️ 基础 | ⭐⭐ | P0 | 1 周 |
| Agent API | ✅ 完整 | ❌ 无 | ⭐⭐⭐⭐ | P0 | 2 周 |
| 内置工具 | ✅ 丰富 | ❌ 无 | ⭐⭐ | P1 | 1 周 |
| Prompt 模板 | ✅ Hub | ❌ 无 | ⭐ | P1 | 0.5 周 |
| Pipeline | ✅ 完整 | ⚠️ 手动 | ⭐⭐ | P1 | 1 周 |

---

## 🎯 关键建议

### 1. 直接参考 Python v1.0 API

**为什么?**
- ✅ Python v1.0 是经过实战验证的稳定版本
- ✅ API 设计经过多次迭代优化
- ✅ 社区广泛采用

**如何参考?**
1. **API 设计**: 接口、方法名、参数
2. **实现模式**: 工厂函数、Builder 模式
3. **默认值**: 合理的默认配置
4. **错误处理**: 边界情况处理

### 2. 保持 Go 特色

**Go 的优势**:
- ✅ 类型安全 (编译期检查)
- ✅ 并发性能 (goroutine)
- ✅ 简洁语法
- ✅ 快速编译

**设计建议**:
```go
// ✅ Good: 函数式选项模式 (Go 惯用法)
func NewRAGChain(retriever Retriever, llm ChatModel, opts ...Option) *RAGChain

// ❌ Bad: Python 风格的 kwargs
func NewRAGChain(args map[string]interface{}) *RAGChain

// ✅ Good: Context 作为第一个参数
func (c *RAGChain) Run(ctx context.Context, query string) (Result, error)

// ❌ Bad: 忽略 Context
func (c *RAGChain) Run(query string) (Result, error)
```

### 3. 性能优化

**利用 Go 优势**:
- 并行处理批量请求
- 使用 channel 实现流式输出
- Worker pool 处理文档
- 连接池管理

---

## 📚 参考资源

### Python 官方文档
- **LangChain API**: https://reference.langchain.com/python/
- **LangGraph**: https://pypi.org/project/langgraph/
- **LangChain Hub**: https://smith.langchain.com/hub

### 关键代码
- **create_retrieval_chain**: 学习 RAG Chain 实现
- **MultiQueryRetriever**: 学习多查询生成
- **EnsembleRetriever**: 学习 RRF 融合
- **create_agent**: 学习 Agent 工厂模式

---

## 🎉 总结

### 核心答案

**对比 LangChain 和 LangGraph 的最新 Python 版本,具备这些扩展功能吗?**

✅ **完全具备!** Python 版本不仅具备我们分析的所有功能,而且实现得更加完善。

### 关键数据

| 维度 | Python | Go | 差距 |
|------|--------|----|----|
| **功能完整度** | 100% | 20% | **80%** |
| **高层 API** | ✅ 齐全 | ❌ 缺失 | **巨大** |
| **开发效率** | 3-5 行 | 100-200 行 | **50x** |
| **学习曲线** | 平缓 | 陡峭 | **显著** |

### 行动建议

1. **立即参考 Python API** - 不需要重新设计
2. **保持 Go 风格** - 使用 Go 惯用法
3. **优先 P0 功能** - RAG Chain, Retriever, Agent
4. **快速迭代** - 8 周完成核心功能

---

**结论**: Python 是我们最好的参考和学习对象! 🎯

---

**分析者**: AI Assistant  
**日期**: 2026-01-16  
**参考版本**: LangChain Python v1.0+, LangGraph v1.0.6

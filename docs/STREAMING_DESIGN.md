# Streaming 支持设计方案

## 1. 概述

为 LangChain-Go 添加完整的流式响应支持，包括 token-level streaming、tool call streaming 和 SSE 支持。

## 2. 核心设计

### 2.1 StreamEvent 增强

```go
// pkg/types/stream.go
type StreamEventType string

const (
    StreamEventStart       StreamEventType = "start"
    StreamEventToken       StreamEventType = "token"        // 新增：单个 token
    StreamEventToolCall    StreamEventType = "tool_call"    // 新增：工具调用
    StreamEventToolResult  StreamEventType = "tool_result"  // 新增：工具结果
    StreamEventEnd         StreamEventType = "end"
    StreamEventError       StreamEventType = "error"
)

type StreamEvent struct {
    Type     StreamEventType    `json:"type"`
    Data     any                `json:"data,omitempty"`
    Token    string             `json:"token,omitempty"`      // 新增
    Delta    string             `json:"delta,omitempty"`      // 新增
    ToolCall *ToolCall          `json:"tool_call,omitempty"`  // 新增
    Error    error              `json:"error,omitempty"`
    Metadata map[string]any     `json:"metadata,omitempty"`
}
```

### 2.2 ChatModel Stream 接口

```go
// core/chat/interface.go
type ChatModel interface {
    // ... 现有方法
    
    // Stream 流式调用（返回完整消息的流）
    Stream(ctx context.Context, messages []types.Message, opts ...Option) (<-chan types.StreamEvent, error)
    
    // StreamTokens 令牌级流式调用（返回单个 token）
    StreamTokens(ctx context.Context, messages []types.Message, opts ...Option) (<-chan string, error)
}
```

### 2.3 Stream Aggregator

```go
// core/chat/stream/aggregator.go
type StreamAggregator struct {
    events []types.StreamEvent
    buffer strings.Builder
}

func NewStreamAggregator() *StreamAggregator

func (a *StreamAggregator) Add(event types.StreamEvent) error
func (a *StreamAggregator) GetResult() (*types.Message, error)
func (a *StreamAggregator) GetContent() string
```

### 2.4 SSE 支持

```go
// core/chat/stream/sse.go
type SSEWriter struct {
    w io.Writer
}

func NewSSEWriter(w io.Writer) *SSEWriter

func (s *SSEWriter) WriteEvent(event types.StreamEvent) error
func (s *SSEWriter) WriteError(err error) error
func (s *SSEWriter) Close() error
```

## 3. Provider 实现

### 3.1 OpenAI Streaming

```go
// core/chat/providers/openai/streaming.go
func (c *OpenAIClient) Stream(ctx context.Context, messages []types.Message, opts ...Option) (<-chan types.StreamEvent, error) {
    out := make(chan types.StreamEvent, 100)
    
    go func() {
        defer close(out)
        
        // 发送开始事件
        out <- types.StreamEvent{Type: types.StreamEventStart}
        
        // 调用 OpenAI streaming API
        stream, err := c.createStreamRequest(ctx, messages, opts...)
        if err != nil {
            out <- types.StreamEvent{Type: types.StreamEventError, Error: err}
            return
        }
        
        // 处理流式响应
        for {
            response, err := stream.Recv()
            if err == io.EOF {
                break
            }
            if err != nil {
                out <- types.StreamEvent{Type: types.StreamEventError, Error: err}
                return
            }
            
            // 发送 token 事件
            if len(response.Choices) > 0 {
                delta := response.Choices[0].Delta.Content
                out <- types.StreamEvent{
                    Type:  types.StreamEventToken,
                    Token: delta,
                    Delta: delta,
                }
            }
        }
        
        // 发送结束事件
        out <- types.StreamEvent{Type: types.StreamEventEnd}
    }()
    
    return out, nil
}
```

## 4. 实现计划

### Phase 1: 核心基础设施 (2天)
- ✅ StreamEvent 类型增强
- ⬜ StreamAggregator 实现
- ⬜ SSEWriter 实现
- ⬜ 基础测试

### Phase 2: Provider 实现 (2天)
- ⬜ OpenAI streaming
- ⬜ Anthropic streaming
- ⬜ Gemini streaming
- ⬜ Ollama streaming

### Phase 3: 集成与测试 (1天)
- ⬜ Runnable Stream 支持
- ⬜ Agent Stream 支持
- ⬜ 集成测试

### Phase 4: 文档与示例 (1天)
- ⬜ API 文档
- ⬜ 使用示例
- ⬜ 最佳实践

## 5. 测试策略

### 5.1 单元测试
- StreamEvent 序列化/反序列化
- StreamAggregator 聚合逻辑
- SSEWriter 输出格式

### 5.2 集成测试
- 端到端流式调用
- 工具调用流式
- 错误处理

### 5.3 性能测试
- 流式延迟测试
- 内存使用测试
- 并发流测试

## 6. 使用示例

### 6.1 基础流式调用

```go
stream, err := chatModel.Stream(ctx, messages)
if err != nil {
    log.Fatal(err)
}

for event := range stream {
    switch event.Type {
    case types.StreamEventToken:
        fmt.Print(event.Token)
    case types.StreamEventEnd:
        fmt.Println("\n完成")
    case types.StreamEventError:
        log.Printf("错误: %v", event.Error)
    }
}
```

### 6.2 SSE 输出

```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    
    sse := stream.NewSSEWriter(w)
    defer sse.Close()
    
    streamCh, err := chatModel.Stream(r.Context(), messages)
    if err != nil {
        sse.WriteError(err)
        return
    }
    
    for event := range streamCh {
        if err := sse.WriteEvent(event); err != nil {
            return
        }
    }
}
```

### 6.3 流式聚合

```go
aggregator := stream.NewStreamAggregator()

streamCh, _ := chatModel.Stream(ctx, messages)
for event := range streamCh {
    aggregator.Add(event)
    
    // 实时显示进度
    fmt.Printf("\r当前内容: %s", aggregator.GetContent())
}

// 获取最终结果
message, err := aggregator.GetResult()
```

## 7. 性能目标

- **首 Token 延迟**: < 500ms
- **Token 间延迟**: < 50ms  
- **内存开销**: < 10MB per stream
- **并发支持**: 1000+ 并发流

## 8. 兼容性

- ✅ 向后兼容现有 Invoke API
- ✅ 所有 Provider 统一接口
- ✅ 可选功能（不影响非流式调用）

## 9. 参考资料

- OpenAI Streaming API: https://platform.openai.com/docs/api-reference/chat/create#stream
- Anthropic Streaming: https://docs.anthropic.com/claude/reference/messages-streaming
- SSE 规范: https://html.spec.whatwg.org/multipage/server-sent-events.html

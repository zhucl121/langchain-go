// Package streaming 提供 LangGraph 流式执行能力。
//
// 包含 ResumableStream（可恢复流）实现，支持网络中断后自动续传，
// 对标 LangGraph 1.0 Resumable Streams 特性。
package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// StreamEvent 流式事件。
type StreamEvent struct {
	// ID 事件唯一 ID（单调递增）
	ID int64

	// Type 事件类型
	Type StreamEventType

	// NodeName 产生该事件的节点名称
	NodeName string

	// Data 事件数据（JSON 可序列化）
	Data any

	// Error 错误信息（仅 ErrorEvent 时非 nil）
	Error string

	// Timestamp 事件时间戳
	Timestamp time.Time

	// IsFinal 是否为最终事件
	IsFinal bool
}

// StreamEventType 流式事件类型。
type StreamEventType string

const (
	EventTypeToken    StreamEventType = "token"     // token 流
	EventTypeState    StreamEventType = "state"     // 状态更新
	EventTypeNodeDone StreamEventType = "node_done" // 节点完成
	EventTypeError    StreamEventType = "error"     // 错误
	EventTypeDone     StreamEventType = "done"      // 流结束
)

// StreamStorage 流状态持久化存储接口。
//
// 用于保存已发送的事件，支持断点续传。
type StreamStorage interface {
	// SaveEvent 保存一个流事件
	SaveEvent(ctx context.Context, streamID string, event *StreamEvent) error

	// LoadEvents 加载指定 streamID 中 afterID 之后的所有事件
	LoadEvents(ctx context.Context, streamID string, afterID int64) ([]*StreamEvent, error)

	// MarkComplete 标记流已完成
	MarkComplete(ctx context.Context, streamID string) error

	// IsComplete 检查流是否已完成
	IsComplete(ctx context.Context, streamID string) (bool, error)

	// TTL 返回存储的事件 TTL（超过 TTL 的流不能续传）
	TTL() time.Duration

	// Delete 删除流状态（清理）
	Delete(ctx context.Context, streamID string) error
}

// ResumableStreamConfig 可恢复流配置。
type ResumableStreamConfig struct {
	// StreamID 流的唯一标识（客户端重连时使用相同 ID）
	StreamID string

	// Storage 事件持久化后端（nil 时使用内存存储，不支持跨进程续传）
	Storage StreamStorage

	// BufferSize channel 缓冲大小
	BufferSize int

	// MaxEvents 最大事件数（超过时停止）（0 表示无限）
	MaxEvents int64
}

// ResumableStream 可恢复的流执行器。
//
// ResumableStream 持久化已发出的事件，当客户端因网络中断重连时，
// 可以从断点（lastEventID）继续接收后续事件，不丢失任何 token。
//
// 对标 LangGraph 1.0 Resumable Streams。
//
// 示例：
//
//	// 服务端：创建流
//	rs := streaming.NewResumableStream(streaming.ResumableStreamConfig{
//	    StreamID: "sess-12345",
//	    Storage:  redisStorage,
//	})
//
//	// 启动流（在 goroutine 中产生事件）
//	outCh := rs.Start(ctx, producerFn)
//
//	// 客户端重连：从断点续传
//	resumedCh, err := rs.Resume(ctx, lastEventID)
type ResumableStream struct {
	config  ResumableStreamConfig
	storage StreamStorage

	mu       sync.Mutex
	idGen    atomic.Int64
	eventBuf []*StreamEvent
	done     chan struct{}
	doneOnce sync.Once
	started  bool
}

// NewResumableStream 创建可恢复流。
func NewResumableStream(config ResumableStreamConfig) *ResumableStream {
	storage := config.Storage
	if storage == nil {
		storage = NewInMemoryStreamStorage(5 * time.Minute)
	}

	bufSize := config.BufferSize
	if bufSize <= 0 {
		bufSize = 128
	}

	return &ResumableStream{
		config:  config,
		storage: storage,
		done:    make(chan struct{}),
	}
}

// ProducerFunc 事件生产函数类型。
// 调用 emit 发送事件，函数返回后流结束。
type ProducerFunc func(ctx context.Context, emit func(event *StreamEvent) error) error

// Start 启动流，使用 producerFn 异步产生事件。
//
// 返回事件 channel，调用方从 channel 消费事件。
// producerFn 在独立 goroutine 中运行，通过 emit 发送事件。
func (rs *ResumableStream) Start(ctx context.Context, producerFn ProducerFunc) (<-chan *StreamEvent, error) {
	rs.mu.Lock()
	if rs.started {
		rs.mu.Unlock()
		return nil, fmt.Errorf("resumable stream %s: already started", rs.config.StreamID)
	}
	rs.started = true
	rs.mu.Unlock()

	bufSize := rs.config.BufferSize
	if bufSize <= 0 {
		bufSize = 128
	}

	out := make(chan *StreamEvent, bufSize)

	emit := func(event *StreamEvent) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-rs.done:
			return errors.New("stream closed")
		default:
		}

		// 分配 ID
		event.ID = rs.idGen.Add(1)
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}

		// 持久化
		if err := rs.storage.SaveEvent(ctx, rs.config.StreamID, event); err != nil {
			return fmt.Errorf("resumable stream: save event failed: %w", err)
		}

		// 发送到 channel
		select {
		case out <- event:
		case <-ctx.Done():
			return ctx.Err()
		}

		// 检查最大事件数
		if rs.config.MaxEvents > 0 && event.ID >= rs.config.MaxEvents {
			return fmt.Errorf("max events %d reached", rs.config.MaxEvents)
		}

		return nil
	}

	go func() {
		defer func() {
			// 标记完成
			_ = rs.storage.MarkComplete(context.Background(), rs.config.StreamID)

			// 发送结束事件
			doneEvent := &StreamEvent{
				ID:        rs.idGen.Add(1),
				Type:      EventTypeDone,
				Timestamp: time.Now(),
				IsFinal:   true,
			}
			select {
			case out <- doneEvent:
			default:
			}
			close(out)
		}()

		if err := producerFn(ctx, emit); err != nil {
			errEvent := &StreamEvent{
				ID:        rs.idGen.Add(1),
				Type:      EventTypeError,
				Error:     err.Error(),
				Timestamp: time.Now(),
				IsFinal:   true,
			}
			select {
			case out <- errEvent:
			default:
			}
		}
	}()

	return out, nil
}

// Resume 从断点续传。
//
// 返回从 afterEventID 之后的所有历史事件 + 后续新事件的 channel。
// 如果流已完成，只返回历史事件（channel 随即关闭）。
//
// 参数：
//   - ctx: 上下文
//   - afterEventID: 客户端已收到的最后一个事件 ID（从 ID+1 开始续传）
func (rs *ResumableStream) Resume(ctx context.Context, afterEventID int64) (<-chan *StreamEvent, error) {
	// 检查流是否还有效（未超过 TTL）
	complete, err := rs.storage.IsComplete(ctx, rs.config.StreamID)
	if err != nil {
		return nil, fmt.Errorf("resumable stream resume: %w", err)
	}

	// 加载历史事件
	history, err := rs.storage.LoadEvents(ctx, rs.config.StreamID, afterEventID)
	if err != nil {
		return nil, fmt.Errorf("resumable stream resume: load history: %w", err)
	}

	bufSize := rs.config.BufferSize
	if bufSize <= 0 {
		bufSize = 128
	}

	out := make(chan *StreamEvent, bufSize+len(history))

	// 先发送历史事件
	for _, e := range history {
		out <- e
	}

	if complete {
		// 流已完成，关闭 channel
		close(out)
		return out, nil
	}

	// 流还在运行，等待新事件（轮询存储）
	go func() {
		defer close(out)

		lastID := afterEventID
		if len(history) > 0 {
			lastID = history[len(history)-1].ID
		}

		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				done, _ := rs.storage.IsComplete(ctx, rs.config.StreamID)
				newEvents, _ := rs.storage.LoadEvents(ctx, rs.config.StreamID, lastID)

				for _, e := range newEvents {
					select {
					case out <- e:
						lastID = e.ID
					case <-ctx.Done():
						return
					}
				}

				if done && len(newEvents) == 0 {
					return
				}
			}
		}
	}()

	return out, nil
}

// StreamID 返回流 ID。
func (rs *ResumableStream) StreamID() string {
	return rs.config.StreamID
}

// ─── InMemoryStreamStorage 内存存储实现 ────────────────────────

// inMemoryStreamData 内存中的流数据。
type inMemoryStreamData struct {
	events    []*StreamEvent
	complete  bool
	createdAt time.Time
}

// InMemoryStreamStorage 基于内存的流存储。
//
// 适用于开发/测试环境。生产环境建议使用 Redis 存储。
type InMemoryStreamStorage struct {
	mu      sync.RWMutex
	streams map[string]*inMemoryStreamData
	ttl     time.Duration
}

// NewInMemoryStreamStorage 创建内存流存储。
func NewInMemoryStreamStorage(ttl time.Duration) *InMemoryStreamStorage {
	s := &InMemoryStreamStorage{
		streams: make(map[string]*inMemoryStreamData),
		ttl:     ttl,
	}
	go s.cleanupLoop()
	return s
}

func (s *InMemoryStreamStorage) SaveEvent(ctx context.Context, streamID string, event *StreamEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.streams[streamID]; !ok {
		s.streams[streamID] = &inMemoryStreamData{
			events:    make([]*StreamEvent, 0, 32),
			createdAt: time.Now(),
		}
	}

	s.streams[streamID].events = append(s.streams[streamID].events, event)
	return nil
}

func (s *InMemoryStreamStorage) LoadEvents(ctx context.Context, streamID string, afterID int64) ([]*StreamEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.streams[streamID]
	if !ok {
		return nil, nil
	}

	// 检查是否过期
	if s.ttl > 0 && time.Since(data.createdAt) > s.ttl {
		return nil, fmt.Errorf("stream %s expired", streamID)
	}

	var result []*StreamEvent
	for _, e := range data.events {
		if e.ID > afterID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *InMemoryStreamStorage) MarkComplete(ctx context.Context, streamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if data, ok := s.streams[streamID]; ok {
		data.complete = true
	}
	return nil
}

func (s *InMemoryStreamStorage) IsComplete(ctx context.Context, streamID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.streams[streamID]
	if !ok {
		return false, nil
	}
	return data.complete, nil
}

func (s *InMemoryStreamStorage) TTL() time.Duration {
	return s.ttl
}

func (s *InMemoryStreamStorage) Delete(ctx context.Context, streamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.streams, streamID)
	return nil
}

func (s *InMemoryStreamStorage) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, data := range s.streams {
			if s.ttl > 0 && now.Sub(data.createdAt) > s.ttl {
				delete(s.streams, id)
			}
		}
		s.mu.Unlock()
	}
}

// ─── JSON 序列化助手 ──────────────────────────────────────────

// EventToJSON 将事件序列化为 JSON。
func EventToJSON(event *StreamEvent) ([]byte, error) {
	return json.Marshal(event)
}

// EventFromJSON 从 JSON 反序列化事件。
func EventFromJSON(data []byte) (*StreamEvent, error) {
	var event StreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ─── 便捷构造函数 ─────────────────────────────────────────────

// NewTokenEvent 创建 token 流事件。
func NewTokenEvent(nodeName, token string) *StreamEvent {
	return &StreamEvent{
		Type:      EventTypeToken,
		NodeName:  nodeName,
		Data:      token,
		Timestamp: time.Now(),
	}
}

// NewStateEvent 创建状态更新事件。
func NewStateEvent(nodeName string, state any) *StreamEvent {
	return &StreamEvent{
		Type:      EventTypeState,
		NodeName:  nodeName,
		Data:      state,
		Timestamp: time.Now(),
	}
}

// NewNodeDoneEvent 创建节点完成事件。
func NewNodeDoneEvent(nodeName string, duration time.Duration) *StreamEvent {
	return &StreamEvent{
		Type:      EventTypeNodeDone,
		NodeName:  nodeName,
		Data:      map[string]any{"duration_ms": duration.Milliseconds()},
		Timestamp: time.Now(),
	}
}

// NewErrorEvent 创建错误事件。
func NewErrorEvent(nodeName string, err error) *StreamEvent {
	return &StreamEvent{
		Type:      EventTypeError,
		NodeName:  nodeName,
		Error:     err.Error(),
		Timestamp: time.Now(),
		IsFinal:   true,
	}
}

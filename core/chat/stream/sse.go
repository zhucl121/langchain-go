package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// SSEWriter 用于写入 Server-Sent Events (SSE) 格式的流式数据。
//
// SSE 是一种服务器向客户端推送实时更新的标准协议。
//
// 示例：
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    w.Header().Set("Content-Type", "text/event-stream")
//	    w.Header().Set("Cache-Control", "no-cache")
//	    
//	    sse := stream.NewSSEWriter(w)
//	    defer sse.Close()
//	    
//	    for event := range streamCh {
//	        sse.WriteEvent(event)
//	    }
//	}
//
type SSEWriter struct {
	w  io.Writer
	mu sync.Mutex

	eventCount int
	flusher    interface{ Flush() } // http.Flusher
}

// NewSSEWriter 创建新的 SSE 写入器。
//
// 参数：
//   - w: 输出写入器（通常是 http.ResponseWriter）
//
// 返回：
//   - *SSEWriter: SSE 写入器实例
//
func NewSSEWriter(w io.Writer) *SSEWriter {
	sse := &SSEWriter{
		w: w,
	}

	// 尝试获取 Flusher 接口
	if f, ok := w.(interface{ Flush() }); ok {
		sse.flusher = f
	}

	return sse
}

// WriteEvent 写入一个流式事件。
//
// 参数：
//   - event: 流式事件
//
// 返回：
//   - error: 写入错误
//
func (s *SSEWriter) WriteEvent(event types.StreamEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 序列化事件为 JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 写入 SSE 格式
	// event: <event_type>
	// data: <json_data>
	// <blank line>
	if _, err := fmt.Fprintf(s.w, "event: %s\n", event.Type); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}

	s.eventCount++

	// 刷新缓冲区（如果支持）
	if s.flusher != nil {
		s.flusher.Flush()
	}

	return nil
}

// WriteData 写入原始数据。
//
// 参数：
//   - data: 数据
//
// 返回：
//   - error: 写入错误
//
func (s *SSEWriter) WriteData(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}

	s.eventCount++

	if s.flusher != nil {
		s.flusher.Flush()
	}

	return nil
}

// WriteComment 写入注释（用于保持连接活跃）。
//
// 参数：
//   - comment: 注释内容
//
// 返回：
//   - error: 写入错误
//
func (s *SSEWriter) WriteComment(comment string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := fmt.Fprintf(s.w, ": %s\n\n", comment); err != nil {
		return err
	}

	if s.flusher != nil {
		s.flusher.Flush()
	}

	return nil
}

// WriteError 写入错误事件。
//
// 参数：
//   - err: 错误
//
// 返回：
//   - error: 写入错误
//
func (s *SSEWriter) WriteError(err error) error {
	event := types.NewErrorEvent(err)
	return s.WriteEvent(event)
}

// WriteToken 写入 token 事件。
//
// 参数：
//   - token: token 内容
//
// 返回：
//   - error: 写入错误
//
func (s *SSEWriter) WriteToken(token string) error {
	event := types.NewTokenEvent(token)
	return s.WriteEvent(event)
}

// WriteEnd 写入结束事件。
//
// 返回：
//   - error: 写入错误
//
func (s *SSEWriter) WriteEnd() error {
	event := types.StreamEvent{
		Type: types.StreamEventEnd,
		Done: true,
	}
	return s.WriteEvent(event)
}

// GetEventCount 获取已写入的事件数量。
//
// 返回：
//   - int: 事件数量
//
func (s *SSEWriter) GetEventCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.eventCount
}

// Close 关闭 SSE 写入器。
//
// 返回：
//   - error: 关闭错误
//
func (s *SSEWriter) Close() error {
	// 写入结束事件
	if err := s.WriteEnd(); err != nil {
		return err
	}

	// 如果实现了 io.Closer，则关闭
	if c, ok := s.w.(io.Closer); ok {
		return c.Close()
	}

	return nil
}

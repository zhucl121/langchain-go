package streaming

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestResumableStream_BasicFlow(t *testing.T) {
	storage := NewInMemoryStreamStorage(5 * time.Minute)
	rs := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-stream-1",
		Storage:  storage,
	})

	ctx := context.Background()
	producer := func(ctx context.Context, emit func(*StreamEvent) error) error {
		for i := 1; i <= 5; i++ {
			if err := emit(NewTokenEvent("agent", fmt.Sprintf("token-%d", i))); err != nil {
				return err
			}
		}
		return nil
	}

	ch, err := rs.Start(ctx, producer)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	var events []*StreamEvent
	for e := range ch {
		events = append(events, e)
	}

	// 5 token events + 1 done event
	if len(events) < 5 {
		t.Errorf("want at least 5 events, got %d", len(events))
	}

	// 最后一个应该是 done
	last := events[len(events)-1]
	if last.Type != EventTypeDone {
		t.Errorf("last event should be done, got %s", last.Type)
	}
}

func TestResumableStream_Resume_AfterComplete(t *testing.T) {
	storage := NewInMemoryStreamStorage(5 * time.Minute)
	rs := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-stream-resume",
		Storage:  storage,
	})

	ctx := context.Background()
	producer := func(ctx context.Context, emit func(*StreamEvent) error) error {
		for i := 1; i <= 10; i++ {
			if err := emit(NewTokenEvent("node", fmt.Sprintf("t%d", i))); err != nil {
				return err
			}
		}
		return nil
	}

	// 启动并消费全部事件
	ch, _ := rs.Start(ctx, producer)
	var allEvents []*StreamEvent
	for e := range ch {
		allEvents = append(allEvents, e)
	}

	// 模拟客户端只收到了前 5 个事件，断线重连
	// afterID = allEvents[4].ID (第 5 个事件的 ID)
	if len(allEvents) < 6 {
		t.Skip("not enough events for resume test")
	}
	breakPointID := allEvents[4].ID

	// 创建新的 ResumableStream 实例，使用相同的 streamID + storage
	rs2 := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-stream-resume",
		Storage:  storage,
	})

	resumeCh, err := rs2.Resume(ctx, breakPointID)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	var resumedEvents []*StreamEvent
	for e := range resumeCh {
		resumedEvents = append(resumedEvents, e)
	}

	// 续传应该包含 ID > breakPointID 的事件
	for _, e := range resumedEvents {
		if e.ID <= breakPointID {
			t.Errorf("resumed event ID %d should be > breakPoint %d", e.ID, breakPointID)
		}
	}
	t.Logf("resumed %d events after breakpoint %d", len(resumedEvents), breakPointID)
}

func TestResumableStream_Resume_FromZero(t *testing.T) {
	storage := NewInMemoryStreamStorage(5 * time.Minute)
	rs := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-stream-from-zero",
		Storage:  storage,
	})

	ctx := context.Background()
	producer := func(ctx context.Context, emit func(*StreamEvent) error) error {
		for i := 1; i <= 3; i++ {
			_ = emit(NewTokenEvent("n", fmt.Sprintf("t%d", i)))
		}
		return nil
	}

	ch, _ := rs.Start(ctx, producer)
	for range ch {
	} // 等完成

	// 从 0 续传（获取所有历史事件）
	rs2 := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-stream-from-zero",
		Storage:  storage,
	})
	resumeCh, err := rs2.Resume(ctx, 0)
	if err != nil {
		t.Fatalf("Resume from 0 failed: %v", err)
	}

	var events []*StreamEvent
	for e := range resumeCh {
		events = append(events, e)
	}

	if len(events) < 3 {
		t.Errorf("want at least 3 events from history, got %d", len(events))
	}
}

func TestResumableStream_ContextCancel(t *testing.T) {
	storage := NewInMemoryStreamStorage(5 * time.Minute)
	rs := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-cancel",
		Storage:  storage,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := func(ctx context.Context, emit func(*StreamEvent) error) error {
		for i := 1; ; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if err := emit(NewTokenEvent("n", fmt.Sprintf("t%d", i))); err != nil {
				return err
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	ch, _ := rs.Start(ctx, producer)

	// 收到 3 个事件后取消
	count := 0
	for range ch {
		count++
		if count >= 3 {
			cancel()
			break
		}
	}
	// 排干 channel
	for range ch {
	}

	if count < 3 {
		t.Errorf("want at least 3 events before cancel, got %d", count)
	}
}

func TestResumableStream_ProducerError(t *testing.T) {
	storage := NewInMemoryStreamStorage(5 * time.Minute)
	rs := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-error",
		Storage:  storage,
	})

	ctx := context.Background()
	producer := func(ctx context.Context, emit func(*StreamEvent) error) error {
		_ = emit(NewTokenEvent("n", "t1"))
		return errors.New("producer failed")
	}

	ch, _ := rs.Start(ctx, producer)

	var hasError bool
	for e := range ch {
		if e.Type == EventTypeError {
			hasError = true
		}
	}

	if !hasError {
		t.Error("expected error event in stream")
	}
}

func TestResumableStream_AlreadyStarted(t *testing.T) {
	storage := NewInMemoryStreamStorage(5 * time.Minute)
	rs := NewResumableStream(ResumableStreamConfig{
		StreamID: "test-double-start",
		Storage:  storage,
	})

	ctx := context.Background()
	noop := func(ctx context.Context, emit func(*StreamEvent) error) error { return nil }

	ch, err := rs.Start(ctx, noop)
	if err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	for range ch {
	}

	_, err = rs.Start(ctx, noop)
	if err == nil {
		t.Error("second start should fail")
	}
}

func TestInMemoryStreamStorage_TTLExpiry(t *testing.T) {
	storage := NewInMemoryStreamStorage(50 * time.Millisecond)

	ctx := context.Background()
	_ = storage.SaveEvent(ctx, "expired-stream", NewTokenEvent("n", "t"))

	time.Sleep(100 * time.Millisecond)

	_, err := storage.LoadEvents(ctx, "expired-stream", 0)
	if err == nil {
		t.Error("expected TTL expiry error")
	}
}

func TestEventConstructors(t *testing.T) {
	token := NewTokenEvent("agent", "hello")
	if token.Type != EventTypeToken || token.NodeName != "agent" {
		t.Errorf("unexpected token event: %+v", token)
	}

	state := NewStateEvent("node", map[string]any{"k": "v"})
	if state.Type != EventTypeState {
		t.Errorf("unexpected state event type: %s", state.Type)
	}

	done := NewNodeDoneEvent("node", 100*time.Millisecond)
	if done.Type != EventTypeNodeDone {
		t.Errorf("unexpected done event type: %s", done.Type)
	}

	errEvent := NewErrorEvent("node", errors.New("oops"))
	if errEvent.Type != EventTypeError || !errEvent.IsFinal {
		t.Errorf("unexpected error event: %+v", errEvent)
	}
}

func TestEventToJSON_RoundTrip(t *testing.T) {
	event := NewTokenEvent("agent", "hello world")
	event.ID = 42

	data, err := EventToJSON(event)
	if err != nil {
		t.Fatalf("EventToJSON failed: %v", err)
	}

	restored, err := EventFromJSON(data)
	if err != nil {
		t.Fatalf("EventFromJSON failed: %v", err)
	}

	if restored.ID != 42 || restored.NodeName != "agent" {
		t.Errorf("roundtrip mismatch: %+v", restored)
	}
}

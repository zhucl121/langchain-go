package failover

import (
	"errors"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config)

	if cb == nil {
		t.Fatal("NewCircuitBreaker() returned nil")
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Initial state = %s, want %s", cb.GetState(), StateClosed)
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config)

	// 成功执行
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// 验证状态仍然是关闭
	if cb.GetState() != StateClosed {
		t.Errorf("State = %s, want %s", cb.GetState(), StateClosed)
	}

	stats := cb.GetStats()
	if stats.SuccessRequests != 1 {
		t.Errorf("SuccessRequests = %d, want 1", stats.SuccessRequests)
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 3
	cb := NewCircuitBreaker(config)

	expectedErr := errors.New("test error")

	// 执行 3 次失败，达到阈值
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return expectedErr
		})

		if err != expectedErr {
			t.Errorf("Execute() error = %v, want %v", err, expectedErr)
		}
	}

	// 验证熔断器已打开
	if cb.GetState() != StateOpen {
		t.Errorf("State = %s, want %s", cb.GetState(), StateOpen)
	}

	stats := cb.GetStats()
	if stats.FailedRequests != 3 {
		t.Errorf("FailedRequests = %d, want 3", stats.FailedRequests)
	}
}

func TestCircuitBreaker_OpenState_RejectsRequests(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.Timeout = 100 * time.Millisecond
	cb := NewCircuitBreaker(config)

	// 触发一次失败，打开熔断器
	cb.Execute(func() error {
		return errors.New("test error")
	})

	// 验证熔断器已打开
	if cb.GetState() != StateOpen {
		t.Fatalf("State = %s, want %s", cb.GetState(), StateOpen)
	}

	// 尝试执行，应该被拒绝
	err := cb.Execute(func() error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Execute() error = %v, want %v", err, ErrCircuitOpen)
	}

	stats := cb.GetStats()
	if stats.RejectedRequests != 1 {
		t.Errorf("RejectedRequests = %d, want 1", stats.RejectedRequests)
	}
}

func TestCircuitBreaker_HalfOpen_Recovery(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.SuccessThreshold = 2
	config.Timeout = 50 * time.Millisecond
	config.MaxRequests = 2 // 允许 2 个请求测试
	cb := NewCircuitBreaker(config)

	// 触发失败，打开熔断器
	cb.Execute(func() error {
		return errors.New("test error")
	})

	if cb.GetState() != StateOpen {
		t.Fatalf("State = %s, want %s after failure", cb.GetState(), StateOpen)
	}

	// 等待超时，应该转为半开
	time.Sleep(60 * time.Millisecond)

	// 执行第一次成功请求
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// 应该转为半开状态
	if cb.GetState() != StateHalfOpen {
		t.Errorf("State = %s, want %s after timeout", cb.GetState(), StateHalfOpen)
	}

	// 执行第二次成功请求，达到成功阈值
	err = cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// 应该转为关闭状态
	if cb.GetState() != StateClosed {
		t.Errorf("State = %s, want %s after recovery", cb.GetState(), StateClosed)
	}
}

func TestCircuitBreaker_HalfOpen_FailureReopens(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.Timeout = 50 * time.Millisecond
	cb := NewCircuitBreaker(config)

	// 触发失败，打开熔断器
	cb.Execute(func() error {
		return errors.New("test error")
	})

	// 等待超时
	time.Sleep(60 * time.Millisecond)

	// 在半开状态下执行失败请求
	err := cb.Execute(func() error {
		return errors.New("another error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// 应该重新打开
	if cb.GetState() != StateOpen {
		t.Errorf("State = %s, want %s after failure in half-open", cb.GetState(), StateOpen)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	cb := NewCircuitBreaker(config)

	// 触发失败
	cb.Execute(func() error {
		return errors.New("test error")
	})

	if cb.GetState() != StateOpen {
		t.Fatalf("State = %s, want %s", cb.GetState(), StateOpen)
	}

	// 重置
	cb.Reset()

	// 验证状态
	if cb.GetState() != StateClosed {
		t.Errorf("State = %s, want %s after reset", cb.GetState(), StateClosed)
	}

	stats := cb.GetStats()
	if stats.FailedRequests != 0 {
		t.Errorf("FailedRequests = %d, want 0 after reset", stats.FailedRequests)
	}
}

func TestCircuitBreaker_ForceOpen(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config)

	// 强制打开
	cb.ForceOpen()

	if cb.GetState() != StateOpen {
		t.Errorf("State = %s, want %s after ForceOpen", cb.GetState(), StateOpen)
	}

	// 尝试执行，应该被拒绝
	err := cb.Execute(func() error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Execute() error = %v, want %v", err, ErrCircuitOpen)
	}
}

func TestCircuitBreaker_ForceClose(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	cb := NewCircuitBreaker(config)

	// 触发失败，打开熔断器
	cb.Execute(func() error {
		return errors.New("test error")
	})

	if cb.GetState() != StateOpen {
		t.Fatalf("State = %s, want %s", cb.GetState(), StateOpen)
	}

	// 强制关闭
	cb.ForceClose()

	if cb.GetState() != StateClosed {
		t.Errorf("State = %s, want %s after ForceClose", cb.GetState(), StateClosed)
	}

	// 应该可以正常执行
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil after ForceClose", err)
	}
}

func TestCircuitBreaker_StateChange_Callback(t *testing.T) {
	stateChanges := []CircuitState{}

	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.OnStateChange = func(from, to CircuitState) {
		stateChanges = append(stateChanges, to)
	}

	cb := NewCircuitBreaker(config)

	// 触发失败
	cb.Execute(func() error {
		return errors.New("test error")
	})

	// 验证状态变化被记录
	if len(stateChanges) != 1 {
		t.Errorf("Expected 1 state change, got %d", len(stateChanges))
	}

	if stateChanges[0] != StateOpen {
		t.Errorf("State change = %s, want %s", stateChanges[0], StateOpen)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 5
	cb := NewCircuitBreaker(config)

	// 执行一些请求
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return nil
		})
	}

	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	// 验证统计
	stats := cb.GetStats()
	if stats.TotalRequests != 5 {
		t.Errorf("TotalRequests = %d, want 5", stats.TotalRequests)
	}
	if stats.SuccessRequests != 3 {
		t.Errorf("SuccessRequests = %d, want 3", stats.SuccessRequests)
	}
	if stats.FailedRequests != 2 {
		t.Errorf("FailedRequests = %d, want 2", stats.FailedRequests)
	}
}

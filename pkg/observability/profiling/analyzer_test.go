package profiling

import (
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
)

func TestCollectMetrics(t *testing.T) {
	metrics := CollectMetrics()
	assert.NotNil(t, metrics)
	assert.Greater(t, metrics.CPUCount, 0)
	assert.Greater(t, metrics.GOMAXPROCS, 0)
	assert.Greater(t, metrics.MemoryAlloc, uint64(0))
	assert.Greater(t, metrics.GoroutineCount, 0)
}

func TestNewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()
	assert.NotNil(t, analyzer)
	assert.Nil(t, analyzer.baseline)
}

func TestAnalyzerSetBaseline(t *testing.T) {
	analyzer := NewAnalyzer()
	analyzer.SetBaseline()
	
	assert.NotNil(t, analyzer.baseline)
	assert.Greater(t, analyzer.baseline.MemoryAlloc, uint64(0))
}

func TestAnalyzerAnalyze(t *testing.T) {
	analyzer := NewAnalyzer()
	analyzer.SetBaseline()
	
	// 模拟一些内存分配
	data := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = make([]byte, 1024)
	}
	
	time.Sleep(10 * time.Millisecond)
	
	report := analyzer.Analyze()
	assert.NotNil(t, report)
	assert.NotNil(t, report.Current)
	assert.NotNil(t, report.Baseline)
	assert.NotNil(t, report.Delta)
	
	// 验证变化量
	assert.Greater(t, report.Delta.Duration, time.Duration(0))
}

func TestPerformanceReportCalculateDelta(t *testing.T) {
	baseline := &PerformanceMetrics{
		MemoryAlloc:      1000000,
		MemoryTotalAlloc: 2000000,
		MemorySys:        3000000,
		GoroutineCount:   10,
		MemoryNumGC:      5,
		Timestamp:        time.Now().Add(-1 * time.Second),
	}
	
	current := &PerformanceMetrics{
		MemoryAlloc:      2000000,
		MemoryTotalAlloc: 4000000,
		MemorySys:        5000000,
		GoroutineCount:   15,
		MemoryNumGC:      10,
		Timestamp:        time.Now(),
	}
	
	report := &PerformanceReport{
		Current:  current,
		Baseline: baseline,
	}
	
	report.CalculateDelta()
	
	assert.NotNil(t, report.Delta)
	assert.Equal(t, int64(1000000), report.Delta.MemoryAllocDelta)
	assert.Equal(t, int64(2000000), report.Delta.MemoryTotalAllocDelta)
	assert.Equal(t, int64(2000000), report.Delta.MemorySysDelta)
	assert.Equal(t, 5, report.Delta.GoroutineCountDelta)
	assert.Equal(t, 5, report.Delta.NumGCDelta)
}

func TestPerformanceReportDetectIssues(t *testing.T) {
	baseline := &PerformanceMetrics{
		MemoryAlloc:    1000000,
		GoroutineCount: 10,
		MemoryNumGC:    5,
		Timestamp:      time.Now().Add(-1 * time.Second),
	}
	
	// 测试内存泄漏检测
	t.Run("memory_leak", func(t *testing.T) {
		current := &PerformanceMetrics{
			MemoryAlloc:    200*1024*1024 + 1000000, // > 100MB 增长
			GoroutineCount: 10,
			MemoryNumGC:    5,
			Timestamp:      time.Now(),
		}
		
		report := &PerformanceReport{
			Current:  current,
			Baseline: baseline,
		}
		
		report.CalculateDelta()
		
		assert.NotEmpty(t, report.Issues)
		assert.Contains(t, report.Issues[0].Type, "memory_leak")
	})
	
	// 测试协程泄漏检测
	t.Run("goroutine_leak", func(t *testing.T) {
		current := &PerformanceMetrics{
			MemoryAlloc:    1000000,
			GoroutineCount: 1510, // > 1000 增长
			MemoryNumGC:    5,
			Timestamp:      time.Now(),
		}
		
		report := &PerformanceReport{
			Current:  current,
			Baseline: baseline,
		}
		
		report.CalculateDelta()
		
		assert.NotEmpty(t, report.Issues)
		hasGoroutineIssue := false
		for _, issue := range report.Issues {
			if issue.Type == "goroutine_leak" {
				hasGoroutineIssue = true
				break
			}
		}
		assert.True(t, hasGoroutineIssue)
	})
	
	// 测试高 GC 频率检测
	t.Run("high_gc", func(t *testing.T) {
		current := &PerformanceMetrics{
			MemoryAlloc:    1000000,
			GoroutineCount: 10,
			MemoryNumGC:    50, // 45 次 GC 在 1 秒内
			Timestamp:      time.Now(),
		}
		
		report := &PerformanceReport{
			Current:  current,
			Baseline: baseline,
		}
		
		report.CalculateDelta()
		
		hasHighGC := false
		for _, issue := range report.Issues {
			if issue.Type == "high_gc" {
				hasHighGC = true
				break
			}
		}
		assert.True(t, hasHighGC)
	})
}

func TestPerformanceReportString(t *testing.T) {
	metrics := CollectMetrics()
	report := &PerformanceReport{
		Current: metrics,
	}
	
	s := report.String()
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "Performance Report")
	assert.Contains(t, s, "Current Metrics")
	assert.Contains(t, s, "CPU Count")
	assert.Contains(t, s, "Memory Alloc")
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"bytes", 500, "500 B"},
		{"kilobytes", 1536, "1.50 KB"},
		{"megabytes", 1572864, "1.50 MB"},
		{"gigabytes", 1610612736, "1.50 GB"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{"nanoseconds", 500 * time.Nanosecond, "ns"},
		{"microseconds", 500 * time.Microsecond, "μs"},
		{"milliseconds", 500 * time.Millisecond, "ms"},
		{"seconds", 2 * time.Second, "s"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestAbs(t *testing.T) {
	assert.Equal(t, int64(10), abs(10))
	assert.Equal(t, int64(10), abs(-10))
	assert.Equal(t, int64(0), abs(0))
}

func TestNewBenchmark(t *testing.T) {
	benchmark := NewBenchmark("test")
	assert.NotNil(t, benchmark)
	assert.Equal(t, "test", benchmark.name)
	assert.NotNil(t, benchmark.analyzer)
	assert.NotNil(t, benchmark.analyzer.baseline)
}

func TestBenchmarkFinish(t *testing.T) {
	benchmark := NewBenchmark("test")
	
	// 模拟一些工作
	data := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		data[i] = make([]byte, 1024)
	}
	
	time.Sleep(10 * time.Millisecond)
	
	report := benchmark.Finish()
	assert.NotNil(t, report)
	assert.NotNil(t, report.Delta)
	assert.Greater(t, report.Delta.Duration, time.Duration(0))
}

func TestRunBenchmark(t *testing.T) {
	report := RunBenchmark("test-operation", func() {
		// 模拟一些工作
		data := make([][]byte, 100)
		for i := 0; i < 100; i++ {
			data[i] = make([]byte, 1024)
		}
		time.Sleep(10 * time.Millisecond)
	})
	
	assert.NotNil(t, report)
	assert.NotNil(t, report.Current)
	assert.NotNil(t, report.Delta)
	assert.Greater(t, report.Delta.Duration, time.Duration(0))
}

func TestPerformanceMetrics(t *testing.T) {
	metrics := &PerformanceMetrics{
		CPUCount:         4,
		GOMAXPROCS:       4,
		MemoryAlloc:      1000000,
		MemoryTotalAlloc: 2000000,
		MemorySys:        3000000,
		GoroutineCount:   10,
		Timestamp:        time.Now(),
	}
	
	assert.Equal(t, 4, metrics.CPUCount)
	assert.Equal(t, 4, metrics.GOMAXPROCS)
	assert.Equal(t, uint64(1000000), metrics.MemoryAlloc)
	assert.Equal(t, 10, metrics.GoroutineCount)
}

func TestPerformanceIssue(t *testing.T) {
	issue := PerformanceIssue{
		Severity:    "high",
		Type:        "memory_leak",
		Description: "Memory increased",
		Value:       "100MB",
	}
	
	assert.Equal(t, "high", issue.Severity)
	assert.Equal(t, "memory_leak", issue.Type)
	assert.NotEmpty(t, issue.Description)
	assert.NotEmpty(t, issue.Value)
}

package profiling

import (
	"fmt"
	"runtime"
	"time"
)

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// CPU 相关
	CPUCount    int     `json:"cpu_count"`
	GOMAXPROCS  int     `json:"gomaxprocs"`
	
	// 内存相关
	MemoryAlloc      uint64  `json:"memory_alloc"`       // 当前分配的字节数
	MemoryTotalAlloc uint64  `json:"memory_total_alloc"` // 累计分配的字节数
	MemorySys        uint64  `json:"memory_sys"`         // 从系统获取的字节数
	MemoryHeapAlloc  uint64  `json:"memory_heap_alloc"`  // 堆上分配的字节数
	MemoryHeapSys    uint64  `json:"memory_heap_sys"`    // 堆系统字节数
	MemoryHeapIdle   uint64  `json:"memory_heap_idle"`   // 空闲堆字节数
	MemoryHeapInuse  uint64  `json:"memory_heap_inuse"`  // 使用中的堆字节数
	MemoryNumGC      uint32  `json:"memory_num_gc"`      // GC 次数
	MemoryGCPause    uint64  `json:"memory_gc_pause"`    // GC 暂停时间（纳秒）
	
	// Goroutine 相关
	GoroutineCount int `json:"goroutine_count"`
	
	// 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// Analyzer 性能分析器
type Analyzer struct {
	baseline *PerformanceMetrics
}

// NewAnalyzer 创建性能分析器
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// SetBaseline 设置基准线
func (a *Analyzer) SetBaseline() {
	a.baseline = CollectMetrics()
}

// Analyze 分析当前性能指标
func (a *Analyzer) Analyze() *PerformanceReport {
	current := CollectMetrics()
	
	report := &PerformanceReport{
		Current:  current,
		Baseline: a.baseline,
	}
	
	if a.baseline != nil {
		report.CalculateDelta()
	}
	
	return report
}

// CollectMetrics 收集性能指标
func CollectMetrics() *PerformanceMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &PerformanceMetrics{
		CPUCount:         runtime.NumCPU(),
		GOMAXPROCS:       runtime.GOMAXPROCS(0),
		MemoryAlloc:      m.Alloc,
		MemoryTotalAlloc: m.TotalAlloc,
		MemorySys:        m.Sys,
		MemoryHeapAlloc:  m.HeapAlloc,
		MemoryHeapSys:    m.HeapSys,
		MemoryHeapIdle:   m.HeapIdle,
		MemoryHeapInuse:  m.HeapInuse,
		MemoryNumGC:      m.NumGC,
		MemoryGCPause:    m.PauseNs[(m.NumGC+255)%256],
		GoroutineCount:   runtime.NumGoroutine(),
		Timestamp:        time.Now(),
	}
}

// PerformanceReport 性能报告
type PerformanceReport struct {
	Current  *PerformanceMetrics `json:"current"`
	Baseline *PerformanceMetrics `json:"baseline,omitempty"`
	Delta    *PerformanceDelta   `json:"delta,omitempty"`
	Issues   []PerformanceIssue  `json:"issues,omitempty"`
}

// PerformanceDelta 性能变化
type PerformanceDelta struct {
	MemoryAllocDelta     int64   `json:"memory_alloc_delta"`
	MemoryTotalAllocDelta int64  `json:"memory_total_alloc_delta"`
	MemorySysDelta       int64   `json:"memory_sys_delta"`
	GoroutineCountDelta  int     `json:"goroutine_count_delta"`
	NumGCDelta           int     `json:"num_gc_delta"`
	Duration             time.Duration `json:"duration"`
}

// CalculateDelta 计算变化量
func (pr *PerformanceReport) CalculateDelta() {
	if pr.Baseline == nil {
		return
	}
	
	pr.Delta = &PerformanceDelta{
		MemoryAllocDelta:      int64(pr.Current.MemoryAlloc) - int64(pr.Baseline.MemoryAlloc),
		MemoryTotalAllocDelta: int64(pr.Current.MemoryTotalAlloc) - int64(pr.Baseline.MemoryTotalAlloc),
		MemorySysDelta:        int64(pr.Current.MemorySys) - int64(pr.Baseline.MemorySys),
		GoroutineCountDelta:   pr.Current.GoroutineCount - pr.Baseline.GoroutineCount,
		NumGCDelta:            int(pr.Current.MemoryNumGC) - int(pr.Baseline.MemoryNumGC),
		Duration:              pr.Current.Timestamp.Sub(pr.Baseline.Timestamp),
	}
	
	// 检测性能问题
	pr.DetectIssues()
}

// PerformanceIssue 性能问题
type PerformanceIssue struct {
	Severity    string `json:"severity"`    // high, medium, low
	Type        string `json:"type"`        // memory_leak, goroutine_leak, high_gc
	Description string `json:"description"`
	Value       string `json:"value"`
}

// DetectIssues 检测性能问题
func (pr *PerformanceReport) DetectIssues() {
	if pr.Delta == nil {
		return
	}
	
	pr.Issues = []PerformanceIssue{}
	
	// 检测内存泄漏（分配持续增长但不释放）
	if pr.Delta.MemoryAllocDelta > 100*1024*1024 { // > 100MB
		pr.Issues = append(pr.Issues, PerformanceIssue{
			Severity:    "high",
			Type:        "memory_leak",
			Description: "Memory allocation increased significantly",
			Value:       fmt.Sprintf("+%d MB", pr.Delta.MemoryAllocDelta/(1024*1024)),
		})
	}
	
	// 检测协程泄漏
	if pr.Delta.GoroutineCountDelta > 1000 {
		pr.Issues = append(pr.Issues, PerformanceIssue{
			Severity:    "high",
			Type:        "goroutine_leak",
			Description: "Goroutine count increased significantly",
			Value:       fmt.Sprintf("+%d goroutines", pr.Delta.GoroutineCountDelta),
		})
	}
	
	// 检测频繁 GC
	if pr.Delta.Duration > 0 {
		gcRate := float64(pr.Delta.NumGCDelta) / pr.Delta.Duration.Seconds()
		if gcRate > 10 { // 每秒超过 10 次 GC
			pr.Issues = append(pr.Issues, PerformanceIssue{
				Severity:    "medium",
				Type:        "high_gc",
				Description: "High GC frequency detected",
				Value:       fmt.Sprintf("%.2f GC/sec", gcRate),
			})
		}
	}
	
	// 检测高内存使用
	memoryUsagePercent := float64(pr.Current.MemoryHeapInuse) / float64(pr.Current.MemoryHeapSys) * 100
	if memoryUsagePercent > 90 {
		pr.Issues = append(pr.Issues, PerformanceIssue{
			Severity:    "medium",
			Type:        "high_memory",
			Description: "High memory usage detected",
			Value:       fmt.Sprintf("%.1f%% heap usage", memoryUsagePercent),
		})
	}
}

// String 返回性能报告的字符串表示
func (pr *PerformanceReport) String() string {
	s := "Performance Report\n"
	s += "==================\n\n"
	
	// 当前指标
	s += fmt.Sprintf("Current Metrics:\n")
	s += fmt.Sprintf("  - CPU Count: %d\n", pr.Current.CPUCount)
	s += fmt.Sprintf("  - GOMAXPROCS: %d\n", pr.Current.GOMAXPROCS)
	s += fmt.Sprintf("  - Memory Alloc: %s\n", formatBytes(pr.Current.MemoryAlloc))
	s += fmt.Sprintf("  - Memory Total Alloc: %s\n", formatBytes(pr.Current.MemoryTotalAlloc))
	s += fmt.Sprintf("  - Memory Sys: %s\n", formatBytes(pr.Current.MemorySys))
	s += fmt.Sprintf("  - Heap Alloc: %s\n", formatBytes(pr.Current.MemoryHeapAlloc))
	s += fmt.Sprintf("  - Heap Sys: %s\n", formatBytes(pr.Current.MemoryHeapSys))
	s += fmt.Sprintf("  - Heap Inuse: %s\n", formatBytes(pr.Current.MemoryHeapInuse))
	s += fmt.Sprintf("  - Heap Idle: %s\n", formatBytes(pr.Current.MemoryHeapIdle))
	s += fmt.Sprintf("  - Num GC: %d\n", pr.Current.MemoryNumGC)
	s += fmt.Sprintf("  - GC Pause: %s\n", formatDuration(time.Duration(pr.Current.MemoryGCPause)))
	s += fmt.Sprintf("  - Goroutine Count: %d\n", pr.Current.GoroutineCount)
	s += fmt.Sprintf("  - Timestamp: %s\n", pr.Current.Timestamp.Format(time.RFC3339))
	
	// 变化量
	if pr.Delta != nil {
		s += fmt.Sprintf("\nChanges (since baseline):\n")
		s += fmt.Sprintf("  - Duration: %s\n", pr.Delta.Duration)
		s += fmt.Sprintf("  - Memory Alloc: %+s\n", formatBytes(uint64(abs(pr.Delta.MemoryAllocDelta))))
		s += fmt.Sprintf("  - Memory Total Alloc: %+s\n", formatBytes(uint64(abs(pr.Delta.MemoryTotalAllocDelta))))
		s += fmt.Sprintf("  - Memory Sys: %+s\n", formatBytes(uint64(abs(pr.Delta.MemorySysDelta))))
		s += fmt.Sprintf("  - Goroutine Count: %+d\n", pr.Delta.GoroutineCountDelta)
		s += fmt.Sprintf("  - Num GC: %+d\n", pr.Delta.NumGCDelta)
	}
	
	// 性能问题
	if len(pr.Issues) > 0 {
		s += fmt.Sprintf("\nDetected Issues:\n")
		for _, issue := range pr.Issues {
			s += fmt.Sprintf("  - [%s] %s: %s (%s)\n", issue.Severity, issue.Type, issue.Description, issue.Value)
		}
	} else if pr.Delta != nil {
		s += fmt.Sprintf("\nNo performance issues detected.\n")
	}
	
	return s
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// abs 返回绝对值
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// Benchmark 基准测试
type Benchmark struct {
	name      string
	analyzer  *Analyzer
	startTime time.Time
}

// NewBenchmark 创建基准测试
func NewBenchmark(name string) *Benchmark {
	analyzer := NewAnalyzer()
	analyzer.SetBaseline()
	
	return &Benchmark{
		name:      name,
		analyzer:  analyzer,
		startTime: time.Now(),
	}
}

// Finish 完成基准测试
func (b *Benchmark) Finish() *PerformanceReport {
	report := b.analyzer.Analyze()
	report.Delta.Duration = time.Since(b.startTime)
	return report
}

// RunBenchmark 运行基准测试
func RunBenchmark(name string, fn func()) *PerformanceReport {
	benchmark := NewBenchmark(name)
	fn()
	return benchmark.Finish()
}

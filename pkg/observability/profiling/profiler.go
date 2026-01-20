package profiling

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"
)

// ProfileType 性能分析类型
type ProfileType string

const (
	ProfileCPU        ProfileType = "cpu"
	ProfileMemory     ProfileType = "memory"
	ProfileGoroutine  ProfileType = "goroutine"
	ProfileBlock      ProfileType = "block"
	ProfileMutex      ProfileType = "mutex"
	ProfileThreadCreate ProfileType = "threadcreate"
	ProfileAllocs     ProfileType = "allocs"
	ProfileHeap       ProfileType = "heap"
	ProfileTrace      ProfileType = "trace"
)

// ProfilerConfig 性能分析器配置
type ProfilerConfig struct {
	// OutputDir 输出目录
	OutputDir string
	
	// EnableCPU 启用 CPU 分析
	EnableCPU bool
	
	// EnableMemory 启用内存分析
	EnableMemory bool
	
	// EnableGoroutine 启用协程分析
	EnableGoroutine bool
	
	// EnableBlock 启用阻塞分析
	EnableBlock bool
	
	// EnableMutex 启用互斥锁分析
	EnableMutex bool
	
	// EnableTrace 启用执行追踪
	EnableTrace bool
	
	// CPUSampleRate CPU 采样率（Hz）
	CPUSampleRate int
	
	// MemorySampleRate 内存采样率（每 N 次分配采样一次）
	MemorySampleRate int
	
	// BlockSampleRate 阻塞采样率（每 N 纳秒采样一次）
	BlockSampleRate int
	
	// MutexSampleFraction 互斥锁采样率（0-1）
	MutexSampleFraction int
}

// DefaultProfilerConfig 返回默认配置
func DefaultProfilerConfig() ProfilerConfig {
	return ProfilerConfig{
		OutputDir:           "./profiles",
		EnableCPU:           true,
		EnableMemory:        true,
		EnableGoroutine:     true,
		EnableBlock:         false,
		EnableMutex:         false,
		EnableTrace:         false,
		CPUSampleRate:       100,
		MemorySampleRate:    512 * 1024,
		BlockSampleRate:     1,
		MutexSampleFraction: 1,
	}
}

// Profiler 性能分析器
type Profiler struct {
	config ProfilerConfig
	mu     sync.RWMutex
	
	// 运行状态
	cpuFile   *os.File
	traceFile *os.File
	isRunning bool
	startTime time.Time
}

// NewProfiler 创建性能分析器
func NewProfiler(config ProfilerConfig) (*Profiler, error) {
	// 创建输出目录
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	
	return &Profiler{
		config: config,
	}, nil
}

// Start 开始性能分析
func (p *Profiler) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.isRunning {
		return fmt.Errorf("profiler is already running")
	}
	
	// 设置采样率
	if p.config.EnableBlock {
		runtime.SetBlockProfileRate(p.config.BlockSampleRate)
	}
	
	if p.config.EnableMutex {
		runtime.SetMutexProfileFraction(p.config.MutexSampleFraction)
	}
	
	// 启动 CPU 分析
	if p.config.EnableCPU {
		cpuFile, err := p.createProfileFile(ProfileCPU)
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %w", err)
		}
		
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			cpuFile.Close()
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}
		
		p.cpuFile = cpuFile
	}
	
	// 启动执行追踪
	if p.config.EnableTrace {
		traceFile, err := p.createProfileFile(ProfileTrace)
		if err != nil {
			return fmt.Errorf("failed to create trace file: %w", err)
		}
		
		if err := trace.Start(traceFile); err != nil {
			traceFile.Close()
			return fmt.Errorf("failed to start trace: %w", err)
		}
		
		p.traceFile = traceFile
	}
	
	p.isRunning = true
	p.startTime = time.Now()
	
	return nil
}

// Stop 停止性能分析
func (p *Profiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.isRunning {
		return fmt.Errorf("profiler is not running")
	}
	
	// 停止 CPU 分析
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
		p.cpuFile = nil
	}
	
	// 停止执行追踪
	if p.traceFile != nil {
		trace.Stop()
		p.traceFile.Close()
		p.traceFile = nil
	}
	
	// 写入其他类型的 profile
	if p.config.EnableMemory {
		if err := p.writeProfile(ProfileMemory); err != nil {
			return fmt.Errorf("failed to write memory profile: %w", err)
		}
	}
	
	if p.config.EnableGoroutine {
		if err := p.writeProfile(ProfileGoroutine); err != nil {
			return fmt.Errorf("failed to write goroutine profile: %w", err)
		}
	}
	
	if p.config.EnableBlock {
		if err := p.writeProfile(ProfileBlock); err != nil {
			return fmt.Errorf("failed to write block profile: %w", err)
		}
	}
	
	if p.config.EnableMutex {
		if err := p.writeProfile(ProfileMutex); err != nil {
			return fmt.Errorf("failed to write mutex profile: %w", err)
		}
	}
	
	p.isRunning = false
	
	return nil
}

// WriteProfile 写入指定类型的 profile
func (p *Profiler) WriteProfile(profileType ProfileType) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.writeProfile(profileType)
}

// writeProfile 内部方法，不加锁
func (p *Profiler) writeProfile(profileType ProfileType) error {
	file, err := p.createProfileFile(profileType)
	if err != nil {
		return err
	}
	defer file.Close()
	
	switch profileType {
	case ProfileMemory, ProfileHeap:
		runtime.GC() // 触发 GC 以获取准确的内存统计
		return pprof.WriteHeapProfile(file)
		
	case ProfileGoroutine:
		profile := pprof.Lookup("goroutine")
		if profile == nil {
			return fmt.Errorf("goroutine profile not found")
		}
		return profile.WriteTo(file, 0)
		
	case ProfileBlock:
		profile := pprof.Lookup("block")
		if profile == nil {
			return fmt.Errorf("block profile not found")
		}
		return profile.WriteTo(file, 0)
		
	case ProfileMutex:
		profile := pprof.Lookup("mutex")
		if profile == nil {
			return fmt.Errorf("mutex profile not found")
		}
		return profile.WriteTo(file, 0)
		
	case ProfileThreadCreate:
		profile := pprof.Lookup("threadcreate")
		if profile == nil {
			return fmt.Errorf("threadcreate profile not found")
		}
		return profile.WriteTo(file, 0)
		
	case ProfileAllocs:
		profile := pprof.Lookup("allocs")
		if profile == nil {
			return fmt.Errorf("allocs profile not found")
		}
		return profile.WriteTo(file, 0)
		
	default:
		return fmt.Errorf("unsupported profile type: %s", profileType)
	}
}

// createProfileFile 创建 profile 文件
func (p *Profiler) createProfileFile(profileType ProfileType) (*os.File, error) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.prof", profileType, timestamp)
	filepath := filepath.Join(p.config.OutputDir, filename)
	
	return os.Create(filepath)
}

// IsRunning 检查是否正在运行
func (p *Profiler) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// GetDuration 获取运行时长
func (p *Profiler) GetDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.isRunning {
		return 0
	}
	
	return time.Since(p.startTime)
}

// ProfileOperation 对操作进行性能分析
func ProfileOperation(ctx context.Context, name string, config ProfilerConfig, fn func(ctx context.Context) error) error {
	profiler, err := NewProfiler(config)
	if err != nil {
		return fmt.Errorf("failed to create profiler: %w", err)
	}
	
	// 启动性能分析
	if err := profiler.Start(); err != nil {
		return fmt.Errorf("failed to start profiler: %w", err)
	}
	
	// 执行操作
	operationErr := fn(ctx)
	
	// 停止性能分析
	if err := profiler.Stop(); err != nil {
		return fmt.Errorf("failed to stop profiler: %w", err)
	}
	
	return operationErr
}

// DumpProfile 快速导出指定类型的 profile
func DumpProfile(profileType ProfileType, writer io.Writer) error {
	switch profileType {
	case ProfileCPU:
		return fmt.Errorf("CPU profile requires Start/Stop, use Profiler")
		
	case ProfileMemory, ProfileHeap:
		runtime.GC()
		return pprof.WriteHeapProfile(writer)
		
	case ProfileGoroutine, ProfileBlock, ProfileMutex, ProfileThreadCreate, ProfileAllocs:
		profile := pprof.Lookup(string(profileType))
		if profile == nil {
			return fmt.Errorf("profile %s not found", profileType)
		}
		return profile.WriteTo(writer, 0)
		
	case ProfileTrace:
		return fmt.Errorf("trace requires Start/Stop, use Profiler")
		
	default:
		return fmt.Errorf("unsupported profile type: %s", profileType)
	}
}

// GetMemoryStats 获取内存统计信息
func GetMemoryStats() *runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return &m
}

// GetGoroutineCount 获取当前协程数量
func GetGoroutineCount() int {
	return runtime.NumGoroutine()
}

// GetCPUCount 获取 CPU 数量
func GetCPUCount() int {
	return runtime.NumCPU()
}

// ForceGC 强制执行垃圾回收
func ForceGC() {
	runtime.GC()
}

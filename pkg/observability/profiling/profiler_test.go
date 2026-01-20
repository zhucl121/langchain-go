package profiling

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfiler(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ProfilerConfig{
		OutputDir:    tmpDir,
		EnableCPU:    true,
		EnableMemory: true,
	}
	
	profiler, err := NewProfiler(config)
	require.NoError(t, err)
	assert.NotNil(t, profiler)
	assert.Equal(t, tmpDir, profiler.config.OutputDir)
}

func TestProfilerStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ProfilerConfig{
		OutputDir:    tmpDir,
		EnableCPU:    true,
		EnableMemory: true,
	}
	
	profiler, err := NewProfiler(config)
	require.NoError(t, err)
	
	// 测试启动
	err = profiler.Start()
	assert.NoError(t, err)
	assert.True(t, profiler.IsRunning())
	
	// 模拟一些工作
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < 1000; i++ {
		_ = make([]byte, 1024)
	}
	
	// 测试停止
	err = profiler.Stop()
	assert.NoError(t, err)
	assert.False(t, profiler.IsRunning())
	
	// 验证文件创建
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Greater(t, len(files), 0)
}

func TestProfilerDoubleStart(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ProfilerConfig{
		OutputDir:    tmpDir,
		EnableCPU:    true,
	}
	
	profiler, err := NewProfiler(config)
	require.NoError(t, err)
	
	// 第一次启动应该成功
	err = profiler.Start()
	assert.NoError(t, err)
	
	// 第二次启动应该失败
	err = profiler.Start()
	assert.Error(t, err)
	
	profiler.Stop()
}

func TestProfilerWriteProfile(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ProfilerConfig{
		OutputDir:       tmpDir,
		EnableGoroutine: true,
	}
	
	profiler, err := NewProfiler(config)
	require.NoError(t, err)
	
	// 写入 goroutine profile
	err = profiler.WriteProfile(ProfileGoroutine)
	assert.NoError(t, err)
	
	// 验证文件创建
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0].Name(), "goroutine")
}

func TestProfileOperation(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ProfilerConfig{
		OutputDir:    tmpDir,
		EnableCPU:    true,
		EnableMemory: true,
	}
	
	err := ProfileOperation(context.Background(), "test-operation", config, func(ctx context.Context) error {
		// 模拟一些工作
		time.Sleep(50 * time.Millisecond)
		for i := 0; i < 1000; i++ {
			_ = make([]byte, 1024)
		}
		return nil
	})
	
	assert.NoError(t, err)
	
	// 验证文件创建
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Greater(t, len(files), 0)
}

func TestGetMemoryStats(t *testing.T) {
	stats := GetMemoryStats()
	assert.NotNil(t, stats)
	assert.Greater(t, stats.Alloc, uint64(0))
	assert.Greater(t, stats.TotalAlloc, uint64(0))
	assert.Greater(t, stats.Sys, uint64(0))
}

func TestGetGoroutineCount(t *testing.T) {
	count := GetGoroutineCount()
	assert.Greater(t, count, 0)
}

func TestGetCPUCount(t *testing.T) {
	count := GetCPUCount()
	assert.Greater(t, count, 0)
}

func TestForceGC(t *testing.T) {
	// 应该不会 panic
	ForceGC()
}

func TestDefaultProfilerConfig(t *testing.T) {
	config := DefaultProfilerConfig()
	assert.Equal(t, "./profiles", config.OutputDir)
	assert.True(t, config.EnableCPU)
	assert.True(t, config.EnableMemory)
	assert.True(t, config.EnableGoroutine)
	assert.Equal(t, 100, config.CPUSampleRate)
}

func TestProfilerCreateProfileFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ProfilerConfig{
		OutputDir: tmpDir,
	}
	
	profiler, err := NewProfiler(config)
	require.NoError(t, err)
	
	file, err := profiler.createProfileFile(ProfileCPU)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	
	// 验证文件路径
	assert.Equal(t, tmpDir, filepath.Dir(file.Name()))
	assert.Contains(t, file.Name(), "cpu")
	
	file.Close()
}

func TestProfilerGetDuration(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := ProfilerConfig{
		OutputDir: tmpDir,
		EnableCPU: true,
	}
	
	profiler, err := NewProfiler(config)
	require.NoError(t, err)
	
	// 未启动时应该返回 0
	assert.Equal(t, time.Duration(0), profiler.GetDuration())
	
	// 启动后应该返回正数
	profiler.Start()
	time.Sleep(100 * time.Millisecond)
	duration := profiler.GetDuration()
	assert.Greater(t, duration, time.Duration(0))
	
	profiler.Stop()
}

func TestProfileTypes(t *testing.T) {
	types := []ProfileType{
		ProfileCPU,
		ProfileMemory,
		ProfileGoroutine,
		ProfileBlock,
		ProfileMutex,
		ProfileThreadCreate,
		ProfileAllocs,
		ProfileHeap,
		ProfileTrace,
	}
	
	for _, ptype := range types {
		assert.NotEmpty(t, string(ptype))
	}
}

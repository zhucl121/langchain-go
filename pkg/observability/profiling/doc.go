// Package profiling 提供性能分析和监控工具。
//
// 该包提供了以下功能：
//
// 1. 性能分析（Profiler）
//   - CPU 分析
//   - 内存分析
//   - Goroutine 分析
//   - 阻塞分析
//   - 互斥锁分析
//   - 执行追踪
//
// 2. 性能指标收集（Analyzer）
//   - 实时收集系统指标
//   - 性能基准对比
//   - 自动问题检测
//
// 3. 基准测试（Benchmark）
//   - 代码性能测试
//   - 性能报告生成
//
// # 基本使用
//
// 性能分析示例：
//
//	config := profiling.DefaultProfilerConfig()
//	config.EnableCPU = true
//	config.EnableMemory = true
//
//	profiler, _ := profiling.NewProfiler(config)
//	profiler.Start()
//
//	// 执行需要分析的代码
//	// ...
//
//	profiler.Stop()
//
// 性能监控示例：
//
//	analyzer := profiling.NewAnalyzer()
//	analyzer.SetBaseline()
//
//	// 执行操作
//	// ...
//
//	report := analyzer.Analyze()
//	fmt.Println(report)
//
// 基准测试示例：
//
//	report := profiling.RunBenchmark("my-operation", func() {
//	    // 执行需要测试的代码
//	})
//	fmt.Println(report)
//
package profiling

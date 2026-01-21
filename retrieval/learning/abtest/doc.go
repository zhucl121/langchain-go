// Package abtest 提供 A/B 测试框架。
//
// 该包实现了学习型检索系统的 A/B 测试组件，支持：
//
// 1. 实验管理
// 2. 用户分流
// 3. 结果记录
// 4. 统计分析
//
// 基本用法：
//
//	// 创建 A/B 测试管理器
//	manager := abtest.NewManager(storage)
//
//	// 创建实验
//	experiment := &abtest.Experiment{
//	    ID:   "exp-001",
//	    Name: "检索策略对比",
//	    Variants: []abtest.Variant{
//	        {ID: "control", Name: "当前策略", Strategy: "hybrid", Weight: 0.5},
//	        {ID: "treatment", Name: "新策略", Strategy: "vector", Weight: 0.5},
//	    },
//	    Traffic: 1.0, // 100% 流量参与实验
//	}
//	manager.CreateExperiment(ctx, experiment)
//
//	// 用户访问时分配变体
//	variantID, _ := manager.AssignVariant(ctx, userID, experimentID)
//
//	// 记录结果
//	manager.RecordResult(ctx, &abtest.ExperimentResult{
//	    ExperimentID: experimentID,
//	    VariantID:    variantID,
//	    UserID:       userID,
//	    Metrics:      metrics,
//	})
//
//	// 分析实验结果
//	analysis, _ := manager.AnalyzeExperiment(ctx, experimentID)
//	fmt.Printf("获胜者: %s, 置信度: %.2f%%\n",
//	    analysis.Winner, analysis.Confidence*100)
//
// 核心概念：
//
// 实验（Experiment）：
//   - 定义实验的目标和参与者
//   - 包含多个变体（通常是对照组和实验组）
//   - 设置流量分配比例
//
// 变体（Variant）：
//   - 实验中的不同版本
//   - 可以是不同的策略、参数或配置
//   - 每个变体有独立的权重
//
// 用户分流：
//   - 使用一致性哈希确保用户始终看到相同变体
//   - 支持流量控制（部分用户参与实验）
//   - 支持白名单/黑名单
//
// 统计分析：
//   - t-test 检验统计显著性
//   - 计算置信区间
//   - 提供可视化数据
//
package abtest

# Learning Retrieval - 参数优化示例

这个示例展示如何使用贝叶斯优化自动调整检索系统的参数。

## 功能演示

1. **参数空间定义** - 定义可优化的参数及其范围
2. **贝叶斯优化** - 智能搜索最佳参数组合
3. **性能评估** - 对比优化前后的性能
4. **参数验证** - 验证参数合法性
5. **参数建议** - 基于历史数据建议参数

## 运行示例

```bash
cd examples/learning_optimization_demo
go run main.go
```

## 输出示例

```
=== LangChain-Go Learning Retrieval - 参数优化示例 ===

1. 准备测试数据...
✓ 已创建 20 个测试查询 (策略: hybrid-search)

2. 评估当前性能...
当前性能:
  📊 综合得分: 0.180
  🎯 NDCG: 0.500
  ⭐ 平均评分: 4.0/5.0
  📈 点击率: 30.0%

3. 创建参数优化器...
✓ 优化器已创建

4. 定义参数空间...
  • top_k (int): 5.0 - 30.0 (默认: 10)
  • temperature (float): 0.1 - 1.0 (默认: 0.7)
  • rerank_strategy (choice): [score diversity mmr] (默认: score)

5. 运行贝叶斯优化...
这将尝试 20 次不同的参数组合...
✓ 优化完成！

6. 优化结果:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎯 最佳参数:
  • top_k: 18
  • temperature: 0.65
  • rerank_strategy: mmr

📊 性能提升:
  优化前得分: 0.180
  优化后得分: 0.230
  提升幅度:   27.78%

⏱️  优化统计:
  迭代次数: 20
  耗时:     50ms

✨ 显著提升！建议应用优化后的参数

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

7. 优化历史（前 5 次迭代）:
  迭代  1: 得分 0.185, 参数 {top_k:12, temperature:0.4, ...}
  迭代  2: 得分 0.192, 参数 {top_k:15, temperature:0.6, ...}
  迭代  3: 得分 0.175, 参数 {top_k:8, temperature:0.9, ...}
  迭代  4: 得分 0.205, 参数 {top_k:20, temperature:0.7, ...}
  迭代  5: 得分 0.220, 参数 {top_k:18, temperature:0.65, ...}
  ... 还有 15 次迭代

8. 参数验证示例:
  ✅ 参数有效: map[top_k:15 temperature:0.8 rerank_strategy:mmr]
  ❌ 参数无效: parameter top_k out of range [5, 30]

9. 获取参数建议:
  💡 建议参数: map[top_k:18 temperature:0.65 rerank_strategy:mmr]

=== 示例完成 ===
```

## 核心概念

### 参数类型

支持三种参数类型：

1. **Int** - 整数参数
   ```go
   {
       Name:    "top_k",
       Type:    ParamTypeInt,
       Min:     5,
       Max:     30,
       Default: 10,
   }
   ```

2. **Float** - 浮点参数
   ```go
   {
       Name:    "temperature",
       Type:    ParamTypeFloat,
       Min:     0.1,
       Max:     1.0,
       Default: 0.7,
   }
   ```

3. **Choice** - 离散选择
   ```go
   {
       Name:    "strategy",
       Type:    ParamTypeChoice,
       Values:  []string{"hybrid", "vector", "graph"},
       Default: "hybrid",
   }
   ```

### 优化算法

使用**贝叶斯优化**进行智能参数搜索：

- **探索阶段**：随机采样参数空间
- **利用阶段**：在高分区域密集搜索
- **平衡策略**：通过 `ExplorationRatio` 控制

**优势**：
- 样本效率高（通常 20-50 次迭代足够）
- 不需要梯度信息
- 适合黑盒优化

### 优化选项

```go
OptimizeOptions{
    MaxIterations:    50,              // 最大迭代次数
    TargetMetric:     "overall_score", // 目标指标
    MinSampleSize:    30,              // 最小样本数
    AcquisitionType:  "EI",            // 采集函数类型
    ExplorationRatio: 0.1,             // 探索比例 (0-1)
}
```

**目标指标选项**：
- `overall_score`: 综合评分（推荐）
- `ndcg`: 归一化折损累计增益
- `mrr`: 平均倒数排名
- `avg_rating`: 平均用户评分

### 自动调优

持续监控性能并自动优化：

```go
go optimizer.AutoTune(ctx, strategyID, paramSpace, optimization.AutoTuneConfig{
    CheckInterval:  1 * time.Hour,  // 每小时检查一次
    ScoreThreshold: 0.7,            // 低于 0.7 时触发优化
    MinSampleSize:  30,             // 最小样本数要求
    OptimizeOptions: optimization.OptimizeOptions{
        MaxIterations: 30,
    },
})
```

**工作流程**：
1. 定期评估当前性能
2. 如果性能低于阈值，触发优化
3. 应用优化后的参数
4. 继续监控

## 使用示例

### 基础用法

```go
// 1. 创建优化器
optimizer := optimization.NewOptimizer(evaluator, collector, 
    optimization.DefaultConfig())

// 2. 定义参数空间
paramSpace := optimization.ParameterSpace{
    Params: []optimization.Parameter{
        {Name: "top_k", Type: optimization.ParamTypeInt, 
         Min: 5, Max: 20, Default: 10},
        {Name: "temperature", Type: optimization.ParamTypeFloat, 
         Min: 0.1, Max: 1.0, Default: 0.7},
    },
}

// 3. 运行优化
result, _ := optimizer.Optimize(ctx, "my-strategy", paramSpace, 
    optimization.OptimizeOptions{
        MaxIterations: 50,
    })

// 4. 应用最佳参数
fmt.Printf("最佳参数: %v\n", result.BestParams)
fmt.Printf("性能提升: %.2f%%\n", result.Improvement)
```

### 参数验证

```go
params := map[string]interface{}{
    "top_k":       15,
    "temperature": 0.8,
}

if err := optimizer.ValidateParams(params, paramSpace); err != nil {
    fmt.Printf("参数无效: %v\n", err)
} else {
    // 应用参数
}
```

### 获取建议

```go
// 基于历史优化结果建议参数
suggested, _ := optimizer.SuggestParams(ctx, strategyID, paramSpace)
fmt.Printf("建议参数: %v\n", suggested)
```

## 优化策略

### 1. 参数范围设置

- **不要太窄**：给算法足够的探索空间
- **不要太宽**：避免浪费计算资源
- **参考经验**：使用领域知识设置合理范围

### 2. 迭代次数选择

| 参数个数 | 建议迭代次数 |
|---------|-------------|
| 1-2 个  | 20-30 次    |
| 3-4 个  | 30-50 次    |
| 5+ 个   | 50-100 次   |

### 3. 采样策略

- **前期**：多探索（ExplorationRatio = 0.2-0.3）
- **中期**：平衡（ExplorationRatio = 0.1-0.15）
- **后期**：多利用（ExplorationRatio = 0.05-0.1）

### 4. 目标指标选择

- **用户满意度优先**：使用 `overall_score` 或 `avg_rating`
- **相关性优先**：使用 `ndcg` 或 `mrr`
- **多目标**：加权组合多个指标

## 应用场景

1. **初始参数调优**
   - 新系统上线前优化参数
   - 找到最佳默认配置

2. **持续优化**
   - 定期重新优化（如每周/每月）
   - 适应数据分布变化

3. **A/B 测试配合**
   - 为不同用户群体优化不同参数
   - 对比不同策略的参数敏感度

4. **多场景优化**
   - 不同场景（新闻、商品、学术等）使用不同参数
   - 自动适应场景特点

## 性能考虑

- **优化耗时**：取决于迭代次数和评估速度
  - 20 次迭代通常在 1 秒内完成
  - 可以异步运行

- **内存占用**：存储优化历史
  - 每次优化约 1-10 KB
  - 建议定期清理旧历史

- **并发安全**：优化器是并发安全的
  - 可以多个 goroutine 同时使用
  - 内部使用 sync.RWMutex 保护

## 限制和注意事项

1. **样本需求**
   - 需要足够的历史数据（建议 > 30 个查询）
   - 数据分布应该具有代表性

2. **评估准确性**
   - 优化效果依赖于评估指标的准确性
   - 建议使用多个指标综合评估

3. **参数独立性**
   - 假设参数之间相对独立
   - 强相关参数可能需要联合优化

4. **局部最优**
   - 可能收敛到局部最优
   - 建议多次运行或增加探索比例

## 下一步

- 查看 **Phase 4: A/B 测试框架** 了解如何对比策略
- 结合反馈收集和质量评估实现完整流程
- 在生产环境中使用自动调优功能

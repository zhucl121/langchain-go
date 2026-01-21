# Learning Retrieval - A/B 测试示例

这个示例展示如何使用 A/B 测试框架对比不同检索策略的效果。

## 功能演示

1. **创建实验** - 定义对照组和实验组
2. **用户分流** - 一致性哈希确保用户体验一致
3. **数据收集** - 记录各变体的性能指标
4. **统计分析** - t-test 检验显著性
5. **实验管理** - 完整的生命周期管理

## 运行示例

```bash
cd examples/learning_abtest_demo
go run main.go
```

## 输出示例

```
=== LangChain-Go Learning Retrieval - A/B 测试示例 ===

1. 创建 A/B 测试实验
✓ 实验创建成功
  实验 ID: exp-search-strategy
  实验名称: 检索策略对比实验
  变体数: 2
    • 对照组 - Hybrid Search (control) - 权重: 50%
    • 实验组 - Vector Search (treatment) - 权重: 50%

2. 开始实验
✓ 实验已开始运行

3. 用户分流演示
  用户 alice    -> 实验组 - Vector Search
  用户 bob      -> 对照组 - Hybrid Search
  用户 charlie  -> 实验组 - Vector Search
  用户 david    -> 对照组 - Hybrid Search
  用户 eve      -> 实验组 - Vector Search

4. 模拟收集实验数据...
  • 对照组收集中...
  • 实验组收集中...
✓ 数据收集完成 (每组 100 个样本)

5. 分析实验结果
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 各变体性能:

  对照组 - Hybrid Search:
    样本数:     100
    平均得分:   0.695
    标准差:     0.058
    置信区间:   [0.683, 0.706]

  实验组 - Vector Search:
    样本数:     100
    平均得分:   0.795
    标准差:     0.058
    置信区间:   [0.783, 0.806]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🏆 实验结论:
  获胜者: 实验组 - Vector Search
  置信度: 99.00%
  P-Value: 0.010
  ✅ 结果具有统计显著性 (p < 0.05)
  📈 性能提升: 14.39%

💡 建议: 可以将实验组策略推广到全量用户

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

6. 结束实验
✓ 实验已结束，获胜者: treatment

7. 查看实验列表
  总实验数: 3
    draft: 1
    running: 1
    ended: 1

=== 示例完成 ===
```

## 核心概念

### 实验（Experiment）

定义 A/B 测试的基本配置：

```go
experiment := &abtest.Experiment{
    ID:          "exp-001",
    Name:        "检索策略对比",
    Description: "对比不同策略的效果",
    Variants:    []abtest.Variant{...},
    Traffic:     1.0,  // 100% 流量参与
}
```

### 变体（Variant）

实验中的不同版本：

```go
variants := []abtest.Variant{
    {
        ID:       "control",
        Name:     "对照组",
        Strategy: "hybrid",
        Params:   map[string]interface{}{"top_k": 10},
        Weight:   0.5,  // 50% 流量
    },
    {
        ID:       "treatment",
        Name:     "实验组",
        Strategy: "vector",
        Params:   map[string]interface{}{"top_k": 15},
        Weight:   0.5,  // 50% 流量
    },
}
```

### 用户分流

使用一致性哈希确保：
- 同一用户始终看到相同变体
- 流量按权重分配
- 支持流量控制

```go
// 用户访问时分配变体
variantID, _ := manager.AssignVariant(ctx, userID, experimentID)

// 同一用户再次访问，返回相同变体
variantID2, _ := manager.AssignVariant(ctx, userID, experimentID)
// variantID == variantID2 ✓
```

### 统计分析

提供完整的统计分析：

- **样本大小**：每组的样本数
- **平均得分**：性能指标的平均值
- **标准差**：数据离散程度
- **置信区间**：95% 置信区间
- **P-Value**：统计显著性（< 0.05 表示显著）
- **获胜者**：性能更好的变体

## 使用示例

### 基础用法

```go
// 1. 创建管理器
storage := abtest.NewMemoryStorage()
manager := abtest.NewManager(storage)

// 2. 创建实验
experiment := &abtest.Experiment{
    ID:   "exp-001",
    Name: "策略对比",
    Variants: []abtest.Variant{
        {ID: "control", Weight: 0.5},
        {ID: "treatment", Weight: 0.5},
    },
    Traffic: 1.0,
}
manager.CreateExperiment(ctx, experiment)

// 3. 开始实验
manager.StartExperiment(ctx, "exp-001")

// 4. 用户分流
variantID, _ := manager.AssignVariant(ctx, userID, "exp-001")

// 5. 记录结果
manager.RecordResult(ctx, &abtest.ExperimentResult{
    ExperimentID: "exp-001",
    VariantID:    variantID,
    Metrics:      metrics,
})

// 6. 分析结果
analysis, _ := manager.AnalyzeExperiment(ctx, "exp-001")
fmt.Printf("获胜者: %s, 置信度: %.2f%%\n",
    analysis.Winner, analysis.Confidence*100)

// 7. 结束实验
manager.EndExperiment(ctx, "exp-001", analysis.Winner)
```

### 流量控制

```go
// 只让 20% 用户参与实验
experiment := &abtest.Experiment{
    ID:      "exp-001",
    Traffic: 0.2,  // 20% 流量
    Variants: []abtest.Variant{
        {ID: "control", Weight: 0.5},
        {ID: "treatment", Weight: 0.5},
    },
}
```

### 多变体实验

```go
// 3 个变体的实验
variants := []abtest.Variant{
    {ID: "control", Weight: 0.5},      // 50%
    {ID: "treatment-a", Weight: 0.25}, // 25%
    {ID: "treatment-b", Weight: 0.25}, // 25%
}
```

## 实验状态

| 状态 | 说明 | 操作 |
|------|------|------|
| **Draft** | 草稿 | 可以修改配置 |
| **Running** | 运行中 | 收集数据，不可修改关键配置 |
| **Paused** | 暂停 | 停止分流新用户 |
| **Ended** | 已结束 | 实验完成，不再收集数据 |

## 统计显著性

### P-Value 解读

- **p < 0.01**: 非常显著（99% 置信）
- **p < 0.05**: 显著（95% 置信）
- **p < 0.10**: 边缘显著（90% 置信）
- **p >= 0.10**: 不显著

### 样本量要求

| 效应大小 | 最小样本量（每组） |
|---------|-----------------|
| 大（>20%） | 30-50 |
| 中（10-20%） | 100-200 |
| 小（<10%） | 500+ |

## 最佳实践

### 1. 实验设计

- ✅ 明确实验假设和目标
- ✅ 控制变量，只改变一个因素
- ✅ 设置合理的实验周期
- ✅ 确定最小样本量
- ❌ 不要频繁查看结果（避免多重比较问题）

### 2. 流量分配

- ✅ 对照组保持现状
- ✅ 合理设置流量比例
- ✅ 考虑业务风险
- ❌ 不要在实验中途改变分配

### 3. 数据收集

- ✅ 记录完整的指标数据
- ✅ 监控异常值
- ✅ 确保数据质量
- ❌ 不要选择性记录数据

### 4. 结果分析

- ✅ 等待足够样本量
- ✅ 检查统计显著性
- ✅ 考虑业务价值
- ✅ 长期监控效果
- ❌ 不要只看单一指标

### 5. 常见陷阱

**新奇效应**：用户对新功能的短期好奇
- 解决：延长实验周期，观察趋势

**Simpson 悖论**：分组看到的趋势与整体相反
- 解决：检查用户分组的均衡性

**多重比较**：测试多个指标导致假阳性
- 解决：预先确定主要指标，调整显著性水平

**过早停止**：样本不足就下结论
- 解决：预先设定最小样本量和实验周期

## 应用场景

### 1. 策略对比

对比不同检索策略（hybrid vs vector vs graph）的效果

### 2. 参数调优

测试不同参数组合（top_k, temperature）的影响

### 3. 模型升级

验证新模型相比旧模型的提升

### 4. UI 变更

测试不同界面设计对用户行为的影响

### 5. 算法优化

对比不同排序算法、重排序策略

## 与其他模块集成

### 与反馈收集集成

```go
// 自动从反馈数据创建实验结果
queryFeedback, _ := collector.GetQueryFeedback(ctx, queryID)
metrics, _ := evaluator.EvaluateQuery(ctx, queryFeedback)

manager.RecordResult(ctx, &abtest.ExperimentResult{
    ExperimentID: experimentID,
    VariantID:    variantID,
    Metrics:      *metrics,
})
```

### 与参数优化集成

```go
// 使用优化器找到的最佳参数作为实验组
result, _ := optimizer.Optimize(ctx, strategyID, paramSpace, opts)

experiment := &abtest.Experiment{
    Variants: []abtest.Variant{
        {ID: "control", Params: currentParams},
        {ID: "treatment", Params: result.BestParams},
    },
}
```

## 高级功能

### 分层实验

```go
// 为不同用户群体运行不同实验
if userSegment == "premium" {
    variantID, _ = manager.AssignVariant(ctx, userID, "exp-premium")
} else {
    variantID, _ = manager.AssignVariant(ctx, userID, "exp-free")
}
```

### 实验互斥

```go
// 确保用户只参与一个实验
experiments := []string{"exp-001", "exp-002", "exp-003"}
assignedExp := experiments[hashUser(userID) % len(experiments)]
variantID, _ = manager.AssignVariant(ctx, userID, assignedExp)
```

## 下一步

- 结合反馈收集实现完整的实验流程
- 使用参数优化找到最佳配置后进行 A/B 测试验证
- 在生产环境中持续运行实验优化系统

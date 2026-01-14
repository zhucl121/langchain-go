// Package hitl 提供 Human-in-the-Loop (HITL) 功能。
//
// hitl 包实现了人机协作的核心机制，允许在图执行过程中
// 暂停执行，等待人类输入或审批，然后继续执行。
//
// # 核心概念
//
// Human-in-the-Loop 允许在自动化流程中引入人类决策：
//   - 中断执行等待输入
//   - 人类审批关键决策
//   - 修改执行状态
//   - 恢复执行
//
// # 核心组件
//
// 1. **Interrupt** - 中断机制
//   - 中断点定义
//   - 中断触发
//   - 中断状态管理
//
// 2. **ApprovalFlow** - 审批流程
//   - 审批请求
//   - 审批决策
//   - 审批历史
//
// 3. **ResumeManager** - 恢复管理
//   - 状态恢复
//   - 输入注入
//   - 续传执行
//
// 4. **InterruptHandler** - 中断处理器
//   - 中断回调
//   - 通知机制
//   - 超时处理
//
// # 基本使用
//
// 添加中断点：
//
//	graph := state.NewStateGraph[MyState]("my-graph")
//	graph.AddInterruptBefore("critical_decision")
//
// 处理中断：
//
//	result, err := graph.Invoke(ctx, initialState)
//	if errors.Is(err, hitl.ErrInterrupted) {
//	    // 获取中断信息
//	    interrupt := graph.GetCurrentInterrupt()
//	    
//	    // 获取人类输入
//	    approval := getHumanApproval(interrupt)
//	    
//	    // 恢复执行
//	    result, err = graph.Resume(ctx, approval)
//	}
//
// 审批流程：
//
//	approval := hitl.NewApprovalRequest("task-1", "需要审批此操作")
//	approval.WithOptions("approve", "reject", "modify")
//	
//	decision, err := approver.RequestApproval(ctx, approval)
//
// # 工作原理
//
// HITL 通过以下机制实现：
//   - 在指定节点前后设置中断点
//   - 执行到中断点时保存状态并暂停
//   - 等待人类输入或审批
//   - 根据输入修改状态或继续执行
//
package hitl

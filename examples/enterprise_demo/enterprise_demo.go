// Package main 演示 LangChain-Go v0.3.0 企业特性
//
// 此示例展示多模态、RBAC 和 HITL 的完整集成。
package main

import (
	"context"
	"fmt"
	"time"
	
	"github.com/zhucl121/langchain-go/graph/hitl"
	"github.com/zhucl121/langchain-go/pkg/auth"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
)

func main() {
	fmt.Println("=== LangChain-Go v0.3.0 企业特性演示 ===\n")
	
	ctx := context.Background()
	
	// 1. 多模态内容处理
	fmt.Println("━━━ 1. 多模态内容处理 ━━━")
	demoMultimodal()
	
	// 2. RBAC 系统
	fmt.Println("\n━━━ 2. RBAC 权限控制 ━━━")
	demoRBAC(ctx)
	
	// 3. 审批工作流
	fmt.Println("\n━━━ 3. 审批工作流 ━━━")
	demoWorkflow(ctx)
	
	// 4. 决策回滚
	fmt.Println("\n━━━ 4. 决策回滚 ━━━")
	demoRollback(ctx)
	
	// 5. 完整集成示例
	fmt.Println("\n━━━ 5. 完整集成演示 ━━━")
	demoIntegration(ctx)
	
	fmt.Println("\n✅ 演示完成！")
}

// demoMultimodal 演示多模态功能
func demoMultimodal() {
	// 创建多模态内容
	textContent := types.NewTextContent("产品描述")
	imageContent := types.NewImageContentFromData([]byte("fake image data"), types.ImageFormatJPEG)
	audioContent := types.NewAudioContentFromData([]byte("fake audio data"), types.AudioFormatMP3)
	
	// 创建多模态文档
	doc := loaders.NewMultimodalDocument("product_001",
		textContent,
		imageContent,
		audioContent,
	)
	
	fmt.Printf("  文档 ID: %s\n", doc.ID)
	fmt.Printf("  内容数量: %d\n", doc.ContentCount())
	fmt.Printf("  包含图像: %v\n", doc.HasImages())
	fmt.Printf("  包含音频: %v\n", doc.HasAudios())
	fmt.Printf("  总大小: %d bytes\n", doc.TotalSize())
	
	// 创建嵌入器（Mock）
	imageEmbed := embeddings.NewMockImageEmbedder(512)
	audioEmbed := embeddings.NewMockAudioEmbedder(512)
	
	ctx := context.Background()
	
	// 图像向量化
	imageData, _ := imageContent.GetImageData()
	imageVector, _ := imageEmbed.EmbedImage(ctx, imageData)
	fmt.Printf("  图像向量维度: %d\n", len(imageVector))
	
	// 音频向量化
	audioData, _ := audioContent.GetAudioData()
	audioVector, _ := audioEmbed.EmbedAudio(ctx, audioData)
	fmt.Printf("  音频向量维度: %d\n", len(audioVector))
}

// demoRBAC 演示 RBAC 功能
func demoRBAC(ctx context.Context) {
	// 创建管理器
	rbacMgr := auth.NewInMemoryRBACManager()
	tenantMgr := auth.NewInMemoryTenantManager()
	quotaMgr := auth.NewInMemoryQuotaManager(tenantMgr)
	
	// 创建租户
	tenant := &auth.Tenant{
		ID:   "tenant001",
		Name: "示例公司",
		Quota: &auth.ResourceQuota{
			APICallsPerDay: 1000,
			MaxDocuments:   5000,
		},
	}
	tenantMgr.CreateTenant(ctx, tenant)
	fmt.Printf("  创建租户: %s\n", tenant.Name)
	
	// 创建用户
	user := &auth.User{
		ID:       "user001",
		Name:     "张三",
		TenantID: "tenant001",
	}
	rbacMgr.CreateUser(ctx, user)
	rbacMgr.AssignRole(ctx, "user001", "user")
	fmt.Printf("  创建用户: %s (角色: user)\n", user.Name)
	
	// 检查权限
	err := rbacMgr.CheckPermission(ctx, "user001",
		auth.ResourceVectorStore,
		auth.ActionWrite,
		"",
	)
	
	if err == nil {
		fmt.Println("  ✅ 用户有写入向量存储的权限")
	} else {
		fmt.Println("  ❌ 用户没有权限")
	}
	
	// 检查配额
	err = quotaMgr.CheckQuota(ctx, "tenant001",
		auth.ResourceTypeAPICall,
		1,
	)
	
	if err == nil {
		fmt.Println("  ✅ 配额检查通过")
		quotaMgr.IncrementUsage(ctx, "tenant001", auth.ResourceTypeAPICall, 1)
	}
	
	// 查询使用情况
	usage, _ := quotaMgr.GetUsage(ctx, "tenant001")
	fmt.Printf("  API 调用使用量: %d/%d\n", usage.APICallsToday, tenant.Quota.APICallsPerDay)
}

// demoWorkflow 演示审批工作流
func demoWorkflow(ctx context.Context) {
	// 创建工作流引擎
	engine := hitl.NewWorkflowEngine()
	
	// 创建工作流
	workflow := hitl.NewApprovalWorkflow("wf001", "重要操作审批")
	
	step1 := hitl.NewApprovalStep("step1", "经理审批", []string{"manager"})
	step2 := hitl.NewApprovalStep("step2", "总监审批", []string{"director"})
	
	workflow.AddStep(step1)
	workflow.AddStep(step2)
	
	engine.CreateWorkflow(workflow)
	fmt.Printf("  创建工作流: %s\n", workflow.Title)
	
	// 启动工作流
	engine.StartWorkflow("wf001")
	fmt.Printf("  工作流状态: %s\n", workflow.Status)
	
	// 第一步审批
	decision1 := hitl.NewApprovalDecision("req001", hitl.ApprovalApproved)
	decision1.Comment = "技术可行"
	engine.SubmitApproval("wf001", "step1", "manager", decision1)
	fmt.Println("  ✅ 经理审批通过")
	
	// 第二步审批
	decision2 := hitl.NewApprovalDecision("req002", hitl.ApprovalApproved)
	decision2.Comment = "业务价值高"
	engine.SubmitApproval("wf001", "step2", "director", decision2)
	fmt.Println("  ✅ 总监审批通过")
	
	// 查看最终状态
	completedWorkflow, _ := engine.GetWorkflow("wf001")
	fmt.Printf("  最终状态: %s\n", completedWorkflow.Status)
}

// demoRollback 演示决策回滚
func demoRollback(ctx context.Context) {
	rollbackMgr := hitl.NewRollbackManager()
	recorder := hitl.NewInterventionRecorder()
	
	// 保存回滚点
	state := map[string]interface{}{
		"step":  "processing",
		"data":  "important_data",
		"count": 100,
	}
	
	point := hitl.NewRollbackPoint("rp001", "checkpoint_001", "process_node", state)
	point.Description = "处理前的安全点"
	rollbackMgr.SaveRollbackPoint(point)
	fmt.Printf("  保存回滚点: %s\n", point.ID)
	
	// 模拟操作
	fmt.Println("  执行操作...")
	time.Sleep(100 * time.Millisecond)
	
	// 发现问题，需要回滚
	fmt.Println("  ⚠️  检测到异常，执行回滚")
	action := hitl.NewRollbackAction("rp001", "数据异常", "admin")
	restoredPoint, _ := rollbackMgr.Rollback(ctx, action)
	
	fmt.Printf("  ✅ 回滚成功，恢复到: %s\n", restoredPoint.NodeName)
	fmt.Printf("  恢复状态: %v\n", restoredPoint.State)
	
	// 记录干预
	recorder.RecordIntervention(&hitl.InterventionRecord{
		Type:   hitl.InterventionTypeRollback,
		Actor:  "admin",
		Action: "回滚操作",
		Before: map[string]interface{}{"step": "failed"},
		After:  restoredPoint.State,
		Reason: "数据异常",
	})
	
	// 查看干预历史
	records := recorder.GetRecordsByType(hitl.InterventionTypeRollback)
	fmt.Printf("  回滚记录数: %d\n", len(records))
}

// demoIntegration 演示完整集成
func demoIntegration(ctx context.Context) {
	fmt.Println("  场景: 处理敏感多模态文档，需要审批和权限控制")
	
	// 1. 初始化系统
	rbacMgr := auth.NewInMemoryRBACManager()
	tenantMgr := auth.NewInMemoryTenantManager()
	quotaMgr := auth.NewInMemoryQuotaManager(tenantMgr)
	engine := hitl.NewWorkflowEngine()
	rollbackMgr := hitl.NewRollbackManager()
	
	// 2. 设置租户和用户
	tenant := &auth.Tenant{
		ID:   "corp001",
		Name: "企业客户",
		Quota: auth.DefaultResourceQuota(),
	}
	tenantMgr.CreateTenant(ctx, tenant)
	
	user := &auth.User{
		ID:       "alice",
		Name:     "Alice",
		TenantID: "corp001",
	}
	rbacMgr.CreateUser(ctx, user)
	rbacMgr.AssignRole(ctx, "alice", "user")
	
	// 3. 设置认证上下文
	authCtx := auth.ContextWithAuth(ctx, "alice", "corp001")
	
	// 4. 创建多模态文档
	doc := loaders.NewMultimodalDocument("sensitive_doc",
		types.NewTextContent("机密信息"),
		types.NewImageContentFromData([]byte("data"), types.ImageFormatJPEG),
	)
	
	// 5. 检查权限
	fmt.Print("  检查权限...")
	err := rbacMgr.CheckPermission(authCtx, "alice",
		auth.ResourceDocument,
		auth.ActionWrite,
		"",
	)
	if err != nil {
		fmt.Println(" 权限被拒绝")
		return
	}
	fmt.Println(" ✅")
	
	// 6. 检查配额
	fmt.Print("  检查配额...")
	err = quotaMgr.CheckQuota(authCtx, "corp001",
		auth.ResourceTypeDocuments,
		1,
	)
	if err != nil {
		fmt.Println(" 配额超限")
		return
	}
	fmt.Println(" ✅")
	
	// 7. 创建审批工作流
	fmt.Print("  创建审批工作流...")
	workflow := hitl.NewApprovalWorkflow("wf_sensitive", "敏感文档处理")
	workflow.AddStep(
		hitl.NewApprovalStep("review", "安全审查", []string{"security_officer"}),
	)
	
	engine.CreateWorkflow(workflow)
	engine.StartWorkflow("wf_sensitive")
	fmt.Println(" ✅")
	
	// 8. 保存回滚点
	fmt.Print("  保存回滚点...")
	point := hitl.NewRollbackPoint("rp_safe", "", "process", doc)
	rollbackMgr.SaveRollbackPoint(point)
	fmt.Println(" ✅")
	
	// 9. 模拟审批
	fmt.Print("  等待审批...")
	time.Sleep(100 * time.Millisecond)
	
	decision := hitl.NewApprovalDecision("req001", hitl.ApprovalApproved)
	decision.Comment = "安全检查通过"
	engine.SubmitApproval("wf_sensitive", "review", "security_officer", decision)
	fmt.Println(" ✅ 审批通过")
	
	// 10. 执行操作
	fmt.Print("  处理文档...")
	quotaMgr.IncrementUsage(authCtx, "corp001", auth.ResourceTypeDocuments, 1)
	time.Sleep(100 * time.Millisecond)
	fmt.Println(" ✅")
	
	// 11. 查看使用情况
	usage, _ := quotaMgr.GetUsage(authCtx, "corp001")
	fmt.Printf("  文档使用量: %d/%d\n", usage.DocumentCount, tenant.Quota.MaxDocuments)
	
	fmt.Println("\n  🎉 完整流程执行成功！")
	fmt.Println("     - 多模态文档处理")
	fmt.Println("     - 权限验证")
	fmt.Println("     - 配额管理")
	fmt.Println("     - 审批流程")
	fmt.Println("     - 回滚保护")
}

package hitl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RollbackPoint 回滚点
type RollbackPoint struct {
	// ID 回滚点 ID
	ID string
	
	// CheckpointID 检查点 ID
	CheckpointID string
	
	// NodeName 节点名称
	NodeName string
	
	// State 状态快照
	State any
	
	// Timestamp 创建时间
	Timestamp time.Time
	
	// Description 描述
	Description string
	
	// Metadata 元数据
	Metadata map[string]interface{}
}

// NewRollbackPoint 创建回滚点
func NewRollbackPoint(id, checkpointID, nodeName string, state any) *RollbackPoint {
	return &RollbackPoint{
		ID:           id,
		CheckpointID: checkpointID,
		NodeName:     nodeName,
		State:        state,
		Timestamp:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}
}

// RollbackAction 回滚操作
type RollbackAction struct {
	// TargetPointID 目标回滚点 ID
	TargetPointID string
	
	// Reason 回滚原因
	Reason string
	
	// Actor 操作人
	Actor string
	
	// Timestamp 回滚时间
	Timestamp time.Time
	
	// Metadata 元数据
	Metadata map[string]interface{}
}

// NewRollbackAction 创建回滚操作
func NewRollbackAction(targetPointID, reason, actor string) *RollbackAction {
	return &RollbackAction{
		TargetPointID: targetPointID,
		Reason:        reason,
		Actor:         actor,
		Timestamp:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}
}

// RollbackManager 回滚管理器
type RollbackManager struct {
	// rollbackPoints 回滚点存储
	rollbackPoints map[string]*RollbackPoint // key: rollbackPointID
	
	// rollbackHistory 回滚历史
	rollbackHistory []*RollbackAction
	
	mu sync.RWMutex
}

// NewRollbackManager 创建回滚管理器
func NewRollbackManager() *RollbackManager {
	return &RollbackManager{
		rollbackPoints:  make(map[string]*RollbackPoint),
		rollbackHistory: make([]*RollbackAction, 0),
	}
}

// SaveRollbackPoint 保存回滚点
func (rm *RollbackManager) SaveRollbackPoint(point *RollbackPoint) error {
	if point.ID == "" {
		return errors.New("rollback point ID is required")
	}
	
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.rollbackPoints[point.ID] = point
	return nil
}

// GetRollbackPoint 获取回滚点
func (rm *RollbackManager) GetRollbackPoint(pointID string) (*RollbackPoint, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	point, exists := rm.rollbackPoints[pointID]
	if !exists {
		return nil, fmt.Errorf("rollback point %s not found", pointID)
	}
	
	return point, nil
}

// ListRollbackPoints 列出回滚点
func (rm *RollbackManager) ListRollbackPoints(checkpointID string) []*RollbackPoint {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	points := make([]*RollbackPoint, 0)
	for _, point := range rm.rollbackPoints {
		if checkpointID == "" || point.CheckpointID == checkpointID {
			points = append(points, point)
		}
	}
	
	return points
}

// Rollback 执行回滚
func (rm *RollbackManager) Rollback(ctx context.Context, action *RollbackAction) (*RollbackPoint, error) {
	// 获取目标回滚点
	point, err := rm.GetRollbackPoint(action.TargetPointID)
	if err != nil {
		return nil, err
	}
	
	// 记录回滚历史
	rm.mu.Lock()
	rm.rollbackHistory = append(rm.rollbackHistory, action)
	rm.mu.Unlock()
	
	return point, nil
}

// GetRollbackHistory 获取回滚历史
func (rm *RollbackManager) GetRollbackHistory() []*RollbackAction {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	history := make([]*RollbackAction, len(rm.rollbackHistory))
	copy(history, rm.rollbackHistory)
	return history
}

// DeleteRollbackPoint 删除回滚点
func (rm *RollbackManager) DeleteRollbackPoint(pointID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	delete(rm.rollbackPoints, pointID)
	return nil
}

// DeleteOldRollbackPoints 删除过期的回滚点
func (rm *RollbackManager) DeleteOldRollbackPoints(before time.Time) int {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	count := 0
	for id, point := range rm.rollbackPoints {
		if point.Timestamp.Before(before) {
			delete(rm.rollbackPoints, id)
			count++
		}
	}
	
	return count
}

// InterventionRecord 人工干预记录
type InterventionRecord struct {
	// ID 记录 ID
	ID string
	
	// Type 干预类型
	Type InterventionType
	
	// WorkflowID 工作流 ID
	WorkflowID string
	
	// Actor 操作人
	Actor string
	
	// Action 操作
	Action string
	
	// Before 干预前的状态
	Before any
	
	// After 干预后的状态
	After any
	
	// Reason 原因
	Reason string
	
	// Timestamp 时间戳
	Timestamp time.Time
	
	// Metadata 元数据
	Metadata map[string]interface{}
}

// InterventionType 干预类型
type InterventionType string

const (
	// InterventionTypeApproval 审批
	InterventionTypeApproval InterventionType = "approval"
	
	// InterventionTypeRollback 回滚
	InterventionTypeRollback InterventionType = "rollback"
	
	// InterventionTypeModify 修改
	InterventionTypeModify InterventionType = "modify"
	
	// InterventionTypeCancel 取消
	InterventionTypeCancel InterventionType = "cancel"
)

// InterventionRecorder 干预记录器
type InterventionRecorder struct {
	records []* InterventionRecord
	mu      sync.RWMutex
}

// NewInterventionRecorder 创建干预记录器
func NewInterventionRecorder() *InterventionRecorder {
	return &InterventionRecorder{
		records: make([]*InterventionRecord, 0),
	}
}

// RecordIntervention 记录干预
func (ir *InterventionRecorder) RecordIntervention(record *InterventionRecord) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	
	record.Timestamp = time.Now()
	ir.records = append(ir.records, record)
}

// GetRecords 获取所有记录
func (ir *InterventionRecorder) GetRecords() []*InterventionRecord {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	
	records := make([]*InterventionRecord, len(ir.records))
	copy(records, ir.records)
	return records
}

// GetRecordsByActor 按操作人过滤记录
func (ir *InterventionRecorder) GetRecordsByActor(actor string) []*InterventionRecord {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	
	records := make([]*InterventionRecord, 0)
	for _, record := range ir.records {
		if record.Actor == actor {
			records = append(records, record)
		}
	}
	
	return records
}

// GetRecordsByType 按类型过滤记录
func (ir *InterventionRecorder) GetRecordsByType(interventionType InterventionType) []*InterventionRecord {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	
	records := make([]*InterventionRecord, 0)
	for _, record := range ir.records {
		if record.Type == interventionType {
			records = append(records, record)
		}
	}
	
	return records
}

// GetRecordsByWorkflow 按工作流过滤记录
func (ir *InterventionRecorder) GetRecordsByWorkflow(workflowID string) []*InterventionRecord {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	
	records := make([]*InterventionRecord, 0)
	for _, record := range ir.records {
		if record.WorkflowID == workflowID {
			records = append(records, record)
		}
	}
	
	return records
}

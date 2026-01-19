package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// AgentState Agent 状态。
//
// 用于保存和恢复 Agent 的执行状态。
type AgentState struct {
	// ID 状态 ID
	ID string

	// Input 输入问题
	Input string

	// Steps 执行步骤（别名为 History）
	Steps []AgentStep

	// History 执行历史（指向 Steps，用于向后兼容）
	History []AgentStep

	// Extra 额外数据
	Extra map[string]any

	// Context 上下文数据
	Context map[string]any

	// CurrentStep 当前步骤
	CurrentStep int

	// TotalSteps 总步数
	TotalSteps int

	// CreatedAt 创建时间
	CreatedAt time.Time

	// UpdatedAt 更新时间
	UpdatedAt time.Time

	// Status 状态 (running, paused, completed, failed)
	Status string

	// Metadata 元数据
	Metadata map[string]any
}

// AgentStatus Agent 状态类型。
type AgentStatus string

const (
	// StatusRunning 运行中
	StatusRunning AgentStatus = "running"

	// StatusPaused 已暂停
	StatusPaused AgentStatus = "paused"

	// StatusCompleted 已完成
	StatusCompleted AgentStatus = "completed"

	// StatusFailed 失败
	StatusFailed AgentStatus = "failed"
)

// StateStore Agent 状态存储接口。
//
// 用于持久化 Agent 状态。
type StateStore interface {
	// Save 保存状态
	Save(ctx context.Context, state *AgentState) error

	// Load 加载状态
	Load(ctx context.Context, id string) (*AgentState, error)

	// Delete 删除状态
	Delete(ctx context.Context, id string) error

	// List 列出所有状态
	List(ctx context.Context) ([]*AgentState, error)
}

// MemoryStateStore 内存状态存储。
//
// 仅用于测试和开发，生产环境应使用持久化存储。
type MemoryStateStore struct {
	states map[string]*AgentState
}

// NewMemoryStateStore 创建内存状态存储。
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		states: make(map[string]*AgentState),
	}
}

// Save 实现 StateStore 接口。
func (m *MemoryStateStore) Save(ctx context.Context, state *AgentState) error {
	if state.ID == "" {
		return fmt.Errorf("state ID cannot be empty")
	}

	// 更新时间
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}

	// 保存状态 (深拷贝)
	stateCopy := *state
	m.states[state.ID] = &stateCopy

	return nil
}

// Load 实现 StateStore 接口。
func (m *MemoryStateStore) Load(ctx context.Context, id string) (*AgentState, error) {
	state, exists := m.states[id]
	if !exists {
		return nil, fmt.Errorf("state not found: %s", id)
	}

	// 返回副本
	stateCopy := *state
	return &stateCopy, nil
}

// Delete 实现 StateStore 接口。
func (m *MemoryStateStore) Delete(ctx context.Context, id string) error {
	delete(m.states, id)
	return nil
}

// List 实现 StateStore 接口。
func (m *MemoryStateStore) List(ctx context.Context) ([]*AgentState, error) {
	states := make([]*AgentState, 0, len(m.states))
	for _, state := range m.states {
		stateCopy := *state
		states = append(states, &stateCopy)
	}
	return states, nil
}

// JSONStateStore JSON 文件状态存储。
//
// 将状态保存为 JSON 文件。
type JSONStateStore struct {
	basePath string
}

// NewJSONStateStore 创建 JSON 状态存储。
//
// 参数：
//   - basePath: 状态文件保存路径
//
func NewJSONStateStore(basePath string) *JSONStateStore {
	return &JSONStateStore{
		basePath: basePath,
	}
}

// Save 实现 StateStore 接口。
func (j *JSONStateStore) Save(ctx context.Context, state *AgentState) error {
	// 更新时间
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}

	// 序列化为 JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// 写入文件 (这里简化实现，实际应该使用 os 包)
	_ = data
	// TODO: 实现文件写入逻辑
	// err = os.WriteFile(filepath.Join(j.basePath, state.ID+".json"), data, 0644)
	
	return fmt.Errorf("JSON state store not fully implemented")
}

// Load 实现 StateStore 接口。
func (j *JSONStateStore) Load(ctx context.Context, id string) (*AgentState, error) {
	// TODO: 实现文件读取逻辑
	return nil, fmt.Errorf("JSON state store not fully implemented")
}

// Delete 实现 StateStore 接口。
func (j *JSONStateStore) Delete(ctx context.Context, id string) error {
	// TODO: 实现文件删除逻辑
	return fmt.Errorf("JSON state store not fully implemented")
}

// List 实现 StateStore 接口。
func (j *JSONStateStore) List(ctx context.Context) ([]*AgentState, error) {
	// TODO: 实现文件列表逻辑
	return nil, fmt.Errorf("JSON state store not fully implemented")
}

// StatefulExecutor 带状态管理的执行器。
type StatefulExecutor struct {
	executor *AgentExecutor
	store    StateStore
	stateID  string
}

// NewStatefulExecutor 创建带状态管理的执行器。
//
// 参数：
//   - executor: Agent 执行器
//   - store: 状态存储
//
// 返回：
//   - *StatefulExecutor: 状态执行器
//
func NewStatefulExecutor(executor *AgentExecutor, store StateStore) *StatefulExecutor {
	return &StatefulExecutor{
		executor: executor,
		store:    store,
		stateID:  generateStateID(),
	}
}

// RunWithState 带状态保存的执行。
//
// 参数：
//   - ctx: 上下文
//   - input: 输入问题
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
//
func (se *StatefulExecutor) RunWithState(ctx context.Context, input string) (*AgentResult, error) {
	// 创建初始状态
	state := &AgentState{
		ID:          se.stateID,
		Input:       input,
		History:     make([]AgentStep, 0),
		Context:     make(map[string]any),
		CurrentStep: 0,
		TotalSteps:  0,
		Status:      string(StatusRunning),
		Metadata:    make(map[string]any),
	}

	// 保存初始状态
	if err := se.store.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save initial state: %w", err)
	}

	// 执行 Agent
	result, err := se.executor.Run(ctx, input)

	// 更新最终状态
	if err != nil {
		state.Status = string(StatusFailed)
	} else {
		state.Status = string(StatusCompleted)
		state.History = result.Steps
		state.TotalSteps = result.TotalSteps
	}

	// 保存最终状态
	if saveErr := se.store.Save(ctx, state); saveErr != nil {
		// 记录保存错误但不影响结果
		fmt.Printf("Warning: failed to save final state: %v\n", saveErr)
	}

	return result, err
}

// SaveState 保存当前状态。
//
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - *AgentState: 保存的状态
//   - error: 错误
//
func (se *StatefulExecutor) SaveState(ctx context.Context) (*AgentState, error) {
	state := &AgentState{
		ID:       se.stateID,
		Status:   string(StatusPaused),
		Metadata: make(map[string]any),
	}

	if err := se.store.Save(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	return state, nil
}

// LoadState 加载状态。
//
// 参数：
//   - ctx: 上下文
//   - stateID: 状态 ID
//
// 返回：
//   - *AgentState: 加载的状态
//   - error: 错误
//
func (se *StatefulExecutor) LoadState(ctx context.Context, stateID string) (*AgentState, error) {
	state, err := se.store.Load(ctx, stateID)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// 更新当前状态 ID
	se.stateID = stateID

	return state, nil
}

// ResumeFromState 从保存的状态恢复执行。
//
// 参数：
//   - ctx: 上下文
//   - stateID: 状态 ID
//
// 返回：
//   - *AgentResult: 执行结果
//   - error: 错误
//
func (se *StatefulExecutor) ResumeFromState(ctx context.Context, stateID string) (*AgentResult, error) {
	// 加载状态
	state, err := se.LoadState(ctx, stateID)
	if err != nil {
		return nil, err
	}

	// 检查状态
	if state.Status == string(StatusCompleted) {
		return nil, fmt.Errorf("cannot resume completed agent")
	}

	// 继续执行 (简化实现，实际应该从中断点继续)
	return se.RunWithState(ctx, state.Input)
}

// generateStateID 生成状态 ID。
func generateStateID() string {
	return fmt.Sprintf("state_%d", time.Now().UnixNano())
}

// WithStateStore 配置状态存储。
//
// 参数：
//   - store: 状态存储
//
// 返回：
//   - AgentOption: 配置选项
//
func WithStateStore(store StateStore) AgentOption {
	return func(config *AgentConfig) {
		if config.Extra == nil {
			config.Extra = make(map[string]any)
		}
		config.Extra["state_store"] = store
	}
}

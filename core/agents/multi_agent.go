// Package agents provides multi-agent system capabilities for complex task coordination.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"langchain-go/core/chat"
	"langchain-go/pkg/types"
)

// MultiAgent 扩展了基础 Agent，支持协作能力
type MultiAgent interface {
	Agent

	// ID 返回 Agent 的唯一标识符
	ID() string

	// ReceiveMessage 接收消息
	ReceiveMessage(ctx context.Context, msg *AgentMessage) error

	// SendMessage 发送消息
	SendMessage(ctx context.Context, msg *AgentMessage) error

	// GetCapabilities 获取 Agent 能力描述
	GetCapabilities() []string

	// CanHandle 检查是否可以处理任务，返回是否可处理和置信度
	CanHandle(task string) (bool, float64)
}

// MessageType 消息类型
type MessageType string

const (
	// MessageTypeRequest 请求消息
	MessageTypeRequest MessageType = "request"
	// MessageTypeResponse 响应消息
	MessageTypeResponse MessageType = "response"
	// MessageTypeTask 任务分配消息
	MessageTypeTask MessageType = "task"
	// MessageTypeResult 任务结果消息
	MessageTypeResult MessageType = "result"
	// MessageTypeQuery 查询消息
	MessageTypeQuery MessageType = "query"
	// MessageTypeBroadcast 广播消息
	MessageTypeBroadcast MessageType = "broadcast"
	// MessageTypeError 错误消息
	MessageTypeError MessageType = "error"
	// MessageTypeAck 确认消息
	MessageTypeAck MessageType = "ack"
)

// AgentMessage Agent 之间的消息
type AgentMessage struct {
	// ID 消息唯一标识符
	ID string

	// From 发送者 Agent ID
	From string

	// To 接收者 Agent ID (空表示广播)
	To string

	// Type 消息类型
	Type MessageType

	// Content 消息内容
	Content string

	// Metadata 元数据
	Metadata map[string]interface{}

	// Priority 优先级 (0-10)
	Priority int

	// Timestamp 时间戳
	Timestamp time.Time

	// ParentID 父消息 ID (用于追踪对话)
	ParentID string

	// RequiresAck 是否需要确认
	RequiresAck bool
}

// MultiAgentSystem 多 Agent 系统
type MultiAgentSystem struct {
	// 系统配置
	config *MultiAgentConfig

	// Agent 注册表
	agents map[string]MultiAgent

	// 协调器 Agent
	coordinator MultiAgent

	// 消息总线
	messageBus *MessageBus

	// 共享状态存储
	sharedState *SharedState

	// 执行历史
	history *ExecutionHistory

	// 监控指标
	metrics *MultiAgentMetrics

	// 互斥锁
	mu sync.RWMutex
}

// MultiAgentConfig 系统配置
type MultiAgentConfig struct {
	// Strategy 协调策略
	Strategy CoordinationStrategy

	// MaxConcurrentAgents 最大并行 Agent 数
	MaxConcurrentAgents int

	// MessageTimeout 消息超时
	MessageTimeout time.Duration

	// TaskTimeout 任务超时
	TaskTimeout time.Duration

	// MaxRetries 最大重试次数
	MaxRetries int

	// EnableSharedState 是否启用共享状态
	EnableSharedState bool

	// EnableHistory 是否启用历史记录
	EnableHistory bool

	// MessageQueueSize 消息队列大小
	MessageQueueSize int
}

// DefaultMultiAgentConfig 返回默认配置
func DefaultMultiAgentConfig() *MultiAgentConfig {
	return &MultiAgentConfig{
		MaxConcurrentAgents: 5,
		MessageTimeout:      30 * time.Second,
		TaskTimeout:         5 * time.Minute,
		MaxRetries:          3,
		EnableSharedState:   true,
		EnableHistory:       true,
		MessageQueueSize:    1000,
	}
}

// NewMultiAgentSystem 创建 Multi-Agent 系统
func NewMultiAgentSystem(coordinator MultiAgent, config *MultiAgentConfig) *MultiAgentSystem {
	if config == nil {
		config = DefaultMultiAgentConfig()
	}

	return &MultiAgentSystem{
		config:      config,
		agents:      make(map[string]MultiAgent),
		coordinator: coordinator,
		messageBus:  NewMessageBus(config.MessageQueueSize),
		sharedState: NewSharedState(),
		history:     NewExecutionHistory(),
		metrics:     NewMultiAgentMetrics(),
	}
}

// AddAgent 添加 Agent
func (mas *MultiAgentSystem) AddAgent(id string, agent MultiAgent) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	if _, exists := mas.agents[id]; exists {
		return fmt.Errorf("agent with id %s already exists", id)
	}

	mas.agents[id] = agent

	// 注册 Agent 到消息总线
	mas.messageBus.RegisterAgent(id, agent)

	return nil
}

// RemoveAgent 移除 Agent
func (mas *MultiAgentSystem) RemoveAgent(id string) error {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	if _, exists := mas.agents[id]; !exists {
		return fmt.Errorf("agent with id %s not found", id)
	}

	delete(mas.agents, id)
	mas.messageBus.UnregisterAgent(id)

	return nil
}

// GetAgent 获取 Agent
func (mas *MultiAgentSystem) GetAgent(id string) (MultiAgent, bool) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	agent, exists := mas.agents[id]
	return agent, exists
}

// ListAgents 列出所有 Agent
func (mas *MultiAgentSystem) ListAgents() []string {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	ids := make([]string, 0, len(mas.agents))
	for id := range mas.agents {
		ids = append(ids, id)
	}
	return ids
}

// Run 执行任务
func (mas *MultiAgentSystem) Run(ctx context.Context, task string) (*MultiAgentResult, error) {
	startTime := time.Now()

	// 1. 创建根消息
	rootMsg := &AgentMessage{
		ID:        generateMessageID(),
		From:      "system",
		To:        mas.coordinator.ID(),
		Type:      MessageTypeTask,
		Content:   task,
		Timestamp: time.Now(),
		Priority:  5,
		Metadata:  make(map[string]interface{}),
	}

	// 2. 记录开始
	if mas.config.EnableHistory {
		mas.history.RecordStart(rootMsg)
	}
	mas.metrics.IncrementTotalRuns()

	// 3. 创建超时上下文
	taskCtx := ctx
	if mas.config.TaskTimeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, mas.config.TaskTimeout)
		defer cancel()
	}

	// 4. 发送给协调器
	if err := mas.messageBus.Send(taskCtx, rootMsg); err != nil {
		mas.metrics.IncrementFailedRuns()
		return nil, fmt.Errorf("failed to send task to coordinator: %w", err)
	}

	// 5. 启动消息处理循环
	resultChan := make(chan *MultiAgentResult, 1)
	errorChan := make(chan error, 1)

	go mas.processMessages(taskCtx, rootMsg.ID, resultChan, errorChan)

	// 6. 等待结果或超时
	select {
	case result := <-resultChan:
		result.Duration = time.Since(startTime)
		mas.metrics.IncrementSuccessfulRuns()
		mas.metrics.RecordExecutionTime(result.Duration)
		if mas.config.EnableHistory {
			mas.history.RecordComplete(rootMsg.ID, result)
		}
		return result, nil

	case err := <-errorChan:
		mas.metrics.IncrementFailedRuns()
		if mas.config.EnableHistory {
			mas.history.RecordError(rootMsg.ID, err)
		}
		return nil, err

	case <-taskCtx.Done():
		mas.metrics.IncrementFailedRuns()
		return nil, taskCtx.Err()
	}
}

// processMessages 处理消息循环
func (mas *MultiAgentSystem) processMessages(ctx context.Context, rootID string, resultChan chan *MultiAgentResult, errorChan chan error) {
	pendingTasks := make(map[string]bool)
	results := make(map[string]string)
	pendingTasks[rootID] = true

	for {
		select {
		case msg := <-mas.messageBus.Messages():
			// 路由消息到目标 Agent
			if err := mas.messageBus.Route(ctx, msg); err != nil {
				errorChan <- fmt.Errorf("failed to route message: %w", err)
				return
			}

			// 处理不同类型的消息
			switch msg.Type {
			case MessageTypeTask:
				pendingTasks[msg.ID] = true
				mas.metrics.IncrementTotalMessages()

			case MessageTypeResult:
				results[msg.ParentID] = msg.Content
				delete(pendingTasks, msg.ParentID)
				mas.metrics.IncrementTotalMessages()

				// 检查是否所有任务完成
				if len(pendingTasks) == 0 {
					resultChan <- &MultiAgentResult{
						RootMessageID: rootID,
						FinalResult:   msg.Content,
						AgentResults:  results,
						MessageCount:  mas.messageBus.GetMessageCount(),
					}
					return
				}

			case MessageTypeError:
				errorChan <- fmt.Errorf("agent error from %s: %s", msg.From, msg.Content)
				return
			}

		case <-ctx.Done():
			errorChan <- ctx.Err()
			return
		}
	}
}

// GetMetrics 获取监控指标
func (mas *MultiAgentSystem) GetMetrics() *MultiAgentMetrics {
	return mas.metrics
}

// GetHistory 获取执行历史
func (mas *MultiAgentSystem) GetHistory() *ExecutionHistory {
	return mas.history
}

// GetSharedState 获取共享状态
func (mas *MultiAgentSystem) GetSharedState() *SharedState {
	return mas.sharedState
}

// MultiAgentResult Multi-Agent 执行结果
type MultiAgentResult struct {
	// RootMessageID 根消息 ID
	RootMessageID string

	// FinalResult 最终结果
	FinalResult string

	// AgentResults 各 Agent 的结果
	AgentResults map[string]string

	// MessageCount 消息数量
	MessageCount int

	// Duration 执行时长
	Duration time.Duration
}

// String 返回字符串表示
func (r *MultiAgentResult) String() string {
	return fmt.Sprintf("MultiAgentResult{RootMessageID: %s, MessageCount: %d, Duration: %v}",
		r.RootMessageID, r.MessageCount, r.Duration)
}

// MessageBus 消息总线
type MessageBus struct {
	// 消息队列
	queue chan *AgentMessage

	// Agent 订阅表
	subscriptions map[string]MultiAgent

	// 消息计数
	messageCount int64

	// 互斥锁
	mu sync.RWMutex
}

// NewMessageBus 创建消息总线
func NewMessageBus(queueSize int) *MessageBus {
	return &MessageBus{
		queue:         make(chan *AgentMessage, queueSize),
		subscriptions: make(map[string]MultiAgent),
	}
}

// RegisterAgent 注册 Agent
func (mb *MessageBus) RegisterAgent(id string, agent MultiAgent) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.subscriptions[id] = agent
}

// UnregisterAgent 注销 Agent
func (mb *MessageBus) UnregisterAgent(id string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	delete(mb.subscriptions, id)
}

// Send 发送消息
func (mb *MessageBus) Send(ctx context.Context, msg *AgentMessage) error {
	atomic.AddInt64(&mb.messageCount, 1)

	select {
	case mb.queue <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Messages 获取消息通道
func (mb *MessageBus) Messages() <-chan *AgentMessage {
	return mb.queue
}

// Route 路由消息到目标 Agent
func (mb *MessageBus) Route(ctx context.Context, msg *AgentMessage) error {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	// 广播消息
	if msg.To == "" || msg.To == "*" {
		for _, agent := range mb.subscriptions {
			if err := agent.ReceiveMessage(ctx, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// 点对点消息
	agent, exists := mb.subscriptions[msg.To]
	if !exists {
		return fmt.Errorf("agent %s not found", msg.To)
	}

	return agent.ReceiveMessage(ctx, msg)
}

// GetMessageCount 获取消息计数
func (mb *MessageBus) GetMessageCount() int {
	return int(atomic.LoadInt64(&mb.messageCount))
}

// SharedState 共享状态存储
type SharedState struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewSharedState 创建共享状态存储
func NewSharedState() *SharedState {
	return &SharedState{
		data: make(map[string]interface{}),
	}
}

// Set 设置状态
func (ss *SharedState) Set(key string, value interface{}) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.data[key] = value
}

// Get 获取状态
func (ss *SharedState) Get(key string) (interface{}, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	value, exists := ss.data[key]
	return value, exists
}

// Delete 删除状态
func (ss *SharedState) Delete(key string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.data, key)
}

// GetAll 获取所有状态
func (ss *SharedState) GetAll() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	result := make(map[string]interface{}, len(ss.data))
	for k, v := range ss.data {
		result[k] = v
	}
	return result
}

// Clear 清空所有状态
func (ss *SharedState) Clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.data = make(map[string]interface{})
}

// ExecutionHistory 执行历史
type ExecutionHistory struct {
	records map[string]*ExecutionRecord
	mu      sync.RWMutex
}

// ExecutionRecord 执行记录
type ExecutionRecord struct {
	MessageID string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	Result    *MultiAgentResult
	Error     error
	Messages  []*AgentMessage
}

// NewExecutionHistory 创建执行历史
func NewExecutionHistory() *ExecutionHistory {
	return &ExecutionHistory{
		records: make(map[string]*ExecutionRecord),
	}
}

// RecordStart 记录开始
func (eh *ExecutionHistory) RecordStart(msg *AgentMessage) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.records[msg.ID] = &ExecutionRecord{
		MessageID: msg.ID,
		StartTime: time.Now(),
		Status:    "running",
		Messages:  []*AgentMessage{msg},
	}
}

// RecordComplete 记录完成
func (eh *ExecutionHistory) RecordComplete(messageID string, result *MultiAgentResult) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	if record, exists := eh.records[messageID]; exists {
		record.EndTime = time.Now()
		record.Status = "completed"
		record.Result = result
	}
}

// RecordError 记录错误
func (eh *ExecutionHistory) RecordError(messageID string, err error) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	if record, exists := eh.records[messageID]; exists {
		record.EndTime = time.Now()
		record.Status = "failed"
		record.Error = err
	}
}

// GetRecord 获取记录
func (eh *ExecutionHistory) GetRecord(messageID string) (*ExecutionRecord, bool) {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	record, exists := eh.records[messageID]
	return record, exists
}

// GetAllRecords 获取所有记录
func (eh *ExecutionHistory) GetAllRecords() []*ExecutionRecord {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	records := make([]*ExecutionRecord, 0, len(eh.records))
	for _, record := range eh.records {
		records = append(records, record)
	}
	return records
}

// MultiAgentMetrics 监控指标
type MultiAgentMetrics struct {
	TotalRuns        int64
	SuccessfulRuns   int64
	FailedRuns       int64
	TotalMessages    int64
	AverageTime      time.Duration
	AgentUtilization map[string]int64
	mu               sync.RWMutex
}

// NewMultiAgentMetrics 创建监控指标
func NewMultiAgentMetrics() *MultiAgentMetrics {
	return &MultiAgentMetrics{
		AgentUtilization: make(map[string]int64),
	}
}

// IncrementTotalRuns 增加总运行次数
func (m *MultiAgentMetrics) IncrementTotalRuns() {
	atomic.AddInt64(&m.TotalRuns, 1)
}

// IncrementSuccessfulRuns 增加成功运行次数
func (m *MultiAgentMetrics) IncrementSuccessfulRuns() {
	atomic.AddInt64(&m.SuccessfulRuns, 1)
}

// IncrementFailedRuns 增加失败运行次数
func (m *MultiAgentMetrics) IncrementFailedRuns() {
	atomic.AddInt64(&m.FailedRuns, 1)
}

// IncrementTotalMessages 增加总消息数
func (m *MultiAgentMetrics) IncrementTotalMessages() {
	atomic.AddInt64(&m.TotalMessages, 1)
}

// RecordExecutionTime 记录执行时间
func (m *MultiAgentMetrics) RecordExecutionTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalRuns := atomic.LoadInt64(&m.TotalRuns)
	if totalRuns == 0 {
		m.AverageTime = duration
	} else {
		m.AverageTime = (m.AverageTime*time.Duration(totalRuns-1) + duration) / time.Duration(totalRuns)
	}
}

// RecordAgentUtilization 记录 Agent 使用次数
func (m *MultiAgentMetrics) RecordAgentUtilization(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AgentUtilization[agentID]++
}

// GetStats 获取统计信息
func (m *MultiAgentMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalRuns := atomic.LoadInt64(&m.TotalRuns)
	successRate := float64(0)
	if totalRuns > 0 {
		successRate = float64(atomic.LoadInt64(&m.SuccessfulRuns)) / float64(totalRuns) * 100
	}

	return map[string]interface{}{
		"total_runs":        totalRuns,
		"successful_runs":   atomic.LoadInt64(&m.SuccessfulRuns),
		"failed_runs":       atomic.LoadInt64(&m.FailedRuns),
		"success_rate":      successRate,
		"total_messages":    atomic.LoadInt64(&m.TotalMessages),
		"average_time":      m.AverageTime.String(),
		"agent_utilization": m.AgentUtilization,
	}
}

// CoordinationStrategy 协调策略接口
type CoordinationStrategy interface {
	// SelectAgent 选择合适的 Agent 处理任务
	SelectAgent(ctx context.Context, task string, agents map[string]MultiAgent) (string, error)

	// DecomposeTask 分解任务
	DecomposeTask(ctx context.Context, task string) ([]SubTask, error)

	// MergeResults 合并结果
	MergeResults(ctx context.Context, results map[string]string) (string, error)
}

// SubTask 子任务
type SubTask struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	AssignedTo   string   `json:"assigned_to,omitempty"`
	Priority     int      `json:"priority"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// SequentialStrategy 顺序执行策略
type SequentialStrategy struct {
	llm chat.ChatModel
}

// NewSequentialStrategy 创建顺序执行策略
func NewSequentialStrategy(llm chat.ChatModel) *SequentialStrategy {
	return &SequentialStrategy{llm: llm}
}

// SelectAgent 选择 Agent
func (s *SequentialStrategy) SelectAgent(ctx context.Context, task string, agents map[string]MultiAgent) (string, error) {
	var bestAgent string
	var bestScore float64

	for id, agent := range agents {
		canHandle, score := agent.CanHandle(task)
		if canHandle && score > bestScore {
			bestAgent = id
			bestScore = score
		}
	}

	if bestAgent == "" {
		return "", fmt.Errorf("no suitable agent found for task: %s", task)
	}

	return bestAgent, nil
}

// DecomposeTask 分解任务
func (s *SequentialStrategy) DecomposeTask(ctx context.Context, task string) ([]SubTask, error) {
	prompt := fmt.Sprintf(`Decompose the following complex task into smaller subtasks:

Task: %s

Return the subtasks as a JSON array with the following format:
[
  {
    "id": "subtask_1",
    "description": "...",
    "priority": 1
  }
]

Only return the JSON array, no additional text.`, task)

	messages := []types.Message{types.NewUserMessage(prompt)}
	response, err := s.llm.Invoke(ctx, messages)
	if err != nil {
		return nil, err
	}

	var subtasks []SubTask
	if err := json.Unmarshal([]byte(response.Content), &subtasks); err != nil {
		return nil, fmt.Errorf("failed to parse subtasks: %w", err)
	}

	return subtasks, nil
}

// MergeResults 合并结果
func (s *SequentialStrategy) MergeResults(ctx context.Context, results map[string]string) (string, error) {
	resultsJSON, _ := json.Marshal(results)

	prompt := fmt.Sprintf(`Merge the following results from multiple agents into a coherent final answer:

Results: %s

Provide a comprehensive and well-structured final answer.`, string(resultsJSON))

	messages := []types.Message{types.NewUserMessage(prompt)}
	response, err := s.llm.Invoke(ctx, messages)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// generateMessageID 生成消息 ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

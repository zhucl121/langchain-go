// Package agents provides specialized multi-agent implementations.
package agents

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"langchain-go/core/chat"
	"langchain-go/core/tools"
)

// BaseMultiAgent 基础 Multi-Agent 实现
type BaseMultiAgent struct {
	id           string
	llm          chat.ChatModel
	capabilities []string
	messageBus   *MessageBus
	mu           sync.RWMutex
}

// ID 返回 Agent ID
func (bma *BaseMultiAgent) ID() string {
	return bma.id
}

// GetCapabilities 获取能力列表
func (bma *BaseMultiAgent) GetCapabilities() []string {
	bma.mu.RLock()
	defer bma.mu.RUnlock()
	return append([]string{}, bma.capabilities...)
}

// ReceiveMessage 接收消息（子类需要重写）
func (bma *BaseMultiAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	return fmt.Errorf("ReceiveMessage not implemented for %s", bma.id)
}

// SendMessage 发送消息
func (bma *BaseMultiAgent) SendMessage(ctx context.Context, msg *AgentMessage) error {
	if bma.messageBus == nil {
		return fmt.Errorf("message bus not set")
	}
	msg.From = bma.id
	return bma.messageBus.Send(ctx, msg)
}

// SetMessageBus 设置消息总线
func (bma *BaseMultiAgent) SetMessageBus(bus *MessageBus) {
	bma.mu.Lock()
	defer bma.mu.Unlock()
	bma.messageBus = bus
}

// CanHandle 检查是否可以处理任务（子类需要重写）
func (bma *BaseMultiAgent) CanHandle(task string) (bool, float64) {
	return false, 0.0
}

// SendResult 发送结果消息
func (bma *BaseMultiAgent) SendResult(ctx context.Context, originalMsg *AgentMessage, result string) error {
	return bma.SendMessage(ctx, &AgentMessage{
		ID:       generateMessageID(),
		From:     bma.id,
		To:       originalMsg.From,
		Type:     MessageTypeResult,
		Content:  result,
		ParentID: originalMsg.ID,
		Priority: originalMsg.Priority,
	})
}

// SendError 发送错误消息
func (bma *BaseMultiAgent) SendError(ctx context.Context, originalMsg *AgentMessage, err error) error {
	return bma.SendMessage(ctx, &AgentMessage{
		ID:       generateMessageID(),
		From:     bma.id,
		To:       originalMsg.From,
		Type:     MessageTypeError,
		Content:  err.Error(),
		ParentID: originalMsg.ID,
		Priority: originalMsg.Priority,
	})
}

// CoordinatorAgent 协调器 Agent
type CoordinatorAgent struct {
	BaseMultiAgent
	strategy   CoordinationStrategy
	agents     map[string]MultiAgent
	taskQueue  map[string][]SubTask
	taskStatus map[string]string
}

// NewCoordinatorAgent 创建协调器 Agent
func NewCoordinatorAgent(id string, llm chat.ChatModel, strategy CoordinationStrategy) *CoordinatorAgent {
	return &CoordinatorAgent{
		BaseMultiAgent: BaseMultiAgent{
			id:           id,
			llm:          llm,
			capabilities: []string{"coordination", "task_decomposition", "result_aggregation"},
		},
		strategy:   strategy,
		agents:     make(map[string]MultiAgent),
		taskQueue:  make(map[string][]SubTask),
		taskStatus: make(map[string]string),
	}
}

// RegisterAgent 注册可用的 Agent
func (ca *CoordinatorAgent) RegisterAgent(agent MultiAgent) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.agents[agent.ID()] = agent
}

// ReceiveMessage 接收消息
func (ca *CoordinatorAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	switch msg.Type {
	case MessageTypeTask:
		return ca.handleTask(ctx, msg)
	case MessageTypeResult:
		return ca.handleResult(ctx, msg)
	default:
		return nil
	}
}

// handleTask 处理任务
func (ca *CoordinatorAgent) handleTask(ctx context.Context, msg *AgentMessage) error {
	// 1. 分解任务
	subtasks, err := ca.strategy.DecomposeTask(ctx, msg.Content)
	if err != nil {
		return ca.SendError(ctx, msg, err)
	}

	// 2. 存储子任务
	ca.mu.Lock()
	ca.taskQueue[msg.ID] = subtasks
	ca.taskStatus[msg.ID] = "processing"
	ca.mu.Unlock()

	// 3. 为每个子任务选择合适的 Agent 并分配
	for _, subtask := range subtasks {
		agentID, err := ca.strategy.SelectAgent(ctx, subtask.Description, ca.agents)
		if err != nil {
			return ca.SendError(ctx, msg, err)
		}

		subtask.AssignedTo = agentID

		// 发送子任务
		taskMsg := &AgentMessage{
			ID:       generateMessageID(),
			From:     ca.id,
			To:       agentID,
			Type:     MessageTypeTask,
			Content:  subtask.Description,
			ParentID: msg.ID,
			Priority: subtask.Priority,
			Metadata: map[string]interface{}{
				"subtask_id": subtask.ID,
			},
		}

		if err := ca.SendMessage(ctx, taskMsg); err != nil {
			return err
		}
	}

	return nil
}

// handleResult 处理结果
func (ca *CoordinatorAgent) handleResult(ctx context.Context, msg *AgentMessage) error {
	// 检查该任务的所有子任务是否完成
	ca.mu.RLock()
	subtasks, exists := ca.taskQueue[msg.ParentID]
	ca.mu.RUnlock()

	if !exists {
		return nil
	}

	// 收集所有结果
	results := make(map[string]string)
	allCompleted := true
	for _, subtask := range subtasks {
		// 这里简化处理，实际需要更复杂的状态跟踪
		results[subtask.ID] = msg.Content
	}

	if allCompleted {
		// 合并结果
		finalResult, err := ca.strategy.MergeResults(ctx, results)
		if err != nil {
			return ca.SendError(ctx, msg, err)
		}

		// 发送最终结果
		return ca.SendResult(ctx, msg, finalResult)
	}

	return nil
}

// CanHandle 协调器可以处理所有任务
func (ca *CoordinatorAgent) CanHandle(task string) (bool, float64) {
	return true, 1.0
}

// ResearcherAgent 研究员 Agent
type ResearcherAgent struct {
	BaseMultiAgent
	searchTool tools.Tool
}

// NewResearcherAgent 创建研究员 Agent
func NewResearcherAgent(id string, llm chat.ChatModel, searchTool tools.Tool) *ResearcherAgent {
	return &ResearcherAgent{
		BaseMultiAgent: BaseMultiAgent{
			id:           id,
			llm:          llm,
			capabilities: []string{"research", "search", "information_gathering", "fact_checking"},
		},
		searchTool: searchTool,
	}
}

// ReceiveMessage 接收消息
func (ra *ResearcherAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	if msg.Type != MessageTypeTask {
		return nil
	}

	// 执行研究任务
	result, err := ra.research(ctx, msg.Content)
	if err != nil {
		return ra.SendError(ctx, msg, err)
	}

	return ra.SendResult(ctx, msg, result)
}

// research 执行研究
func (ra *ResearcherAgent) research(ctx context.Context, query string) (string, error) {
	// 1. 使用搜索工具收集信息
	searchResult := ""
	if ra.searchTool != nil {
		result, err := ra.searchTool.Run(ctx, query)
		if err == nil {
			searchResult = result
		}
	}

	// 2. 使用 LLM 分析和组织信息
	prompt := fmt.Sprintf(`You are a research assistant. Based on the following search results, provide a comprehensive research summary.

Query: %s

Search Results:
%s

Provide a well-structured research summary with key findings.`, query, searchResult)

	messages := []chat.Message{chat.NewHumanMessage(prompt)}
	response, err := ra.llm.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// CanHandle 检查是否可以处理任务
func (ra *ResearcherAgent) CanHandle(task string) (bool, float64) {
	keywords := []string{"research", "search", "find", "investigate", "explore", "study", "analyze"}
	taskLower := strings.ToLower(task)

	for _, keyword := range keywords {
		if strings.Contains(taskLower, keyword) {
			return true, 0.9
		}
	}

	return false, 0.0
}

// WriterAgent 写作 Agent
type WriterAgent struct {
	BaseMultiAgent
	style string
}

// NewWriterAgent 创建写作 Agent
func NewWriterAgent(id string, llm chat.ChatModel, style string) *WriterAgent {
	return &WriterAgent{
		BaseMultiAgent: BaseMultiAgent{
			id:           id,
			llm:          llm,
			capabilities: []string{"writing", "editing", "summarization", "content_creation"},
		},
		style: style,
	}
}

// ReceiveMessage 接收消息
func (wa *WriterAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	if msg.Type != MessageTypeTask {
		return nil
	}

	// 执行写作任务
	result, err := wa.write(ctx, msg.Content)
	if err != nil {
		return wa.SendError(ctx, msg, err)
	}

	return wa.SendResult(ctx, msg, result)
}

// write 执行写作
func (wa *WriterAgent) write(ctx context.Context, task string) (string, error) {
	prompt := fmt.Sprintf(`You are a professional writer with a %s writing style.

Task: %s

Write a well-structured, engaging, and high-quality content that fulfills the task requirements.`, wa.style, task)

	messages := []chat.Message{chat.NewHumanMessage(prompt)}
	response, err := wa.llm.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// CanHandle 检查是否可以处理任务
func (wa *WriterAgent) CanHandle(task string) (bool, float64) {
	keywords := []string{"write", "compose", "create", "draft", "author", "summarize", "edit"}
	taskLower := strings.ToLower(task)

	for _, keyword := range keywords {
		if strings.Contains(taskLower, keyword) {
			return true, 0.85
		}
	}

	return false, 0.0
}

// ReviewerAgent 审核 Agent
type ReviewerAgent struct {
	BaseMultiAgent
	criteria []string
}

// NewReviewerAgent 创建审核 Agent
func NewReviewerAgent(id string, llm chat.ChatModel, criteria []string) *ReviewerAgent {
	if len(criteria) == 0 {
		criteria = []string{"accuracy", "clarity", "completeness", "grammar"}
	}

	return &ReviewerAgent{
		BaseMultiAgent: BaseMultiAgent{
			id:           id,
			llm:          llm,
			capabilities: []string{"review", "critique", "evaluation", "quality_assurance"},
		},
		criteria: criteria,
	}
}

// ReceiveMessage 接收消息
func (ra *ReviewerAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	if msg.Type != MessageTypeTask {
		return nil
	}

	// 执行审核任务
	result, err := ra.review(ctx, msg.Content)
	if err != nil {
		return ra.SendError(ctx, msg, err)
	}

	return ra.SendResult(ctx, msg, result)
}

// review 执行审核
func (ra *ReviewerAgent) review(ctx context.Context, content string) (string, error) {
	criteriaStr := strings.Join(ra.criteria, ", ")

	prompt := fmt.Sprintf(`You are a professional reviewer. Review the following content based on these criteria: %s.

Content to Review:
%s

Provide a detailed review with:
1. Overall assessment
2. Strengths
3. Areas for improvement
4. Specific suggestions
5. Final rating (1-10)`, criteriaStr, content)

	messages := []chat.Message{chat.NewHumanMessage(prompt)}
	response, err := ra.llm.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// CanHandle 检查是否可以处理任务
func (ra *ReviewerAgent) CanHandle(task string) (bool, float64) {
	keywords := []string{"review", "evaluate", "assess", "critique", "check", "verify"}
	taskLower := strings.ToLower(task)

	for _, keyword := range keywords {
		if strings.Contains(taskLower, keyword) {
			return true, 0.88
		}
	}

	return false, 0.0
}

// AnalystAgent 分析 Agent
type AnalystAgent struct {
	BaseMultiAgent
}

// NewAnalystAgent 创建分析 Agent
func NewAnalystAgent(id string, llm chat.ChatModel) *AnalystAgent {
	return &AnalystAgent{
		BaseMultiAgent: BaseMultiAgent{
			id:           id,
			llm:          llm,
			capabilities: []string{"analysis", "data_processing", "insights", "pattern_recognition"},
		},
	}
}

// ReceiveMessage 接收消息
func (aa *AnalystAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	if msg.Type != MessageTypeTask {
		return nil
	}

	// 执行分析任务
	result, err := aa.analyze(ctx, msg.Content)
	if err != nil {
		return aa.SendError(ctx, msg, err)
	}

	return aa.SendResult(ctx, msg, result)
}

// analyze 执行分析
func (aa *AnalystAgent) analyze(ctx context.Context, data string) (string, error) {
	prompt := fmt.Sprintf(`You are a data analyst. Analyze the following data and provide insights.

Data/Task: %s

Provide:
1. Key findings
2. Patterns and trends
3. Insights and recommendations
4. Statistical summary (if applicable)`, data)

	messages := []chat.Message{chat.NewHumanMessage(prompt)}
	response, err := aa.llm.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// CanHandle 检查是否可以处理任务
func (aa *AnalystAgent) CanHandle(task string) (bool, float64) {
	keywords := []string{"analyze", "analysis", "data", "insights", "trends", "patterns", "statistics"}
	taskLower := strings.ToLower(task)

	for _, keyword := range keywords {
		if strings.Contains(taskLower, keyword) {
			return true, 0.87
		}
	}

	return false, 0.0
}

// PlannerAgent 规划 Agent
type PlannerAgent struct {
	BaseMultiAgent
}

// NewPlannerAgent 创建规划 Agent
func NewPlannerAgent(id string, llm chat.ChatModel) *PlannerAgent {
	return &PlannerAgent{
		BaseMultiAgent: BaseMultiAgent{
			id:           id,
			llm:          llm,
			capabilities: []string{"planning", "strategy", "task_decomposition", "scheduling"},
		},
	}
}

// ReceiveMessage 接收消息
func (pa *PlannerAgent) ReceiveMessage(ctx context.Context, msg *AgentMessage) error {
	if msg.Type != MessageTypeTask {
		return nil
	}

	// 执行规划任务
	result, err := pa.plan(ctx, msg.Content)
	if err != nil {
		return pa.SendError(ctx, msg, err)
	}

	return pa.SendResult(ctx, msg, result)
}

// plan 执行规划
func (pa *PlannerAgent) plan(ctx context.Context, goal string) (string, error) {
	prompt := fmt.Sprintf(`You are a strategic planner. Create a detailed plan for the following goal.

Goal: %s

Provide a comprehensive plan with:
1. Objectives
2. Steps (in order)
3. Timeline estimates
4. Resources needed
5. Risk considerations
6. Success criteria`, goal)

	messages := []chat.Message{chat.NewHumanMessage(prompt)}
	response, err := pa.llm.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// CanHandle 检查是否可以处理任务
func (pa *PlannerAgent) CanHandle(task string) (bool, float64) {
	keywords := []string{"plan", "strategy", "organize", "schedule", "coordinate", "prepare"}
	taskLower := strings.ToLower(task)

	for _, keyword := range keywords {
		if strings.Contains(taskLower, keyword) {
			return true, 0.86
		}
	}

	return false, 0.0
}

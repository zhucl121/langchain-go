package a2a

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

// CollaborationCoordinator coordinates multiple agents to complete complex tasks.
type CollaborationCoordinator struct {
	registry AgentRegistry
	router   TaskRouter
	
	// Collaboration sessions
	sessions   map[string]*CollaborationSession
	sessionsMu sync.RWMutex
}

// CollaborationSession represents an active collaboration session.
type CollaborationSession struct {
	ID           string
	MainTask     *Task
	Participants map[string]A2AAgent
	SubTasks     map[string]*Task
	Results      map[string]*TaskResult
	Status       SessionStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SessionStatus represents session status.
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// NewCollaborationCoordinator creates a new collaboration coordinator.
func NewCollaborationCoordinator(registry AgentRegistry, router TaskRouter) *CollaborationCoordinator {
	return &CollaborationCoordinator{
		registry: registry,
		router:   router,
		sessions: make(map[string]*CollaborationSession),
	}
}

// Coordinate coordinates multiple agents to complete a complex task.
func (c *CollaborationCoordinator) Coordinate(ctx context.Context, task *Task) (*TaskResult, error) {
	// Create collaboration session
	session := c.createSession(task)
	
	// Decompose task into subtasks
	subTasks, err := c.decomposeTask(task)
	if err != nil {
		return nil, fmt.Errorf("decompose task: %w", err)
	}
	
	session.SubTasks = make(map[string]*Task)
	for _, subTask := range subTasks {
		session.SubTasks[subTask.ID] = subTask
	}
	
	// Route each subtask to an agent
	for _, subTask := range subTasks {
		agent, err := c.router.Route(ctx, subTask)
		if err != nil {
			return nil, fmt.Errorf("route subtask %s: %w", subTask.ID, err)
		}
		
		info, _ := agent.GetInfo(ctx)
		session.Participants[info.ID] = agent
	}
	
	// Execute subtasks in parallel
	results, err := c.executeParallel(ctx, session)
	if err != nil {
		session.Status = SessionStatusFailed
		return nil, fmt.Errorf("execute parallel: %w", err)
	}
	
	// Aggregate results
	finalResult, err := c.aggregateResults(ctx, results)
	if err != nil {
		session.Status = SessionStatusFailed
		return nil, fmt.Errorf("aggregate results: %w", err)
	}
	
	session.Status = SessionStatusCompleted
	session.UpdatedAt = time.Now()
	
	return finalResult, nil
}

// createSession creates a new collaboration session.
func (c *CollaborationCoordinator) createSession(task *Task) *CollaborationSession {
	c.sessionsMu.Lock()
	defer c.sessionsMu.Unlock()
	
	session := &CollaborationSession{
		ID:           uuid.New().String(),
		MainTask:     task,
		Participants: make(map[string]A2AAgent),
		SubTasks:     make(map[string]*Task),
		Results:      make(map[string]*TaskResult),
		Status:       SessionStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	c.sessions[session.ID] = session
	
	return session
}

// decomposeTask decomposes a complex task into subtasks.
func (c *CollaborationCoordinator) decomposeTask(task *Task) ([]*Task, error) {
	// Simple decomposition strategy (can be enhanced with LLM)
	if task.Type != TaskTypeComplex {
		// Not a complex task, no decomposition needed
		return []*Task{task}, nil
	}
	
	// For demo, create 3 subtasks based on common workflow
	subtasks := []*Task{
		{
			ID:       uuid.New().String(),
			Type:     TaskTypeQuery,
			Priority: task.Priority,
			Input: &TaskInput{
				Type:    "text",
				Content: fmt.Sprintf("Research phase: %s", task.Input.Content),
			},
			Metadata: map[string]any{
				"phase":   "research",
				"mainTask": task.ID,
			},
		},
		{
			ID:       uuid.New().String(),
			Type:     TaskTypeAnalyze,
			Priority: task.Priority,
			Input: &TaskInput{
				Type:    "text",
				Content: fmt.Sprintf("Analysis phase: %s", task.Input.Content),
			},
			Metadata: map[string]any{
				"phase":    "analysis",
				"mainTask": task.ID,
			},
		},
		{
			ID:       uuid.New().String(),
			Type:     TaskTypeGenerate,
			Priority: task.Priority,
			Input: &TaskInput{
				Type:    "text",
				Content: fmt.Sprintf("Generation phase: %s", task.Input.Content),
			},
			Metadata: map[string]any{
				"phase":    "generation",
				"mainTask": task.ID,
			},
		},
	}
	
	return subtasks, nil
}

// executeParallel executes subtasks in parallel.
func (c *CollaborationCoordinator) executeParallel(ctx context.Context, session *CollaborationSession) (map[string]*TaskResult, error) {
	results := make(map[string]*TaskResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	
	for taskID, subTask := range session.SubTasks {
		wg.Add(1)
		go func(tid string, task *Task) {
			defer wg.Done()
			
			// Find agent for this task
			var agent A2AAgent
			for _, a := range session.Participants {
				// Simple: use first available agent (can be improved)
				agent = a
				break
			}
			
			if agent == nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("no agent available for task %s", tid)
				}
				mu.Unlock()
				return
			}
			
			// Execute task
			response, err := agent.SendTask(ctx, task)
			
			mu.Lock()
			defer mu.Unlock()
			
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			
			if response.Status == TaskStatusCompleted {
				results[tid] = response.Result
				session.Results[tid] = response.Result
			} else {
				if firstErr == nil {
					firstErr = fmt.Errorf("task %s failed: %s", tid, response.Error.Message)
				}
			}
		}(taskID, subTask)
	}
	
	wg.Wait()
	
	if firstErr != nil {
		return nil, firstErr
	}
	
	return results, nil
}

// aggregateResults aggregates results from multiple subtasks.
func (c *CollaborationCoordinator) aggregateResults(ctx context.Context, results map[string]*TaskResult) (*TaskResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to aggregate")
	}
	
	// Simple aggregation: combine all text content
	var combined string
	for taskID, result := range results {
		combined += fmt.Sprintf("[Subtask %s]: %s\n\n", taskID, result.Content)
	}
	
	return &TaskResult{
		Type:       "text",
		Content:    combined,
		Confidence: 0.9,
	}, nil
}

// GetSession returns a collaboration session.
func (c *CollaborationCoordinator) GetSession(sessionID string) *CollaborationSession {
	c.sessionsMu.RLock()
	defer c.sessionsMu.RUnlock()
	
	return c.sessions[sessionID]
}

// ListSessions lists all collaboration sessions.
func (c *CollaborationCoordinator) ListSessions() []*CollaborationSession {
	c.sessionsMu.RLock()
	defer c.sessionsMu.RUnlock()
	
	sessions := make([]*CollaborationSession, 0, len(c.sessions))
	for _, session := range c.sessions {
		sessions = append(sessions, session)
	}
	
	return sessions
}

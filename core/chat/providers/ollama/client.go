package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zhucl121/langchain-go/core/chat"
	"github.com/zhucl121/langchain-go/core/runnable"
	"github.com/zhucl121/langchain-go/pkg/types"
)

const (
	// DefaultBaseURL is the default Ollama API base URL
	DefaultBaseURL = "http://localhost:11434"

	// DefaultTimeout is the default request timeout
	DefaultTimeout = 120 * time.Second

	// DefaultTemperature is the default temperature
	DefaultTemperature = 0.7

	// DefaultTopK is the default top-K sampling parameter
	DefaultTopK = 40

	// DefaultTopP is the default top-P sampling parameter
	DefaultTopP = 0.9

	// DefaultRepeatPenalty is the default repeat penalty
	DefaultRepeatPenalty = 1.1
)

// Config is the configuration for Ollama ChatModel.
type Config struct {
	// BaseURL is the Ollama API base URL (optional, default: http://localhost:11434)
	BaseURL string

	// Model is the model name (required, e.g., "llama2", "mistral", "codellama")
	Model string

	// Temperature controls randomness (optional, 0.0-2.0, default: 0.7)
	// Lower values make output more deterministic, higher values more random
	Temperature float64

	// NumPredict is the maximum number of tokens to generate (optional)
	// If 0, there is no limit
	NumPredict int

	// TopK is the top-K sampling parameter (optional, default: 40)
	TopK int

	// TopP is the top-P (nucleus) sampling parameter (optional, default: 0.9)
	TopP float64

	// RepeatPenalty is the penalty for repeating tokens (optional, default: 1.1)
	RepeatPenalty float64

	// Seed is the random seed for reproducibility (optional)
	Seed *int

	// Timeout is the request timeout (optional, default: 120s)
	Timeout time.Duration

	// Format specifies the response format (optional, "json" for JSON mode)
	Format string
}

// Validate validates the configuration.
func (c Config) Validate() error {
	if c.Model == "" {
		return fmt.Errorf("Model is required")
	}

	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("Temperature must be between 0 and 2")
	}

	if c.TopP < 0 || c.TopP > 1 {
		return fmt.Errorf("TopP must be between 0 and 1")
	}

	return nil
}

// ChatModel is the Ollama ChatModel implementation.
type ChatModel struct {
	*chat.BaseChatModel
	config Config
	client *http.Client
}

// New creates a new Ollama ChatModel.
//
// Parameters:
//   - config: Ollama configuration
//
// Returns:
//   - *ChatModel: ChatModel instance
//   - error: Configuration error
func New(config Config) (*ChatModel, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set defaults
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Temperature == 0 {
		config.Temperature = DefaultTemperature
	}
	if config.TopK == 0 {
		config.TopK = DefaultTopK
	}
	if config.TopP == 0 {
		config.TopP = DefaultTopP
	}
	if config.RepeatPenalty == 0 {
		config.RepeatPenalty = DefaultRepeatPenalty
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: config.Timeout,
	}

	model := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(config.Model, "ollama"),
		config:        config,
		client:        client,
	}

	return model, nil
}

// Invoke implements the Runnable interface, performs a single invocation.
func (m *ChatModel) Invoke(ctx context.Context, messages []types.Message, opts ...runnable.Option) (types.Message, error) {
	// Validate messages
	if err := chat.ValidateMessages(messages); err != nil {
		return types.Message{}, err
	}

	// Build request
	reqBody, err := m.buildRequest(messages, false)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to build request: %w", err)
	}

	// Send request
	respBody, err := m.doRequest(ctx, reqBody)
	if err != nil {
		return types.Message{}, err
	}

	// Parse response
	var response ollamaResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return types.Message{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build message
	message := types.Message{
		Role:    types.RoleAssistant,
		Content: response.Message.Content,
	}

	return message, nil
}

// Stream implements the Runnable interface, performs streaming invocation.
func (m *ChatModel) Stream(ctx context.Context, messages []types.Message, opts ...runnable.Option) (<-chan runnable.StreamEvent[types.Message], error) {
	// Validate messages
	if err := chat.ValidateMessages(messages); err != nil {
		return nil, err
	}

	// Build request
	reqBody, err := m.buildRequest(messages, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Create output channel
	out := make(chan runnable.StreamEvent[types.Message], 10)

	// Start goroutine to handle streaming response
	go func() {
		defer close(out)

		// Send start event
		out <- runnable.StreamEvent[types.Message]{
			Type: runnable.EventStart,
			Name: m.GetName(),
		}

		// Send request
		resp, err := m.doStreamRequest(ctx, reqBody)
		if err != nil {
			out <- runnable.StreamEvent[types.Message]{
				Type:  runnable.EventError,
				Error: err,
			}
			return
		}
		defer resp.Body.Close()

		// Read streaming response
		if err := m.processStream(resp.Body, out); err != nil {
			out <- runnable.StreamEvent[types.Message]{
				Type:  runnable.EventError,
				Error: err,
			}
			return
		}

		// Send end event
		out <- runnable.StreamEvent[types.Message]{
			Type: runnable.EventEnd,
		}
	}()

	return out, nil
}

// buildRequest builds the Ollama API request.
func (m *ChatModel) buildRequest(messages []types.Message, stream bool) ([]byte, error) {
	// Convert messages to Ollama format
	ollamaMessages := make([]ollamaMessage, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Build request
	req := ollamaRequest{
		Model:    m.config.Model,
		Messages: ollamaMessages,
		Stream:   stream,
		Options: ollamaOptions{
			Temperature:   m.config.Temperature,
			NumPredict:    m.config.NumPredict,
			TopK:          m.config.TopK,
			TopP:          m.config.TopP,
			RepeatPenalty: m.config.RepeatPenalty,
			Seed:          m.config.Seed,
		},
	}

	if m.config.Format != "" {
		req.Format = m.config.Format
	}

	return json.Marshal(req)
}

// doRequest sends a non-streaming request.
func (m *ChatModel) doRequest(ctx context.Context, reqBody []byte) ([]byte, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", m.config.BaseURL+"/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// doStreamRequest sends a streaming request.
func (m *ChatModel) doStreamRequest(ctx context.Context, reqBody []byte) (*http.Response, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", m.config.BaseURL+"/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// processStream processes the streaming response.
func (m *ChatModel) processStream(body io.Reader, out chan<- runnable.StreamEvent[types.Message]) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line size

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse chunk
		var chunk ollamaStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return fmt.Errorf("failed to parse chunk: %w", err)
		}

		// Send stream event
		if chunk.Message.Content != "" {
			out <- runnable.StreamEvent[types.Message]{
				Type: runnable.EventStream,
				Data: types.Message{
					Role:    types.RoleAssistant,
					Content: chunk.Message.Content,
				},
			}
		}

		// Check if done
		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// GetType returns the model type.
func (m *ChatModel) GetType() string {
	return "ollama"
}

// Batch implements Runnable interface for batch execution.
func (m *ChatModel) Batch(ctx context.Context, inputs [][]types.Message, opts ...runnable.Option) ([]types.Message, error) {
	if len(inputs) == 0 {
		return []types.Message{}, nil
	}

	results := make([]types.Message, len(inputs))
	errs := make([]error, len(inputs))

	// Use channel for parallel execution
	type result struct {
		index int
		msg   types.Message
		err   error
	}

	resultChan := make(chan result, len(inputs))

	// Execute all requests in parallel
	for i, messages := range inputs {
		go func(idx int, msgs []types.Message) {
			msg, err := m.Invoke(ctx, msgs, opts...)
			resultChan <- result{index: idx, msg: msg, err: err}
		}(i, messages)
	}

	// Collect results
	for i := 0; i < len(inputs); i++ {
		res := <-resultChan
		results[res.index] = res.msg
		errs[res.index] = res.err
	}

	// Check for errors
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("batch request %d failed: %w", i, err)
		}
	}

	return results, nil
}

// BindTools binds tools to the model (returns a new instance).
func (m *ChatModel) BindTools(tools []types.Tool) chat.ChatModel {
	// Create new instance
	newModel := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "ollama"),
		config:        m.config,
		client:        m.client,
	}

	// Set tools
	newModel.SetBoundTools(tools)

	return newModel
}

// WithStructuredOutput configures the model to return structured output.
func (m *ChatModel) WithStructuredOutput(schema types.Schema) chat.ChatModel {
	// Create new instance
	newModel := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "ollama"),
		config:        m.config,
		client:        m.client,
	}

	// Set format to JSON for structured output
	newConfig := m.config
	newConfig.Format = "json"
	newModel.config = newConfig

	return newModel
}

// WithConfig implements Runnable interface.
func (m *ChatModel) WithConfig(config *types.Config) runnable.Runnable[[]types.Message, types.Message] {
	newModel := &ChatModel{
		BaseChatModel: chat.NewBaseChatModel(m.config.Model, "ollama"),
		config:        m.config,
		client:        m.client,
	}
	newModel.SetConfig(config)
	newModel.SetBoundTools(m.GetBoundTools())
	if schema := m.GetOutputSchema(); schema != nil {
		newModel.SetOutputSchema(*schema)
	}
	return newModel
}

// WithFallbacks implements Runnable interface.
func (m *ChatModel) WithFallbacks(fallbacks ...runnable.Runnable[[]types.Message, types.Message]) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewFallbackRunnable[[]types.Message, types.Message](m, fallbacks)
}

// WithRetry implements Runnable interface.
func (m *ChatModel) WithRetry(policy types.RetryPolicy) runnable.Runnable[[]types.Message, types.Message] {
	return runnable.NewRetryRunnable[[]types.Message, types.Message](m, policy)
}

// Request/Response types

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format,omitempty"`
	Options  ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature   float64 `json:"temperature,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"`
	Seed          *int    `json:"seed,omitempty"`
}

type ollamaResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

type ollamaStreamChunk struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

// Package ollama provides a ChatModel implementation for Ollama.
//
// Ollama is a tool for running large language models locally.
// This package allows you to use Ollama models with LangChain-Go.
//
// # Installation
//
// First, install and run Ollama from https://ollama.ai
//
// Then pull a model:
//
//	ollama pull llama2
//
// # Usage
//
// Basic usage:
//
//	import (
//	    "context"
//	    "fmt"
//
//	    "github.com/zhucl121/langchain-go/core/chat/providers/ollama"
//	    "github.com/zhucl121/langchain-go/pkg/types"
//	)
//
//	func main() {
//	    // Create Ollama ChatModel
//	    model, err := ollama.New(ollama.Config{
//	        BaseURL:     "http://localhost:11434",
//	        Model:       "llama2",
//	        Temperature: 0.7,
//	    })
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    // Use the model
//	    ctx := context.Background()
//	    messages := []types.Message{
//	        {Role: types.RoleUser, Content: "Hello, how are you?"},
//	    }
//
//	    response, err := model.Invoke(ctx, messages)
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    fmt.Println(response.Content)
//	}
//
// # Streaming
//
// Ollama supports streaming responses:
//
//	stream, err := model.Stream(ctx, messages)
//	if err != nil {
//	    panic(err)
//	}
//
//	for event := range stream {
//	    if event.Type == runnable.EventStream {
//	        fmt.Print(event.Data.Content)
//	    } else if event.Type == runnable.EventError {
//	        fmt.Println("Error:", event.Error)
//	    }
//	}
//
// # Configuration
//
// Available configuration options:
//
//   - BaseURL: Ollama API base URL (default: http://localhost:11434)
//   - Model: Model name (required, e.g., "llama2", "mistral", "codellama")
//   - Temperature: Controls randomness (0.0-2.0, default: 0.7)
//   - NumPredict: Maximum tokens to generate (default: unlimited)
//   - TopK: Top-K sampling parameter (default: 40)
//   - TopP: Top-P (nucleus) sampling parameter (default: 0.9)
//   - RepeatPenalty: Penalty for repeating tokens (default: 1.1)
//   - Seed: Random seed for reproducibility (optional)
//   - Timeout: Request timeout (default: 120s)
//
// # Supported Models
//
// Ollama supports many models. Some popular ones:
//
//   - llama2: Meta's Llama 2 (7B, 13B, 70B)
//   - mistral: Mistral AI's model (7B)
//   - codellama: Code-specialized Llama (7B, 13B, 34B)
//   - phi: Microsoft's Phi model
//   - vicuna: Vicuna model
//   - orca-mini: Smaller, faster model
//
// Check https://ollama.ai/library for the full list.
//
// # Performance Tips
//
//  1. Use smaller models for faster responses (e.g., orca-mini)
//  2. Adjust NumPredict to limit response length
//  3. Use streaming for better user experience
//  4. Run Ollama with GPU support for better performance
//
// # Error Handling
//
// Common errors:
//
//   - Connection refused: Make sure Ollama is running
//   - Model not found: Pull the model with `ollama pull <model>`
//   - Timeout: Increase Timeout or use a smaller model
//
// # Thread Safety
//
// ChatModel instances are safe for concurrent use.
package ollama

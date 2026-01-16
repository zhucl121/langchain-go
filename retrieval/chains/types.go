// Package chains 提供高层的链式调用抽象，简化 RAG 和其他复杂工作流的实现。
//
// Chains 是预构建的组件组合，可以开箱即用。
//
// 支持的 Chain 类型：
//   - RAGChain: 检索增强生成
//   - ConversationalRAGChain: 对话式 RAG
//
// 使用示例：
//
//	// 3 行代码完成 RAG
//	retriever := retrievers.NewVectorStoreRetriever(vectorStore)
//	ragChain := chains.NewRAGChain(retriever, llm)
//	result, _ := ragChain.Run(ctx, "What is LangChain?")
//
package chains

import (
	"context"
	"time"

	"langchain-go/core/chat"
	"langchain-go/core/prompts"
	"langchain-go/retrieval/loaders"
)

// Retriever 检索器接口
// 注意：这是一个简化的接口定义，实际应该在 retrieval/retrievers 包中
type Retriever interface {
	// GetRelevantDocuments 获取相关文档
	GetRelevantDocuments(ctx context.Context, query string) ([]*loaders.Document, error)
}

// RAGChain RAG 链
//
// RAG (Retrieval-Augmented Generation) 结合了检索和生成两个步骤：
// 1. 从向量存储中检索相关文档
// 2. 基于检索到的文档生成答案
//
type RAGChain struct {
	retriever Retriever
	llm       chat.ChatModel
	prompt    *prompts.PromptTemplate
	config    RAGConfig
}

// RAGConfig RAG 配置
type RAGConfig struct {
	// ReturnSources 是否返回来源文档
	ReturnSources bool

	// ScoreThreshold 相似度阈值 (0-1)
	// 低于此阈值的文档将被过滤
	ScoreThreshold float32

	// MaxContextLen 最大上下文长度（字符数）
	// 超过此长度的上下文将被截断
	MaxContextLen int

	// TopK 返回的文档数量
	TopK int

	// ContextFormatter 上下文格式化器
	// 将文档列表格式化为上下文字符串
	ContextFormatter ContextFormatter
}

// RAGResult RAG 执行结果
type RAGResult struct {
	// Question 原始问题
	Question string

	// Answer 生成的答案
	Answer string

	// Context 检索到的上下文文档
	Context []*loaders.Document

	// Confidence 置信度 (0-1)
	// 基于检索分数计算
	Confidence float64

	// TimeElapsed 执行耗时
	TimeElapsed time.Duration

	// Metadata 额外的元数据
	Metadata map[string]interface{}
}

// RAGChunk 流式输出块
type RAGChunk struct {
	// Type 块类型: "retrieval", "llm_token", "done", "error"
	Type string

	// Data 数据内容
	Data interface{}

	// Timestamp 时间戳
	Timestamp time.Time
}

// Option RAG Chain 配置选项
type Option func(*RAGChain)

// ContextFormatter 上下文格式化器函数类型
//
// 将文档列表格式化为上下文字符串，供 LLM 使用。
//
type ContextFormatter func([]*loaders.Document) string

// ConversationalRAGChain 对话式 RAG 链
//
// 支持多轮对话的 RAG，会考虑对话历史。
//
type ConversationalRAGChain struct {
	*RAGChain
	memory ConversationMemory
}

// ConversationMemory 对话记忆接口
type ConversationMemory interface {
	// AddMessage 添加消息到历史
	AddMessage(role, content string)

	// GetHistory 获取对话历史
	GetHistory() string

	// Clear 清空历史
	Clear()
}

// SimpleMemory 简单的对话记忆实现
type SimpleMemory struct {
	messages []Message
	maxLen   int
}

// Message 消息
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// NewSimpleMemory 创建简单记忆
func NewSimpleMemory(maxLen int) *SimpleMemory {
	return &SimpleMemory{
		messages: make([]Message, 0),
		maxLen:   maxLen,
	}
}

// AddMessage 实现 ConversationMemory 接口
func (m *SimpleMemory) AddMessage(role, content string) {
	m.messages = append(m.messages, Message{
		Role:    role,
		Content: content,
	})

	// 保持最大长度
	if len(m.messages) > m.maxLen {
		m.messages = m.messages[len(m.messages)-m.maxLen:]
	}
}

// GetHistory 实现 ConversationMemory 接口
func (m *SimpleMemory) GetHistory() string {
	var result string
	for _, msg := range m.messages {
		result += msg.Role + ": " + msg.Content + "\n"
	}
	return result
}

// Clear 实现 ConversationMemory 接口
func (m *SimpleMemory) Clear() {
	m.messages = make([]Message, 0)
}

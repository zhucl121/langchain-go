// Package types 提供 LangChain-Go 的基础类型定义。
//
// 主要类型：
//   - Message: 消息类型，用于 LLM 对话
//   - Tool: 工具定义，用于 Agent 工具调用
//   - Schema: JSON Schema，用于工具参数验证
//   - Config: 配置类型，用于运行时配置
//
// 使用示例：
//
//	msg := types.NewUserMessage("Hello, AI!")
//	systemMsg := types.NewSystemMessage("You are a helpful assistant.")
//
package types

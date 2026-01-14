// Package prompts 提供 LangChain 提示词模板系统的实现。
//
// prompts 包实现了灵活的提示词模板引擎，支持变量替换、条件渲染、
// Few-shot 示例等功能。所有 Prompt 都实现了 runnable.Runnable 接口，
// 可以与其他组件无缝组合。
//
// 核心特性：
//   - 字符串模板（变量替换）
//   - 聊天提示词模板（多角色消息）
//   - Few-shot 示例支持
//   - 部分变量（Partial Variables）
//   - 输出格式化
//   - 与 Runnable 系统集成
//
// 基本用法：
//
//	import (
//	    "context"
//	    "langchain-go/core/prompts"
//	)
//
//	// 创建简单模板
//	template, err := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
//	    Template:      "Tell me a {adjective} joke about {content}.",
//	    InputVariables: []string{"adjective", "content"},
//	})
//
//	// 格式化
//	result, err := template.Format(map[string]any{
//	    "adjective": "funny",
//	    "content":   "chickens",
//	})
//	fmt.Println(result) // "Tell me a funny joke about chickens."
//
// 聊天模板示例：
//
//	// 创建聊天模板
//	chatTemplate := prompts.NewChatPromptTemplate(
//	    prompts.SystemMessagePromptTemplate("You are a helpful assistant."),
//	    prompts.HumanMessagePromptTemplate("Hello, my name is {name}!"),
//	    prompts.AIMessagePromptTemplate("Nice to meet you, {name}!"),
//	    prompts.HumanMessagePromptTemplate("Can you help me with {topic}?"),
//	)
//
//	// 格式化为消息列表
//	messages, err := chatTemplate.FormatMessages(map[string]any{
//	    "name":  "Alice",
//	    "topic": "Python programming",
//	})
//
// Few-shot 示例：
//
//	examples := []map[string]string{
//	    {"input": "happy", "output": "sad"},
//	    {"input": "tall", "output": "short"},
//	}
//
//	fewShotPrompt := prompts.NewFewShotPromptTemplate(prompts.FewShotConfig{
//	    Examples:       examples,
//	    ExamplePrompt:  exampleTemplate,
//	    Prefix:         "Give the antonym of every input\n\n",
//	    Suffix:         "\nInput: {input}\nOutput:",
//	    InputVariables: []string{"input"},
//	})
//
// 部分变量：
//
//	// 预填充部分变量
//	partialTemplate := template.Partial(map[string]any{
//	    "adjective": "funny",
//	})
//
//	// 只需提供剩余变量
//	result, _ := partialTemplate.Format(map[string]any{
//	    "content": "chickens",
//	})
//
// 与 Runnable 集成：
//
//	// Prompt 可以与其他 Runnable 组合
//	chain := template.Pipe(model).Pipe(outputParser)
//	result, _ := chain.Invoke(ctx, map[string]any{...})
//
package prompts

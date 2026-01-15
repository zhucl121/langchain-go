# LangChain-Go 快速开始 - Prompts 系统

本指南帮助你快速上手 LangChain-Go 的 Prompts 系统。

## 安装

```bash
cd /Users/zhuchenglong/Documents/workspace/随笔/langchain-go
go mod download
```

## 5 分钟快速上手

### 1. 字符串模板

```go
package main

import (
	"fmt"
	"log"

	"langchain-go/core/prompts"
)

func main() {
	// 创建模板
	template, err := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
		Template: "Write a {length} {genre} story about {topic}.",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 格式化
	result, _ := template.Format(map[string]any{
		"length": "short",
		"genre":  "sci-fi",
		"topic":  "time travel",
	})

	fmt.Println(result)
	// 输出: Write a short sci-fi story about time travel.
}
```

### 2. 聊天模板

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"langchain-go/core/prompts"
	"langchain-go/core/chat/providers/openai"
)

func main() {
	// 创建聊天模板
	promptTemplate := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("You are a helpful assistant."),
		prompts.HumanMessagePromptTemplate("Tell me about {topic}."),
	)

	// 格式化消息
	messages, _ := promptTemplate.FormatMessages(map[string]any{
		"topic": "Go programming",
	})

	// 创建模型并调用
	model, _ := openai.New(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o-mini",
	})

	response, err := model.Invoke(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response.Content)
}
```

### 3. Few-shot 学习

```go
package main

import (
	"fmt"
	"log"

	"langchain-go/core/prompts"
)

func main() {
	// 定义示例
	examples := []map[string]any{
		{"input": "happy", "output": "sad"},
		{"input": "tall", "output": "short"},
		{"input": "hot", "output": "cold"},
	}

	// 定义示例格式
	examplePrompt, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
		Template: "Input: {input}\nOutput: {output}",
	})

	// 创建 Few-shot 模板
	fewShotPrompt, err := prompts.NewFewShotPromptTemplate(prompts.FewShotConfig{
		Examples:       examples,
		ExamplePrompt:  examplePrompt,
		Prefix:         "Give the antonym of every input.\n\n",
		Suffix:         "\nInput: {input}\nOutput:",
		InputVariables: []string{"input"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用
	result, _ := fewShotPrompt.Format(map[string]any{
		"input": "big",
	})

	fmt.Println(result)
}
```

## 常见场景

### 场景 1: 多语言翻译

```go
template := prompts.NewChatPromptTemplate(
	prompts.SystemMessagePromptTemplate("You are a professional translator."),
	prompts.HumanMessagePromptTemplate("Translate '{text}' from {source} to {target}."),
)

messages, _ := template.FormatMessages(map[string]any{
	"text":   "Hello, world!",
	"source": "English",
	"target": "Spanish",
})
```

### 场景 2: 代码生成

```go
template := prompts.NewChatPromptTemplate(
	prompts.SystemMessagePromptTemplate("You are an expert {language} programmer."),
	prompts.HumanMessagePromptTemplate("Write a function that {task}."),
)

messages, _ := template.FormatMessages(map[string]any{
	"language": "Go",
	"task":     "sorts an array of integers",
})
```

### 场景 3: 问答系统

```go
template := prompts.NewChatPromptTemplate(
	prompts.SystemMessagePromptTemplate("Answer based on the context.\n\nContext: {context}"),
	prompts.HumanMessagePromptTemplate("Question: {question}"),
)

messages, _ := template.FormatMessages(map[string]any{
	"context":  "Go is a programming language created by Google...",
	"question": "Who created Go?",
})
```

## 下一步

- 📖 查看 [完整 Prompts 示例](./prompts-examples.md)
- 🤖 学习 [ChatModel 使用](./chat-examples.md)
- 🔧 了解 [Runnable 系统](../core/runnable/)

## 提示

1. **变量自动检测**: 无需手动指定 InputVariables
2. **部分变量**: 使用 `Partial()` 预填充常用值
3. **错误处理**: 始终检查错误返回值
4. **模板复用**: 创建基础模板，通过 Partial 特化

祝你使用愉快！🎉

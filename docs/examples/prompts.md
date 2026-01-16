# Prompts 系统使用指南

本指南详细介绍如何使用 LangChain-Go 的 Prompts 系统。

## 目录

1. [PromptTemplate - 字符串模板](#prompttemplate---字符串模板)
2. [ChatPromptTemplate - 聊天模板](#chatprompttemplate---聊天模板)
3. [FewShotPromptTemplate - Few-shot 学习](#fewshotprompttemplate---few-shot-学习)
4. [与 ChatModel 集成](#与-chatmodel-集成)
5. [高级用法](#高级用法)

---

## PromptTemplate - 字符串模板

### 基础用法

```go
package main

import (
	"fmt"
	"log"

	"github.com/zhuchenglong/langchain-go/core/prompts"
)

func main() {
	// 创建简单模板
	template, err := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
		Template:       "Tell me a {adjective} joke about {content}.",
		InputVariables: []string{"adjective", "content"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 格式化
	result, err := template.Format(map[string]any{
		"adjective": "funny",
		"content":   "chickens",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// 输出: Tell me a funny joke about chickens.
}
```

### 自动检测变量

不需要显式指定变量列表，模板会自动检测：

```go
template, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: "Hello, {name}! You are {age} years old.",
	// InputVariables 会自动设置为 ["name", "age"]
})
```

### 部分变量（Partial Variables）

预填充部分变量，后续只需提供剩余变量：

```go
// 创建模板
template, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: "Write a {length} {genre} story about {topic}.",
})

// 部分填充
partialTemplate := template.Partial(map[string]any{
	"length": "short",
	"genre":  "science fiction",
})

// 只需提供 topic
result, _ := partialTemplate.Format(map[string]any{
	"topic": "time travel",
})

fmt.Println(result)
// 输出: Write a short science fiction story about time travel.
```

### 使用 Partial Variables 创建模板

```go
template, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: "Today is {date}. {question}",
	PartialVariables: map[string]any{
		"date": time.Now().Format("2006-01-02"),
	},
})

// 只需提供 question
result, _ := template.Format(map[string]any{
	"question": "What's the weather like?",
})
```

---

## ChatPromptTemplate - 聊天模板

### 基础用法

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhuchenglong/langchain-go/core/prompts"
)

func main() {
	// 创建聊天模板
	template := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("You are a helpful {role}."),
		prompts.HumanMessagePromptTemplate("Hello, my name is {name}!"),
		prompts.AIMessagePromptTemplate("Nice to meet you, {name}!"),
		prompts.HumanMessagePromptTemplate("Can you help me with {topic}?"),
	)

	// 格式化为消息列表
	messages, err := template.FormatMessages(map[string]any{
		"role":  "assistant",
		"name":  "Alice",
		"topic": "Python programming",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 打印消息
	for _, msg := range messages {
		fmt.Printf("%s: %s\n", msg.Role, msg.Content)
	}
	/*
	输出:
	system: You are a helpful assistant.
	user: Hello, my name is Alice!
	assistant: Nice to meet you, Alice!
	user: Can you help me with Python programming?
	*/
}
```

### 使用 FromMessages 构造

更灵活的构造方式：

```go
template, _ := prompts.FromMessages([]any{
	[]any{"system", "You are a {role}."},
	[]any{"human", "Hello, {name}!"},
	[]any{"ai", "Hi there!"},
})

messages, _ := template.FormatMessages(map[string]any{
	"role": "teacher",
	"name": "Bob",
})
```

### 部分变量

```go
template := prompts.NewChatPromptTemplate(
	prompts.SystemMessagePromptTemplate("You are a {role}."),
	prompts.HumanMessagePromptTemplate("{name} asks: {question}"),
)

// 预填充角色
partialTemplate := template.Partial(map[string]any{
	"role": "helpful assistant",
})

// 只需提供 name 和 question
messages, _ := partialTemplate.FormatMessages(map[string]any{
	"name":     "Alice",
	"question": "What is Python?",
})
```

---

## FewShotPromptTemplate - Few-shot 学习

Few-shot 学习通过提供示例来引导模型学习任务模式。

### 基础用法

```go
package main

import (
	"fmt"
	"log"

	"github.com/zhuchenglong/langchain-go/core/prompts"
)

func main() {
	// 定义示例
	examples := []map[string]any{
		{"input": "happy", "output": "sad"},
		{"input": "tall", "output": "short"},
		{"input": "energetic", "output": "lethargic"},
		{"input": "sunny", "output": "gloomy"},
		{"input": "windy", "output": "calm"},
	}

	// 定义示例格式
	examplePrompt, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
		Template: "Input: {input}\nOutput: {output}",
	})

	// 创建 Few-shot 模板
	fewShotPrompt, err := prompts.NewFewShotPromptTemplate(prompts.FewShotConfig{
		Examples:         examples,
		ExamplePrompt:    examplePrompt,
		ExampleSeparator: "\n\n",
		Prefix:           "Give the antonym of every input.\n\n",
		Suffix:           "\nInput: {input}\nOutput:",
		InputVariables:   []string{"input"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用
	result, _ := fewShotPrompt.Format(map[string]any{
		"input": "big",
	})

	fmt.Println(result)
	/*
	输出:
	Give the antonym of every input.

	Input: happy
	Output: sad

	Input: tall
	Output: short

	Input: energetic
	Output: lethargic

	Input: sunny
	Output: gloomy

	Input: windy
	Output: calm

	Input: big
	Output:
	*/
}
```

### 使用 LengthBasedExampleSelector

根据长度限制动态选择示例：

```go
// 定义示例
examples := []map[string]any{
	{"input": "happy", "output": "sad"},
	{"input": "tall", "output": "short"},
	{"input": "energetic", "output": "lethargic"},
	{"input": "sunny", "output": "gloomy"},
	// ... 更多示例
}

// 创建示例选择器
examplePrompt, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: "Input: {input}\nOutput: {output}",
})

selector := prompts.NewLengthBasedExampleSelector(
	examples,
	examplePrompt,
	500, // 最大长度（字符数）
)

// 创建 Few-shot 模板（使用选择器）
fewShotPrompt, _ := prompts.NewFewShotPromptTemplate(prompts.FewShotConfig{
	ExamplePrompt:   examplePrompt,
	ExampleSelector: selector, // 使用选择器而不是固定示例
	Prefix:          "Give the antonym of every input.\n\n",
	Suffix:          "\nInput: {input}\nOutput:",
	InputVariables:  []string{"input"},
})

// 使用时会自动选择合适数量的示例
result, _ := fewShotPrompt.Format(map[string]any{
	"input": "beautiful",
})
```

### 动态添加示例

```go
selector := prompts.NewLengthBasedExampleSelector(
	[]map[string]any{}, // 初始为空
	examplePrompt,
	500,
)

// 动态添加示例
selector.AddExample(map[string]any{
	"input":  "good",
	"output": "bad",
})
selector.AddExample(map[string]any{
	"input":  "fast",
	"output": "slow",
})
```

---

## 与 ChatModel 集成

Prompts 可以与 ChatModel 无缝集成，形成完整的处理链。

### 简单链式调用

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zhuchenglong/langchain-go/core/prompts"
	"github.com/zhuchenglong/langchain-go/core/chat/providers/openai"
)

func main() {
	// 创建提示词模板
	promptTemplate := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("You are a helpful assistant."),
		prompts.HumanMessagePromptTemplate("Tell me a {adjective} joke about {topic}."),
	)

	// 创建模型
	model, _ := openai.New(openai.Config{
		APIKey: "sk-...",
		Model:  "gpt-4o-mini",
	})

	// 方式1: 手动链接
	messages, _ := promptTemplate.FormatMessages(map[string]any{
		"adjective": "funny",
		"topic":     "programming",
	})
	response, _ := model.Invoke(context.Background(), messages)
	fmt.Println(response.Content)

	// 方式2: 使用 Pipe (未来版本支持)
	// chain := promptTemplate.Pipe(model)
	// response, _ := chain.Invoke(ctx, map[string]any{...})
}
```

### 使用部分变量固定系统提示词

```go
// 创建基础模板
baseTemplate := prompts.NewChatPromptTemplate(
	prompts.SystemMessagePromptTemplate("You are a {role} specialized in {domain}."),
	prompts.HumanMessagePromptTemplate("{question}"),
)

// 为不同场景创建特化版本
mathTeacher := baseTemplate.Partial(map[string]any{
	"role":   "teacher",
	"domain": "mathematics",
})

scienceTeacher := baseTemplate.Partial(map[string]any{
	"role":   "teacher",
	"domain": "science",
})

// 使用
mathMessages, _ := mathTeacher.FormatMessages(map[string]any{
	"question": "What is calculus?",
})
```

### Few-shot 与 ChatModel

```go
// 创建 Few-shot 提示词
fewShotPrompt, _ := prompts.NewFewShotPromptTemplate(prompts.FewShotConfig{
	Examples: []map[string]any{
		{"question": "What is 2+2?", "answer": "4"},
		{"question": "What is 3*3?", "answer": "9"},
	},
	ExamplePrompt:  exampleTemplate,
	Prefix:         "Answer the math questions.\n\n",
	Suffix:         "\nQuestion: {question}\nAnswer:",
	InputVariables: []string{"question"},
})

// 格式化提示词
promptText, _ := fewShotPrompt.Format(map[string]any{
	"question": "What is 5+5?",
})

// 转换为聊天消息
messages := []types.Message{
	types.NewUserMessage(promptText),
}

// 调用模型
response, _ := model.Invoke(ctx, messages)
```

---

## 高级用法

### 多语言支持

```go
// 创建多语言模板
templates := map[string]*prompts.PromptTemplate{
	"en": promptsNewPromptTemplate(prompts.PromptTemplateConfig{
		Template: "Hello, {name}!",
	}),
	"zh": prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
		Template: "你好，{name}！",
	}),
}

// 根据语言选择
language := "zh"
result, _ := templates[language].Format(map[string]any{
	"name": "张三",
})
```

### 条件渲染（手动实现）

```go
func CreateConditionalPrompt(includeExamples bool) *prompts.ChatPromptTemplate {
	messages := []prompts.MessagePromptTemplate{
		prompts.SystemMessagePromptTemplate("You are a helpful assistant."),
	}

	if includeExamples {
		messages = append(messages,
			prompts.HumanMessagePromptTemplate("Example: {example}"),
		)
	}

	messages = append(messages,
		prompts.HumanMessagePromptTemplate("{question}"),
	)

	return prompts.NewChatPromptTemplate(messages...)
}
```

### 模板复用

```go
// 定义可复用的消息模板
systemPrompt := prompts.SystemMessagePromptTemplate("You are a {role}.")
greetingPrompt := prompts.HumanMessagePromptTemplate("Hello, my name is {name}.")

// 组合不同的模板
template1 := prompts.NewChatPromptTemplate(
	systemPrompt,
	greetingPrompt,
	prompts.HumanMessagePromptTemplate("Tell me about {topic}."),
)

template2 := prompts.NewChatPromptTemplate(
	systemPrompt,
	greetingPrompt,
	prompts.HumanMessagePromptTemplate("Ask me about {subject}."),
)
```

### 嵌套变量替换

```go
// 第一层：格式化内层模板
innerTemplate, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: "a {adjective} person",
})
innerResult, _ := innerTemplate.Format(map[string]any{
	"adjective": "kind",
})

// 第二层：使用内层结果
outerTemplate, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: "Tell me about {description}.",
})
outerResult, _ := outerTemplate.Format(map[string]any{
	"description": innerResult,
})

fmt.Println(outerResult)
// 输出: Tell me about a kind person.
```

### 批量处理

```go
template := prompts.NewChatPromptTemplate(
	prompts.HumanMessagePromptTemplate("Translate '{text}' to {language}."),
)

inputs := []map[string]any{
	{"text": "Hello", "language": "Spanish"},
	{"text": "Goodbye", "language": "French"},
	{"text": "Thank you", "language": "German"},
}

// 批量格式化
results, _ := template.Batch(context.Background(), inputs)

for i, messages := range results {
	fmt.Printf("%d: %s\n", i+1, messages[0].Content)
}
```

---

## 最佳实践

### 1. 变量命名

使用清晰、描述性的变量名：

```go
// ✅ 好
template := "Write a {length} {genre} story about {topic}."

// ❌ 不好
template := "Write a {a} {b} story about {c}."
```

### 2. 模板组织

将复杂模板分解为可复用的组件：

```go
// 定义可复用组件
systemRole := prompts.SystemMessagePromptTemplate("You are a {role}.")
userGreeting := prompts.HumanMessagePromptTemplate("Hello, {name}!")

// 组合使用
func CreateTemplate(includeGreeting bool) *prompts.ChatPromptTemplate {
	messages := []prompts.MessagePromptTemplate{systemRole}
	if includeGreeting {
		messages = append(messages, userGreeting)
	}
	messages = append(messages, prompts.HumanMessagePromptTemplate("{question}"))
	return prompts.NewChatPromptTemplate(messages...)
}
```

### 3. 错误处理

始终检查错误：

```go
template, err := prompts.NewPromptTemplate(config)
if err != nil {
	log.Printf("Failed to create template: %v", err)
	return
}

result, err := template.Format(values)
if err != nil {
	log.Printf("Failed to format template: %v", err)
	return
}
```

### 4. 模板验证

在开发时验证模板：

```go
config := prompts.PromptTemplateConfig{
	Template:         "Hello, {name}!",
	InputVariables:   []string{"name"},
	ValidateTemplate: true, // 启用验证
}
template, err := prompts.NewPromptTemplate(config)
```

### 5. 使用部分变量

对于频繁使用的固定值，使用部分变量：

```go
// 创建基础模板
baseTemplate, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
	Template: "Company: {company}\nProduct: {product}\nQuestion: {question}",
})

// 为特定公司创建特化版本
appleTemplate := baseTemplate.Partial(map[string]any{
	"company": "Apple Inc.",
})

// 使用时只需提供产品和问题
result, _ := appleTemplate.Format(map[string]any{
	"product":  "iPhone",
	"question": "What are the specs?",
})
```

---

## 常见问题

### Q: 如何处理大括号字符？

如果需要在模板中使用字面大括号，使用双大括号：

```go
// 注意：当前实现不支持转义，这是未来功能
// 临时方案：避免使用字面大括号或使用其他分隔符
```

### Q: 模板中的变量会自动转义吗？

不会，变量值按原样插入。如果需要转义，请在传入前处理。

### Q: 可以在模板中使用函数吗？

当前版本不支持模板函数。可以在格式化前预处理数据。

### Q: Few-shot 示例太多怎么办？

使用 `LengthBasedExampleSelector` 根据长度限制自动选择：

```go
selector := prompts.NewLengthBasedExampleSelector(
	examples,
	examplePrompt,
	1000, // 最大长度
)
```

---

## 参考

- [API 文档](https://pkg.go.dev/langchain-go/core/prompts)
- [ChatModel 使用指南](./chat-examples.md)
- [Runnable 系统](../core/runnable/)

---

**最后更新**: 2026-01-14

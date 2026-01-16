# LangChain-Go 快速开始 - OutputParser 系统

本指南帮助你快速上手 LangChain-Go 的 OutputParser 系统。

## 什么是 OutputParser？

OutputParser 从 LLM 的文本输出中提取结构化数据，让你能够：
- 将文本转换为 JSON
- 解析为类型安全的 Go 结构体
- 提取列表、布尔值等数据
- 验证输出格式

## 5 分钟快速上手

### 1. JSON 解析

```go
package main

import (
	"fmt"
	"log"

	"github.com/zhuchenglong/langchain-go/core/output"
)

func main() {
	// 创建 JSON 解析器
	parser := output.NewJSONParser()

	// 解析 LLM 输出
	llmOutput := `{"name": "Alice", "age": 30, "city": "NYC"}`
	
	result, err := parser.Parse(llmOutput)
	if err != nil {
		log.Fatal(err)
	}

	// 使用解析结果
	fmt.Printf("Name: %s\n", result["name"])
	fmt.Printf("Age: %.0f\n", result["age"])
	fmt.Printf("City: %s\n", result["city"])
}
```

### 2. 类型安全的结构化解析

```go
package main

import (
	"fmt"
	"log"

	"github.com/zhuchenglong/langchain-go/core/output"
)

// 定义你的数据结构
type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	City string `json:"city"`
}

func main() {
	// 创建类型安全的解析器
	parser := output.NewStructuredParser[Person]()

	// 解析 LLM 输出
	llmOutput := `{"name": "Bob", "age": 25, "city": "LA"}`
	
	person, err := parser.Parse(llmOutput)
	if err != nil {
		log.Fatal(err)
	}

	// 类型安全访问
	fmt.Printf("Name: %s\n", person.Name)    // string
	fmt.Printf("Age: %d\n", person.Age)      // int
	fmt.Printf("City: %s\n", person.City)    // string
}
```

### 3. 完整的 Prompt + Model + Parser 链路

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zhuchenglong/langchain-go/core/output"
	"github.com/zhuchenglong/langchain-go/core/prompts"
	"github.com/zhuchenglong/langchain-go/core/chat/providers/openai"
)

type MovieReview struct {
	Title      string  `json:"title"`
	Rating     float64 `json:"rating"`
	Summary    string  `json:"summary"`
	Recommend  bool    `json:"recommend"`
}

func main() {
	// 1. 创建解析器
	parser := output.NewStructuredParser[MovieReview]()

	// 2. 创建提示词（包含格式指令）
	template := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("Extract movie review information."),
		prompts.HumanMessagePromptTemplate(`Review: {review}

{format_instructions}`),
	)

	// 3. 格式化提示词
	messages, _ := template.FormatMessages(map[string]any{
		"review": "The Matrix is an amazing sci-fi movie. I rate it 9.5/10. Highly recommended!",
		"format_instructions": parser.GetFormatInstructions(),
	})

	// 4. 调用模型
	model, _ := openai.New(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o-mini",
	})
	response, _ := model.Invoke(context.Background(), messages)

	// 5. 解析输出
	review, err := parser.Parse(response.Content)
	if err != nil {
		panic(err)
	}

	// 6. 使用结构化数据
	fmt.Printf("Movie: %s\n", review.Title)
	fmt.Printf("Rating: %.1f/10\n", review.Rating)
	fmt.Printf("Summary: %s\n", review.Summary)
	fmt.Printf("Recommend: %v\n", review.Recommend)
}
```

## 常用解析器

### JSONParser - 通用 JSON

```go
parser := output.NewJSONParser()
result, _ := parser.Parse(`{"key": "value"}`)
// map[string]any
```

### StructuredParser - 类型安全

```go
type MyData struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}

parser := output.NewStructuredParser[MyData]()
data, _ := parser.Parse(`{"field1": "value", "field2": 42}`)
// MyData (强类型)
```

### ListParser - 列表数据

```go
parser := output.NewListParser(",")
items, _ := parser.Parse("apple, banana, orange")
// []string{"apple", "banana", "orange"}
```

### BooleanParser - 是/否判断

```go
parser := output.NewBooleanParser()
result, _ := parser.Parse("yes")
// true
```

## 常见场景

### 场景 1: 信息提取

```go
type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

parser := output.NewStructuredParser[Contact]()
instructions := parser.GetFormatInstructions()

// 在提示词中使用格式指令
// ... 调用 LLM ...
// 解析结果
contact, _ := parser.Parse(llmOutput)
```

### 场景 2: 分类任务

```go
type Classification struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

parser := output.NewStructuredParser[Classification]()
// ... 使用解析器
```

### 场景 3: 多项提取

```go
type Analysis struct {
	KeyPoints []string          `json:"key_points"`
	Sentiment string            `json:"sentiment"`
	Entities  []map[string]any  `json:"entities"`
}

parser := output.NewStructuredParser[Analysis]()
// ... 使用解析器
```

## 提示

1. **总是使用格式指令**: `parser.GetFormatInstructions()`
2. **类型安全优先**: 能用 StructuredParser 就用
3. **处理解析错误**: 始终检查 err 返回值
4. **智能提取**: Parser 会自动处理 Markdown 和混合文本

## 下一步

- 📖 查看 [完整示例文档](./output-examples.md)
- 🔗 学习 [Prompts 使用](./prompts-examples.md)
- 🤖 了解 [ChatModel 集成](./chat-examples.md)

祝你使用愉快！🎉

# OutputParser 系统使用指南

本指南详细介绍如何使用 LangChain-Go 的 OutputParser 系统。

## 目录

1. [基础概念](#基础概念)
2. [StringOutputParser - 字符串解析](#stringoutputparser---字符串解析)
3. [JSONParser - JSON 解析](#jsonparser---json-解析)
4. [StructuredParser - 结构化解析](#structuredparser---结构化解析)
5. [ListParser - 列表解析](#listparser---列表解析)
6. [BooleanParser - 布尔值解析](#booleanparser---布尔值解析)
7. [与其他组件集成](#与其他组件集成)
8. [最佳实践](#最佳实践)

---

## 基础概念

**OutputParser** 的作用是从 LLM 的文本输出中提取结构化数据。

LLM 输出通常是自由格式的文本，OutputParser 帮助我们：
- 提取 JSON 数据
- 解析列表
- 转换为强类型结构体
- 验证输出格式

所有 OutputParser 都实现了 `Runnable` 接口，可以与其他组件链式组合。

---

## StringOutputParser - 字符串解析

最简单的解析器，原样返回 LLM 输出。

```go
package main

import (
	"context"
	"fmt"
	"os"

	"langchain-go/core/output"
	"langchain-go/core/chat/providers/openai"
	"langchain-go/pkg/types"
)

func main() {
	// 创建解析器
	parser := output.NewStringOutputParser()

	// 创建模型
	model, _ := openai.New(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o-mini",
	})

	// 调用模型
	messages := []types.Message{
		types.NewUserMessage("Say 'Hello, World!'"),
	}
	response, _ := model.Invoke(context.Background(), messages)

	// 解析输出
	result, _ := parser.Parse(response.Content)
	fmt.Println(result) // 原样返回
}
```

---

## JSONParser - JSON 解析

从 LLM 输出中提取 JSON 数据。

### 基础用法

```go
package main

import (
	"fmt"
	"log"

	"langchain-go/core/output"
)

func main() {
	parser := output.NewJSONParser()

	// 场景 1: 纯 JSON
	result, err := parser.Parse(`{"name": "Alice", "age": 30}`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result["name"]) // "Alice"
	fmt.Println(result["age"])  // 30

	// 场景 2: Markdown 代码块中的 JSON
	result, _ = parser.Parse("```json\n{\"name\": \"Bob\"}\n```")
	fmt.Println(result["name"]) // "Bob"

	// 场景 3: 混合文本中的 JSON
	result, _ = parser.Parse("The data is: {\"name\": \"Charlie\"} here.")
	fmt.Println(result["name"]) // "Charlie"
}
```

### 与 Prompts 集成

```go
package main

import (
	"context"
	"fmt"
	"os"

	"langchain-go/core/output"
	"langchain-go/core/prompts"
	"langchain-go/core/chat/providers/openai"
)

func main() {
	// 创建解析器
	parser := output.NewJSONParser()

	// 创建提示词模板（包含格式指令）
	template := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("Extract information as JSON."),
		prompts.HumanMessagePromptTemplate(`Extract person info from: {text}

{format_instructions}`),
	)

	// 格式化提示词
	messages, _ := template.FormatMessages(map[string]any{
		"text":                "John Doe is 35 years old and lives in NYC",
		"format_instructions": parser.GetFormatInstructions(),
	})

	// 调用模型
	model, _ := openai.New(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o-mini",
	})
	response, _ := model.Invoke(context.Background(), messages)

	// 解析输出
	data, _ := parser.Parse(response.Content)
	fmt.Printf("Name: %s\n", data["name"])
	fmt.Printf("Age: %.0f\n", data["age"])
	fmt.Printf("City: %s\n", data["city"])
}
```

### JSON 数组解析

```go
parser := output.NewJSONArrayParser()

// 解析数组
result, _ := parser.Parse(`["apple", "banana", "orange"]`)
for i, item := range result {
	fmt.Printf("%d: %v\n", i, item)
}

// 从 Markdown 提取
result, _ = parser.Parse("```json\n[1, 2, 3]\n```")
// [1, 2, 3]
```

---

## StructuredParser - 结构化解析

类型安全的结构化数据解析器，将 JSON 解析为 Go 结构体。

### 基础用法

```go
package main

import (
	"context"
	"fmt"
	"os"

	"langchain-go/core/output"
	"langchain-go/core/prompts"
	"langchain-go/core/chat/providers/openai"
)

// 定义结构体
type Person struct {
	Name       string `json:"name"`
	Age        int    `json:"age"`
	Occupation string `json:"occupation"`
	City       string `json:"city,omitempty"`
}

func main() {
	// 创建类型安全的解析器
	parser := output.NewStructuredParser[Person]()

	// 获取 Schema（自动从 Person 生成）
	schema := parser.GetSchema()
	fmt.Printf("Schema: %+v\n", schema)

	// 创建提示词
	template := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("Extract person information."),
		prompts.HumanMessagePromptTemplate(`Text: {text}

{format_instructions}`),
	)

	// 格式化
	messages, _ := template.FormatMessages(map[string]any{
		"text":                "Alice is a 30-year-old software engineer in San Francisco",
		"format_instructions": parser.GetFormatInstructions(),
	})

	// 调用模型
	model, _ := openai.New(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o-mini",
	})
	response, _ := model.Invoke(context.Background(), messages)

	// 解析为类型安全的结构体
	person, err := parser.Parse(response.Content)
	if err != nil {
		panic(err)
	}

	// 类型安全访问
	fmt.Printf("Name: %s\n", person.Name)
	fmt.Printf("Age: %d\n", person.Age)
	fmt.Printf("Occupation: %s\n", person.Occupation)
	fmt.Printf("City: %s\n", person.City)
}
```

### 复杂结构体

```go
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zip_code"`
}

type Employee struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Salary    float64 `json:"salary"`
	Address   Address `json:"address"`
	IsActive  bool    `json:"is_active"`
}

// 创建解析器
parser := output.NewStructuredParser[Employee]()

// 解析复杂 JSON
llmOutput := `{
	"id": 1001,
	"name": "Bob Smith",
	"email": "bob@example.com",
	"salary": 75000.50,
	"address": {
		"street": "123 Main St",
		"city": "Boston",
		"zip_code": "02101"
	},
	"is_active": true
}`

employee, _ := parser.Parse(llmOutput)
fmt.Println(employee.Address.City) // "Boston"
```

### 自定义 Schema

```go
import "langchain-go/pkg/types"

// 使用自定义 Schema
customSchema := &types.Schema{
	Type: "object",
	Properties: map[string]types.Schema{
		"name": {
			Type:        "string",
			Description: "Person's full name",
		},
		"age": {
			Type:        "integer",
			Description: "Person's age in years",
			Minimum:     ptr(0.0),
			Maximum:     ptr(150.0),
		},
	},
	Required: []string{"name", "age"},
}

parser := output.NewStructuredParserWithSchema[Person](customSchema)
```

---

## ListParser - 列表解析

解析分隔的文本为字符串列表。

### 基础用法

```go
package main

import (
	"fmt"

	"langchain-go/core/output"
)

func main() {
	// 逗号分隔
	parser := output.NewListParser(",")
	
	result, _ := parser.Parse("apple, banana, orange")
	fmt.Println(result) // ["apple", "banana", "orange"]

	// 换行分隔
	nlParser := output.NewListParser("\n")
	result, _ = nlParser.Parse("line1\nline2\nline3")
	fmt.Println(result) // ["line1", "line2", "line3"]

	// 自动去除空白
	result, _ = parser.Parse("  item1  ,  item2  ,  item3  ")
	fmt.Println(result) // ["item1", "item2", "item3"]
}
```

### 与 LLM 集成

```go
// 创建解析器
parser := output.NewListParser(",")

// 创建提示词
template := prompts.NewChatPromptTemplate(
	prompts.HumanMessagePromptTemplate(`List 5 programming languages.

{format_instructions}`),
)

messages, _ := template.FormatMessages(map[string]any{
	"format_instructions": parser.GetFormatInstructions(),
})

// 调用模型
response, _ := model.Invoke(ctx, messages)

// 解析列表
languages, _ := parser.Parse(response.Content)
for i, lang := range languages {
	fmt.Printf("%d. %s\n", i+1, lang)
}
```

---

## BooleanParser - 布尔值解析

解析文本为布尔值。

```go
parser := output.NewBooleanParser()

// 支持多种格式
result, _ := parser.Parse("yes")      // true
result, _ = parser.Parse("no")        // false
result, _ = parser.Parse("true")      // true
result, _ = parser.Parse("false")     // false
result, _ = parser.Parse("1")         // true
result, _ = parser.Parse("0")         // false

// 大小写不敏感
result, _ = parser.Parse("TRUE")      // true
result, _ = parser.Parse("Yes")       // true

// 自动去除空白
result, _ = parser.Parse("  yes  ")   // true
```

### 用于判断任务

```go
// 创建提示词
template := prompts.NewChatPromptTemplate(
	prompts.HumanMessagePromptTemplate(`Is the following statement true or false?

Statement: {statement}

{format_instructions}

Answer:`),
)

// 创建解析器
parser := output.NewBooleanParser()

// 格式化
messages, _ := template.FormatMessages(map[string]any{
	"statement":           "The Earth is flat",
	"format_instructions": parser.GetFormatInstructions(),
})

// 调用模型
response, _ := model.Invoke(ctx, messages)

// 解析为布尔值
isTruevalue, _ := parser.Parse(response.Content)
fmt.Printf("Is true: %v\n", isTrue)
```

---

## 与其他组件集成

### Prompt + Model + Parser 完整链路

```go
package main

import (
	"context"
	"fmt"
	"os"

	"langchain-go/core/output"
	"langchain-go/core/prompts"
	"langchain-go/core/chat/providers/openai"
)

type Recipe struct {
	Name        string   `json:"name"`
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
	PrepTime    int      `json:"prep_time_minutes"`
}

func main() {
	// 1. 创建解析器
	parser := output.NewStructuredParser[Recipe]()

	// 2. 创建提示词（包含格式指令）
	template := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("You are a chef. Extract recipe information."),
		prompts.HumanMessagePromptTemplate(`Create a recipe for {dish}.

{format_instructions}`),
	)

	// 3. 格式化提示词
	messages, _ := template.FormatMessages(map[string]any{
		"dish":                "chocolate chip cookies",
		"format_instructions": parser.GetFormatInstructions(),
	})

	// 4. 调用模型
	model, _ := openai.New(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o-mini",
	})
	response, _ := model.Invoke(context.Background(), messages)

	// 5. 解析输出
	recipe, err := parser.Parse(response.Content)
	if err != nil {
		panic(err)
	}

	// 6. 使用结构化数据
	fmt.Printf("Recipe: %s\n", recipe.Name)
	fmt.Printf("Prep Time: %d minutes\n", recipe.PrepTime)
	fmt.Println("\nIngredients:")
	for _, ing := range recipe.Ingredients {
		fmt.Printf("  - %s\n", ing)
	}
	fmt.Println("\nSteps:")
	for i, step := range recipe.Steps {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
}
```

### 批量处理

```go
// 批量解析多个输出
parser := output.NewJSONParser()

llmOutputs := []string{
	`{"name": "Alice", "score": 95}`,
	`{"name": "Bob", "score": 87}`,
	`{"name": "Charlie", "score": 92}`,
}

results, err := parser.Batch(context.Background(), llmOutputs)
if err != nil {
	panic(err)
}

for _, data := range results {
	fmt.Printf("%s: %.0f\n", data["name"], data["score"])
}
```

### 错误处理和重试

```go
result, err := parser.Parse(llmOutput)
if err != nil {
	// 检查是否是解析错误
	if parseErr, ok := err.(*output.ParseError); ok {
		fmt.Printf("Failed to parse: %s\n", parseErr.Output)
		fmt.Printf("Error: %v\n", parseErr.Err)
		
		// 可以尝试修复或使用备选方案
		// ...
	}
}
```

---

## 实战示例

### 示例 1: 信息提取

```go
type ArticleMetadata struct {
	Title    string   `json:"title"`
	Author   string   `json:"author"`
	Date     string   `json:"date"`
	Tags     []string `json:"tags"`
	Summary  string   `json:"summary"`
}

func extractArticleMetadata(articleText string) (*ArticleMetadata, error) {
	parser := output.NewStructuredParser[ArticleMetadata]()
	
	template := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate("Extract article metadata."),
		prompts.HumanMessagePromptTemplate(`Article: {article}

{format_instructions}`),
	)

	messages, _ := template.FormatMessages(map[string]any{
		"article":             articleText,
		"format_instructions": parser.GetFormatInstructions(),
	})

	response, _ := model.Invoke(ctx, messages)
	return parser.Parse(response.Content)
}
```

### 示例 2: 情感分析

```go
type SentimentAnalysis struct {
	Sentiment string  `json:"sentiment"` // "positive", "negative", "neutral"
	Score     float64 `json:"score"`     // 0.0 to 1.0
	Keywords  []string `json:"keywords"`
}

func analyzeSentiment(text string) (*SentimentAnalysis, error) {
	parser := output.NewStructuredParser[SentimentAnalysis]()
	
	template := prompts.NewChatPromptTemplate(
		prompts.HumanMessagePromptTemplate(`Analyze the sentiment of: "{text}"

{format_instructions}`),
	)

	messages, _ := template.FormatMessages(map[string]any{
		"text":                text,
		"format_instructions": parser.GetFormatInstructions(),
	})

	response, _ := model.Invoke(ctx, messages)
	return parser.Parse(response.Content)
}
```

### 示例 3: 多步骤提取

```go
type ExtractedData struct {
	Entities []Entity `json:"entities"`
	Summary  string   `json:"summary"`
}

type Entity struct {
	Name string `json:"name"`
	Type string `json:"type"` // "person", "organization", "location"
}

func extractEntities(text string) (*ExtractedData, error) {
	parser := output.NewStructuredParser[ExtractedData]()
	
	// 使用 Few-shot 提示词
	examples := []map[string]any{
		{
			"input": "Apple Inc. announced new products in Cupertino.",
			"output": `{"entities": [{"name": "Apple Inc.", "type": "organization"}, {"name": "Cupertino", "type": "location"}], "summary": "Company announcement"}`,
		},
	}
	
	examplePrompt, _ := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
		Template: "Input: {input}\nOutput: {output}",
	})
	
	fewShotPrompt, _ := prompts.NewFewShotPromptTemplate(prompts.FewShotConfig{
		Examples:       examples,
		ExamplePrompt:  examplePrompt,
		Prefix:         "Extract entities from text.\n\n",
		Suffix:         "\nInput: {text}\n{format_instructions}\nOutput:",
		InputVariables: []string{"text"},
	})

	promptText, _ := fewShotPrompt.Format(map[string]any{
		"text":                text,
		"format_instructions": parser.GetFormatInstructions(),
	})

	messages := []types.Message{types.NewUserMessage(promptText)}
	response, _ := model.Invoke(ctx, messages)
	
	return parser.Parse(response.Content)
}
```

### 示例 4: 问答系统

```go
type QAResult struct {
	Answer     string   `json:"answer"`
	Sources    []string `json:"sources"`
	Confidence float64  `json:"confidence"`
}

func answerQuestion(question, context string) (*QAResult, error) {
	parser := output.NewStructuredParser[QAResult]()
	
	template := prompts.NewChatPromptTemplate(
		prompts.SystemMessagePromptTemplate(`Answer questions based on the context.
Provide sources and confidence score (0.0-1.0).`),
		prompts.HumanMessagePromptTemplate(`Context: {context}

Question: {question}

{format_instructions}`),
	)

	messages, _ := template.FormatMessages(map[string]any{
		"context":             context,
		"question":            question,
		"format_instructions": parser.GetFormatInstructions(),
	})

	response, _ := model.Invoke(ctx, messages)
	return parser.Parse(response.Content)
}
```

---

## 最佳实践

### 1. 选择合适的解析器

```go
// ✅ 需要类型安全 -> StructuredParser
parser := output.NewStructuredParser[MyStruct]()

// ✅ 只需要 map -> JSONParser
parser := output.NewJSONParser()

// ✅ 简单列表 -> ListParser
parser := output.NewListParser(",")

// ✅ 不需要解析 -> StringOutputParser
parser := output.NewStringOutputParser()
```

### 2. 提供清晰的格式指令

```go
// 总是在提示词中包含格式指令
messages, _ := template.FormatMessages(map[string]any{
	"input":               userInput,
	"format_instructions": parser.GetFormatInstructions(),
})
```

### 3. 错误处理

```go
result, err := parser.Parse(llmOutput)
if err != nil {
	// 记录错误，可能需要重试
	log.Printf("Parse failed: %v", err)
	
	// 可以尝试使用另一个解析器
	// 或者要求 LLM 重新生成
	return nil, err
}
```

### 4. 验证解析结果

```go
person, err := parser.Parse(llmOutput)
if err != nil {
	return err
}

// 验证数据的有效性
if person.Age < 0 || person.Age > 150 {
	return fmt.Errorf("invalid age: %d", person.Age)
}

if person.Name == "" {
	return fmt.Errorf("name is required")
}
```

### 5. 使用结构化输出模式

对于支持的模型（如 GPT-4o），使用 ChatModel 的 `WithStructuredOutput`:

```go
// 方式 1: 使用 ChatModel 的结构化输出（推荐）
schema := parser.GetSchema()
modelWithSchema := model.WithStructuredOutput(*schema)
response, _ := modelWithSchema.Invoke(ctx, messages)
// response.Content 已经是格式化的 JSON

// 方式 2: 手动解析（更灵活）
response, _ := model.Invoke(ctx, messages)
result, _ := parser.Parse(response.Content)
```

---

## 常见问题

### Q: 如何处理嵌套的 JSON？

使用嵌套的结构体：

```go
type Parent struct {
	Name  string  `json:"name"`
	Child Child   `json:"child"`
}

type Child struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

parser := output.NewStructuredParser[Parent]()
```

### Q: 如何处理可选字段？

使用 `omitempty` tag 或指针：

```go
type Data struct {
	Required string  `json:"required"`
	Optional string  `json:"optional,omitempty"`
	Pointer  *string `json:"pointer,omitempty"`
}
```

### Q: 解析失败怎么办？

```go
result, err := parser.Parse(llmOutput)
if err != nil {
	// 选项 1: 重试（使用不同的提示词）
	// 选项 2: 使用更宽松的解析器
	// 选项 3: 要求 LLM 修复输出
}
```

### Q: 如何提高解析成功率？

1. **提供清晰的格式指令**
2. **使用 Few-shot 示例**
3. **选择支持结构化输出的模型**（如 GPT-4o）
4. **验证和重试机制**

---

## 参考

- [API 文档](https://pkg.go.dev/langchain-go/core/output)
- [Prompts 使用指南](./prompts-examples.md)
- [ChatModel 使用指南](./chat-examples.md)

---

**最后更新**: 2026-01-14

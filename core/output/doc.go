// Package output 提供 LangChain 输出解析器（OutputParser）的实现。
//
// output 包实现了从 LLM 输出中提取结构化数据的解析器系统。
// 所有 OutputParser 都实现了 runnable.Runnable 接口，可以与其他组件无缝组合。
//
// 核心特性：
//   - 通用的 OutputParser 接口
//   - JSON 解析器（从文本中提取 JSON）
//   - 结构化输出解析器（类型安全的结构化数据）
//   - 列表解析器（解析列表数据）
//   - 自动重试和修复
//   - 与 Runnable 系统集成
//
// 基本用法：
//
//	import (
//	    "context"
//	    "github.com/zhucl121/langchain-go/core/output"
//	)
//
//	// 创建 JSON 解析器
//	parser := output.NewJSONParser()
//
//	// 解析 LLM 输出
//	llmOutput := `{"name": "Alice", "age": 30}`
//	result, err := parser.Parse(llmOutput)
//	if err != nil {
//	    panic(err)
//	}
//
//	data := result.(map[string]any)
//	fmt.Println(data["name"]) // "Alice"
//
// 结构化输出示例：
//
//	type Person struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	}
//
//	parser := output.NewStructuredParser[Person]()
//
//	llmOutput := `{"name": "Bob", "age": 25}`
//	person, err := parser.Parse(llmOutput)
//	if err != nil {
//	    panic(err)
//	}
//
//	fmt.Println(person.Name) // "Bob"
//	fmt.Println(person.Age)  // 25
//
// 与提示词集成：
//
//	// 解析器可以提供格式指令
//	instructions := parser.GetFormatInstructions()
//
//	prompt := prompts.NewPromptTemplate(prompts.PromptTemplateConfig{
//	    Template: "Extract person info.\n\n{format_instructions}\n\nText: {text}",
//	})
//
//	// 将格式指令传递给提示词
//	formatted, _ := prompt.Format(map[string]any{
//	    "format_instructions": instructions,
//	    "text":                "Alice is 30 years old",
//	})
//
// 链式调用示例：
//
//	// Prompt -> Model -> Parser
//	chain := promptTemplate.
//	    Pipe(model).
//	    Pipe(parser)
//
//	result, _ := chain.Invoke(ctx, map[string]any{"input": "..."})
//
// 列表解析器：
//
//	parser := output.NewListParser(",")
//	result, _ := parser.Parse("apple, banana, orange")
//	// []string{"apple", "banana", "orange"}
//
// 自动重试：
//
//	parser := output.NewJSONParserWithRetry(
//	    fixingModel, // 用于修复错误的模型
//	    maxRetries,
//	)
//
package output

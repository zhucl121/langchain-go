# 核心功能指南

LangChain 核心组件的详细使用指南。

---

## 📖 指南列表

### 基础组件
- [Runnable 系统](./runnable.md) - LCEL 链式组合
- [ChatModel 集成](./chat-models.md) - 对话模型使用
- [Prompts 模板](./prompts.md) - 提示词工程
- [OutputParser 解析](./output-parsers.md) - 结构化输出
- [Tools 工具](./tools.md) - 工具定义和使用
- Memory 记忆 - 对话历史管理（即将添加）

---

## 🎯 学习路径

### 第一步：理解 Runnable
[Runnable 系统](./runnable.md)是 LangChain-Go 的核心抽象，掌握它可以让你：
- 使用 LCEL 风格链式组合
- 理解 Invoke/Batch/Stream 模式
- 构建可重用的组件

### 第二步：集成 LLM
[ChatModel 集成](./chat-models.md)教你如何：
- 配置 OpenAI 和 Anthropic
- 处理流式输出
- 使用 Function Calling

### 第三步：设计提示词
[Prompts 模板](./prompts.md)帮你：
- 创建可重用的提示词模板
- 实现 Few-Shot 学习
- 管理提示词变量

### 第四步：解析输出
[OutputParser 解析](./output-parsers.md)让你：
- 解析 JSON 输出
- 创建类型安全的结构化解析器
- 自动生成 Schema

### 第五步：使用工具
[Tools 工具](./tools.md)教你：
- 创建自定义工具
- 集成内置工具
- 在 Agent 中使用工具

---

## 💡 快速示例

### Runnable 链式组合
```go
chain := prompt.Pipe(model).Pipe(parser)
result, _ := chain.Invoke(ctx, input)
```

### ChatModel 调用
```go
model := openai.New(openai.Config{APIKey: "sk-..."})
response, _ := model.Invoke(ctx, messages)
```

### Prompts 模板
```go
template := prompts.NewPromptTemplate("Tell me about {topic}")
prompt, _ := template.Format(map[string]any{"topic": "AI"})
```

### OutputParser 解析
```go
parser := output.NewJSONParser[MyStruct]()
result, _ := parser.Parse(response.Content)
```

### Tools 使用
```go
tool := tools.NewFunctionTool("calculator", calcFunc, schema)
result, _ := tool.Execute(ctx, input)
```

---

## 📚 相关资源

- [快速开始](../../getting-started/) - 新手入门
- [LangGraph 指南](../langgraph/) - 工作流系统
- [Agent 指南](../agents/) - Agent 系统
- [示例代码](../../examples/) - 实用示例

---

<div align="center">

**[⬆ 回到指南首页](../README.md)** | **[回到文档首页](../../README.md)**

</div>

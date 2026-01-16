# LangChain-Go 快速开始指南

> 更新时间: 2026-01-16  
> 版本: v0.1.0 (修复完成版)

---

## 🚀 快速开始

### 1. 验证安装

```bash
cd langchain-go
./verify.sh
```

**预期输出**: 
- ✅ 编译通过
- ✅ 测试通过
- ✅ 示例程序大部分可用

---

## 📚 常用命令

### 编译和测试

```bash
# 编译所有包
go build $(go list ./... | grep -v '/examples')

# 运行所有测试
go test $(go list ./... | grep -v '/examples')

# 编译单个示例
go build examples/agent_simple_demo.go

# 运行示例
go run examples/agent_simple_demo.go
```

### 代码质量检查

```bash
# 运行 go vet
go vet ./...

# 格式化代码
go fmt ./...

# 检查依赖
go mod tidy
```

---

## 🎯 核心功能使用

### 1. 创建简单 Agent

```go
package main

import (
    "context"
    "log"
    
    "langchain-go/core/agents"
    "langchain-go/core/chat/providers/openai"
    "langchain-go/core/tools"
)

func main() {
    // 1. 创建 LLM
    llm, err := openai.New(openai.Config{
        APIKey: "your-api-key",
        Model:  "gpt-3.5-turbo",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. 创建工具
    tools := []tools.Tool{
        tools.NewCalculatorTool(),
    }
    
    // 3. 创建 Agent
    agent := agents.CreateReActAgent(llm, tools)
    
    // 4. 运行
    executor := agents.NewSimplifiedAgentExecutor(agent, tools)
    result, _ := executor.Run(context.Background(), "计算 25 * 4")
    
    log.Println(result)
}
```

### 2. 使用工具

```go
// 获取内置工具
basicTools := tools.GetBasicTools()
timeTools := tools.GetTimeTools()
allTools := tools.GetBuiltinTools()

// 创建自定义工具
customTool := tools.NewFunctionTool(tools.FunctionToolConfig{
    Name:        "my_tool",
    Description: "My custom tool",
    Fn: func(ctx context.Context, input map[string]any) (any, error) {
        return "result", nil
    },
})
```

### 3. 搜索工具

```go
import "langchain-go/core/tools/search"

// 创建搜索工具
provider := search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{})
searchTool, _ := search.NewSearchTool(provider, search.SearchOptions{
    MaxResults: 5,
})
```

### 4. Multi-Agent 系统

```go
// 创建协调器
coordinator := agents.NewCoordinatorAgent("coordinator", llm, strategy)

// 创建系统
system := agents.NewMultiAgentSystem(coordinator, config)

// 添加专用 Agent
researcher := agents.NewResearcherAgent("researcher", llm, searchTool)
system.AddAgent("researcher", researcher)
```

---

## 📖 可用示例

### 成功运行的示例 (9个)

1. **agent_simple_demo.go** - 基础 Agent 使用
2. **advanced_search_demo.go** - 高级搜索功能
3. **multi_agent_demo.go** - Multi-Agent 系统
4. **multimodal_demo.go** - 多模态工具
5. **pdf_loader_demo.go** - PDF 加载
6. **prompt_hub_demo.go** - Prompt 管理
7. **search_tools_demo.go** - 搜索工具
8. **selfask_agent_demo.go** - Self-Ask Agent
9. **structured_chat_demo.go** - 结构化对话

### 运行示例

```bash
# 简单 Agent
go run examples/agent_simple_demo.go

# 搜索工具
go run examples/search_tools_demo.go

# Multi-Agent
go run examples/multi_agent_demo.go
```

---

## 🔧 常见问题

### Q: 编译错误 "package not found"
```bash
go mod tidy
go mod download
```

### Q: 测试失败
```bash
# 清理缓存
go clean -testcache
go test ./...
```

### Q: 示例程序需要 API Key
```bash
# 设置环境变量
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
```

---

## 📝 API 变更说明

### OpenAI Client

```go
// ❌ 旧版本
llm := openai.NewChatOpenAI("gpt-3.5-turbo")

// ✅ 新版本
llm, err := openai.New(openai.Config{
    APIKey: "your-api-key",
    Model:  "gpt-3.5-turbo",
})
```

### 工具调用

```go
// ❌ 旧版本
tools.NewCalculator()

// ✅ 新版本
tools.NewCalculatorTool()
```

### FunctionTool

```go
// ❌ 旧版本
tools.NewFunctionTool("name", "desc", fn)

// ✅ 新版本
tools.NewFunctionTool(tools.FunctionToolConfig{
    Name:        "name",
    Description: "desc",
    Fn:          fn,
})
```

### Memory

```go
// ❌ 旧版本
memory.NewBufferMemory(10)

// ✅ 新版本
memory.NewBufferMemory()
```

---

## 🎓 学习路径

### 初学者
1. 阅读 `examples/agent_simple_demo.go`
2. 运行基础示例
3. 尝试修改参数

### 进阶
1. 学习 Multi-Agent 系统
2. 创建自定义工具
3. 集成搜索功能

### 高级
1. 实现自定义 Agent 类型
2. 扩展 Provider 支持
3. 性能优化

---

## 📚 文档资源

### 项目文档
- `COMPLETION_SUMMARY.md` - 详细修复过程
- `FINAL_REPORT.md` - 完整报告
- `REMAINING_ISSUES.md` - 已知问题
- `README.md` - 项目说明

### 在线资源
- 代码示例: `examples/` 目录
- 测试用例: `*_test.go` 文件
- API 文档: Go doc 注释

---

## 🤝 贡献指南

### 报告问题
1. 运行 `./verify.sh` 检查状态
2. 收集错误信息
3. 提供复现步骤

### 提交代码
1. Fork 项目
2. 创建特性分支
3. 编写测试
4. 提交 Pull Request

### 代码规范
```bash
# 格式化
go fmt ./...

# 检查
go vet ./...

# 测试
go test ./...
```

---

## 🎯 下一步

### 立即可用
- ✅ 开发新功能
- ✅ 集成到项目
- ✅ 学习和实验

### 后续改进
- [ ] 实现 Ollama Provider
- [ ] 完善 BaseChatModel
- [ ] 提升测试覆盖率

---

## 💡 小贴士

1. **使用验证脚本**: `./verify.sh` 快速检查项目状态
2. **查看示例**: `examples/` 目录有完整的使用示例
3. **阅读测试**: 测试文件是最好的 API 文档
4. **渐进学习**: 从简单示例开始,逐步深入

---

## 📞 获取帮助

### 项目状态
```bash
./verify.sh
```

### 详细文档
- 修复过程: `COMPLETION_SUMMARY.md`
- 完整报告: `FINAL_REPORT.md`
- 已知问题: `REMAINING_ISSUES.md`

---

**祝使用愉快!** 🚀

如有问题,请参考文档或提交 Issue。

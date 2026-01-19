# LangChain-Go 文档

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Documentation](https://img.shields.io/badge/docs-latest-brightgreen.svg)](https://pkg.go.dev/langchain-go)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](../LICENSE)

**完整的 LangChain & LangGraph Go 实现文档**

[快速开始](#-快速开始) • [使用指南](#-使用指南) • [示例](#-示例) • [API 文档](#-api-文档)

</div>

---

## 🎉 最新更新 (v0.1.1 - 2026-01-19)

**重大更新：15个核心功能全部完成！**

- ✅ 4个新向量存储 (Chroma, Qdrant, Weaviate, Redis)
- ✅ 3个新LLM提供商 (Gemini, Bedrock, Azure)
- ✅ 3个新文档加载器 (GitHub, Confluence, PostgreSQL)
- ✅ 4个高级RAG技术 (Multi-Query, HyDE, Parent Document, Self-Query)
- ✅ LCEL等效语法实现

**详细报告**: [COMPLETION_REPORT.md](./COMPLETION_REPORT.md)  
**使用指南**: [高级 RAG 检索技术](./guides/rag/advanced-retrievers.md)

---

## 📚 文档导航

### 🚀 快速开始
新手入门，5分钟上手 LangChain-Go

- [安装指南](./getting-started/installation.md) - 环境准备和安装
- [快速开始](./getting-started/quickstart.md) - 5分钟入门教程
- [ChatModel 快速开始](./getting-started/quickstart-chat.md) - 对话模型使用
- [Prompts 快速开始](./getting-started/quickstart-prompts.md) - 提示词模板
- [OutputParser 快速开始](./getting-started/quickstart-output.md) - 输出解析
- [Tools 快速开始](./getting-started/quickstart-tools.md) - 工具系统
- [Memory 快速开始](./getting-started/quickstart-memory.md) - 记忆系统
- [StateGraph 快速开始](./getting-started/quickstart-stategraph.md) - 状态图工作流

### 📖 使用指南
详细的功能使用文档和最佳实践

#### 核心功能
- [Runnable 系统](./guides/core/runnable.md) - LCEL 链式组合
- [ChatModel 集成](./guides/core/chat-models.md) - OpenAI、Anthropic
- [Prompts 模板](./guides/core/prompts.md) - 提示词工程
- [OutputParser 解析](./guides/core/output-parsers.md) - 结构化输出
- [Tools 工具](./guides/core/tools.md) - 工具定义和使用
- [Memory 记忆](./guides/core/memory.md) - 对话历史管理

#### LangGraph
- [StateGraph 状态图](./guides/langgraph/stategraph.md) - 工作流编排
- [Checkpoint 检查点](./guides/langgraph/checkpoint.md) - 状态持久化
- [Durability 持久性](./guides/langgraph/durability.md) - 故障恢复

#### Agent 系统
- [Agent 概述](./guides/agents/overview.md) - Agent 系统介绍
- [Plan-Execute Agent](./guides/agents/plan-execute.md) - 计划执行

#### RAG 系统
- [RAG 概述](./guides/rag/overview.md) - RAG 系统介绍
- [Milvus](./guides/rag/milvus.md) - Milvus 使用和 Hybrid Search
- [MMR 搜索](./guides/rag/mmr.md) - 最大边际相关性
- [LLM Reranking](./guides/rag/reranking.md) - 智能重排序
- [PDF 加载器](./guides/rag/pdf-loader.md) - PDF 文档处理

### 🔬 高级主题
生产级功能和最佳实践

- [搜索工具](./advanced/search-tools.md) - Google、Bing、DuckDuckGo
- [性能优化](./advanced/performance.md) - 性能调优指南

### 💡 示例
实用代码示例和最佳实践

- [示例索引](./examples/) - 所有示例列表

### 📚 API 文档
完整的 API 参考文档

- [GoDoc](https://pkg.go.dev/langchain-go) - 完整的 API 文档
- [核心类型](./api/#core-types) - Message、Tool、Schema
- [Runnable 接口](./api/#runnable) - LCEL 接口
- [ChatModel 接口](./api/#chatmodel) - 对话模型接口

### 🛠️ 开发文档
为贡献者准备的开发指南

- [项目进度](./development/project-progress.md) - 开发进度

### 📋 参考资料
路线图、FAQ 和其他参考信息

- [扩展功能清单](./reference/enhancements.md) - 增强功能

---

## 🎯 推荐学习路径

### 初学者路径
1. [安装指南](./getting-started/installation.md)
2. [5分钟快速开始](./getting-started/quickstart.md)
3. [ChatModel 快速开始](./getting-started/quickstart-chat.md)
4. [Prompts 快速开始](./getting-started/quickstart-prompts.md)

### 进阶路径
1. [StateGraph 工作流](./guides/langgraph/stategraph.md)
2. [Agent 系统](./guides/agents/overview.md)
3. [RAG 系统](./guides/rag/overview.md)

### 生产部署路径
1. [Checkpoint 持久化](./guides/langgraph/checkpoint.md)
2. [Durability 故障恢复](./guides/langgraph/durability.md)
3. [性能优化](./advanced/performance.md)

---

## 🔍 快速查找

### 我想...

- **开始使用 LangChain-Go** → [快速开始](../QUICK_START.md)
- **构建 Agent** → [Agent 指南](./guides/agents/overview.md)
- **实现 RAG** → [RAG 指南](./guides/rag/overview.md)
- **贡献代码** → [贡献指南](../CONTRIBUTING.md)
- **查看 API** → [GoDoc](https://pkg.go.dev/github.com/zhucl121/langchain-go)

---

## 📝 文档约定

### 代码示例

所有代码示例都经过测试验证。示例格式：

```go
// 简单示例
model := openai.New(openai.Config{APIKey: "sk-..."})
response, _ := model.Invoke(ctx, []types.Message{
    types.NewUserMessage("Hello!"),
})
```

### 符号说明

- 🚀 快速开始
- 📖 使用指南
- 🔬 高级主题
- 💡 示例代码
- ⚠️ 注意事项
- 💡 提示
- 📝 最佳实践

---

## 🆘 获取帮助

- **文档问题**: [提交 Issue](https://github.com/zhucl121/langchain-go/issues)
- **功能请求**: [Feature Request](https://github.com/zhucl121/langchain-go/issues/new?template=feature_request.md)
- **Bug 报告**: [Bug Report](https://github.com/zhucl121/langchain-go/issues/new?template=bug_report.md)
- **讨论交流**: [Discussions](https://github.com/zhucl121/langchain-go/discussions)

---

## 📖 相关资源

- [主项目 README](../README.md)
- [变更日志](../CHANGELOG.md)
- [贡献指南](../CONTRIBUTING.md)
- [安全政策](../SECURITY.md)

---

<div align="center">

**[⬆ 回到顶部](#langchain-go-文档)**

Made with ❤️ by the LangChain-Go Team

</div>

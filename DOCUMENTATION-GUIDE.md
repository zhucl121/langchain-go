# LangChain-Go 文档导航指南 🧭

快速找到你需要的文档！

---

## 🚀 我是新手，想快速上手

👉 **[快速开始指南](docs/getting-started/)**

1. **[安装指南](docs/getting-started/installation.md)** - 第一步，设置开发环境
2. **[5 分钟快速开始](docs/getting-started/quickstart.md)** - Hello World
3. **[ChatModel 快速开始](docs/getting-started/quickstart-chat.md)** - 对话模型
4. **[Prompts 快速开始](docs/getting-started/quickstart-prompts.md)** - 提示词模板
5. **[StateGraph 快速开始](docs/getting-started/quickstart-stategraph.md)** - 工作流
6. **[Tools 快速开始](docs/getting-started/quickstart-tools.md)** - 工具系统

---

## 📖 我想深入学习某个功能

### 核心功能
👉 **[核心功能指南](docs/guides/core/)**

- [Runnable 系统](docs/guides/core/runnable.md) - 链式组合
- [ChatModel](docs/guides/core/chat-models.md) - 对话模型
- [Prompts](docs/guides/core/prompts.md) - 提示词
- [OutputParser](docs/guides/core/output-parsers.md) - 输出解析
- [Tools](docs/guides/core/tools.md) - 工具

### LangGraph 工作流
👉 **[LangGraph 指南](docs/guides/langgraph/)**

- [StateGraph](docs/guides/langgraph/stategraph.md) - 状态图
- [Checkpoint](docs/guides/langgraph/checkpoint.md) - 持久化
- [Durability](docs/guides/langgraph/durability.md) - 容错

### Agent 系统
👉 **[Agent 指南](docs/guides/agents/)**

- [Agent 概述](docs/guides/agents/overview.md)
- [Plan-Execute Agent](docs/guides/agents/plan-execute.md)

### RAG 系统
👉 **[RAG 指南](docs/guides/rag/)**

- [RAG 概述](docs/guides/rag/overview.md)
- [Milvus](docs/guides/rag/milvus.md)
- [Hybrid Search](docs/guides/rag/milvus-hybrid.md)
- [MMR 搜索](docs/guides/rag/mmr.md)
- [LLM Reranking](docs/guides/rag/reranking.md)
- [PDF 加载器](docs/guides/rag/pdf-loader.md)

---

## 💡 我需要代码示例

👉 **[示例代码](docs/examples/)**

- [Chat 示例](docs/examples/chat-examples.md)
- [Prompt 示例](docs/examples/prompt-examples.md)
- [Output 示例](docs/examples/output-examples.md)
- [Tool 示例](docs/examples/tool-examples.md)

完整应用示例：**[examples/ 目录](examples/)**

---

## 🔥 我想了解高级主题

👉 **[高级主题](docs/advanced/)**

- [搜索工具](docs/advanced/search-tools.md) - Google/Bing/DuckDuckGo
- [OpenTelemetry](docs/advanced/opentelemetry.md) - 分布式追踪
- [Prometheus](docs/advanced/prometheus.md) - 监控指标
- [图可视化](docs/advanced/graph-visualization.md) - Mermaid 可视化
- [性能优化](docs/advanced/performance.md) - 优化建议

---

## 🔍 我需要查询 API

👉 **[API 参考](docs/api/)**

- [API 文档](docs/api/README.md)
- [GoDoc 在线文档](https://pkg.go.dev/langchain-go)

---

## 👨‍💻 我想参与开发

👉 **[开发者文档](docs/development/)**

- [项目进度](docs/development/project-progress.md) - 开发状态
- [贡献指南](docs/development/contributing.md) - 如何贡献
- [测试指南](docs/development/testing-guide.md) - 测试规范

---

## 📚 我想查看规划和参考

👉 **[参考文档](docs/reference/)**

- [扩展功能清单](docs/reference/enhancements.md) - 功能规划
- [功能实现说明](docs/reference/simplified-implementations.md) - 实现细节

---

## 🎯 按场景查找

### 场景 1: 构建聊天机器人
1. [ChatModel 快速开始](docs/getting-started/quickstart-chat.md)
2. [ChatModel 详细指南](docs/guides/core/chat-models.md)
3. [Prompts 指南](docs/guides/core/prompts.md)
4. [Chat 示例](docs/examples/chat-examples.md)

### 场景 2: 构建 RAG 应用
1. [RAG 概述](docs/guides/rag/overview.md)
2. [Milvus 使用指南](docs/guides/rag/milvus.md)
3. [PDF 加载器](docs/guides/rag/pdf-loader.md)
4. [MMR 搜索](docs/guides/rag/mmr.md)
5. [LLM Reranking](docs/guides/rag/reranking.md)

### 场景 3: 构建复杂工作流
1. [StateGraph 快速开始](docs/getting-started/quickstart-stategraph.md)
2. [StateGraph 详细指南](docs/guides/langgraph/stategraph.md)
3. [Checkpoint 持久化](docs/guides/langgraph/checkpoint.md)

### 场景 4: 构建 Agent
1. [Agent 概述](docs/guides/agents/overview.md)
2. [Tools 指南](docs/guides/core/tools.md)
3. [Plan-Execute Agent](docs/guides/agents/plan-execute.md)
4. [搜索工具](docs/advanced/search-tools.md)

### 场景 5: 生产部署
1. [安装指南](docs/getting-started/installation.md)
2. [OpenTelemetry](docs/advanced/opentelemetry.md)
3. [Prometheus](docs/advanced/prometheus.md)
4. [性能优化](docs/advanced/performance.md)

---

## 📖 推荐学习路径

### 入门路径（1-2 天）
```
安装指南 → 5分钟快速开始 → ChatModel → Prompts → 基础示例
```

### 进阶路径（3-5 天）
```
Runnable 系统 → StateGraph → Tools → Agent 系统
```

### 高级路径（1-2 周）
```
RAG 系统 → 向量数据库 → 高级检索 → 监控和优化
```

---

## 🆘 需要帮助？

- **有问题？** 查看 [FAQ](docs/reference/faq.md)（即将添加）
- **发现 Bug？** 提交 [Issue](https://github.com/your-org/langchain-go/issues)
- **想贡献？** 阅读 [贡献指南](docs/development/contributing.md)
- **需要讨论？** 加入 [Discussions](https://github.com/your-org/langchain-go/discussions)

---

## 📂 文档结构总览

```
docs/
├── 📖 getting-started/    # 快速开始（新手必看）
├── 📘 guides/             # 详细指南
│   ├── core/              # 核心功能
│   ├── langgraph/         # LangGraph
│   ├── agents/            # Agent 系统
│   └── rag/               # RAG 系统
├── 💡 examples/           # 代码示例
├── 🔥 advanced/           # 高级主题
├── 📚 api/                # API 参考
├── 👨‍💻 development/       # 开发者文档
├── 📋 reference/          # 参考资料
└── 🗄️ archive/            # 历史归档
```

---

<div align="center">

**[开始使用](docs/getting-started/)** | **[浏览文档](docs/)** | **[查看示例](docs/examples/)**

**祝你使用愉快！** 🎉

</div>

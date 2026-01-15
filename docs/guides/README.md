# 使用指南

欢迎来到 LangChain-Go 使用指南！这里包含了所有核心功能的详细文档。

---

## 📖 指南分类

### [核心功能](./core/)
LangChain 核心组件的使用指南

- [Runnable 系统](./core/runnable.md) - LCEL 链式组合
- [ChatModel 集成](./core/chat-models.md) - OpenAI、Anthropic
- [Prompts 模板](./core/prompts.md) - 提示词工程
- [OutputParser 解析](./core/output-parsers.md) - 结构化输出
- [Tools 工具](./core/tools.md) - 工具定义和使用
- Memory 记忆 - 对话历史管理（即将添加）

### [LangGraph](./langgraph/)
状态图工作流系统

- [StateGraph 状态图](./langgraph/stategraph.md) - 工作流编排
- [Checkpoint 检查点](./langgraph/checkpoint.md) - 状态持久化
- [Durability 持久性](./langgraph/durability.md) - 故障恢复
- HITL 人机协作 - Human-in-the-Loop（即将添加）

### [Agent 系统](./agents/)
智能 Agent 的构建和使用

- [Agent 概述](./agents/overview.md) - Agent 系统介绍
- [Plan-Execute Agent](./agents/plan-execute.md) - 计划执行 Agent
- ReAct Agent - 推理和行动（即将添加）
- 自定义 Agent - 创建自定义 Agent（即将添加）

### [RAG 系统](./rag/)
检索增强生成系统

- [RAG 概述](./rag/overview.md) - RAG 系统介绍
- 文档加载器 - 多格式文档加载（即将添加）
- 文本分割器 - 智能文本分割（即将添加）
- 嵌入模型 - Embedding 集成（即将添加）
- 向量存储 - 向量数据库概述（即将添加）
- [Milvus](./rag/milvus.md) - Milvus 使用指南
- [Milvus Hybrid Search](./rag/milvus-hybrid.md) - 混合搜索
- Chroma - Chroma 向量数据库（即将添加）
- Pinecone - Pinecone 云服务（即将添加）
- [MMR 搜索](./rag/mmr.md) - 最大边际相关性
- [LLM Reranking](./rag/reranking.md) - 智能重排序
- [PDF 加载器](./rag/pdf-loader.md) - PDF 文档处理

---

## 🎯 推荐阅读顺序

### 初学者
1. [Runnable 系统](./core/runnable.md) - 理解核心抽象
2. [ChatModel 集成](./core/chat-models.md) - 学习 LLM 调用
3. [Prompts 模板](./core/prompts.md) - 掌握提示词工程

### 进阶开发者
1. [StateGraph 状态图](./langgraph/stategraph.md) - 构建复杂工作流
2. [Agent 概述](./agents/overview.md) - 创建智能 Agent
3. [RAG 概述](./rag/overview.md) - 实现知识检索

### 生产部署
1. [Checkpoint 检查点](./langgraph/checkpoint.md) - 状态持久化
2. [Durability 持久性](./langgraph/durability.md) - 故障恢复
3. 查看[高级主题](../advanced/)了解监控和优化

---

## 💡 按功能查找

### 我想...

- **调用 LLM** → [ChatModel 指南](./core/chat-models.md)
- **设计提示词** → [Prompts 指南](./core/prompts.md)
- **解析 JSON 输出** → [OutputParser 指南](./core/output-parsers.md)
- **让 Agent 使用工具** → [Tools 指南](./core/tools.md)
- **记住对话历史** → Memory 指南（即将添加）
- **构建工作流** → [StateGraph 指南](./langgraph/stategraph.md)
- **保存执行状态** → [Checkpoint 指南](./langgraph/checkpoint.md)
- **创建 Agent** → [Agent 概述](./agents/overview.md)
- **实现 RAG** → [RAG 概述](./rag/overview.md)
- **处理 PDF 文档** → [PDF 加载器](./rag/pdf-loader.md)
- **搜索相关文档** → [Milvus 指南](./rag/milvus.md)

---

## 📚 相关资源

- [快速开始](../getting-started/) - 新手入门
- [示例代码](../examples/) - 实用示例
- [高级主题](../advanced/) - 生产级功能
- [API 文档](https://pkg.go.dev/langchain-go) - API 参考

---

<div align="center">

**[⬆ 回到文档首页](../README.md)**

</div>

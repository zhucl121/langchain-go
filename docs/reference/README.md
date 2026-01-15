# 参考资料

路线图、FAQ、迁移指南等参考信息。

---

## 📖 文档列表

### 规划和路线图
- [扩展功能清单](./enhancements.md) - 功能规划和完成状态
- 路线图 - 功能规划和时间线（即将添加）

### 帮助和支持
- 常见问题 (FAQ) - 常见问题解答（即将添加）
- 故障排查 - 问题诊断指南（即将添加）
- 迁移指南 - 版本升级指南（即将添加）

---

## 🗺️ 项目路线图

查看[扩展功能清单](./enhancements.md)了解：
- 已完成功能（17项）
- 计划中功能（45+项）
- 优先级和时间规划

### 已完成阶段
- ✅ Phase 1: 基础核心
- ✅ Phase 2: LangGraph 核心
- ✅ Phase 3: Agent 系统
- ✅ Phase 4: RAG 系统
- ✅ Phase 5 Stage 1-4: 扩展增强

### 进行中
- ⏸️ 语义分割器
- ⏸️ Multi-Agent 系统
- ⏸️ API 工具集成

---

## ❓ 常见问题

### 功能相关

**Q: LangChain-Go 与 Python LangChain 的区别？**  
A: LangChain-Go 提供相同的核心功能，但具有更高的性能和更低的资源消耗。

**Q: 支持哪些 LLM 提供商？**  
A: 目前支持 OpenAI 和 Anthropic，更多提供商正在开发中。

**Q: 如何选择向量数据库？**  
A: 
- 本地开发 → Chroma 或 InMemory
- 轻量级生产 → Chroma 或 Milvus
- 大规模生产 → Milvus 或 Pinecone

### 技术相关

**Q: 需要 Go 版本？**  
A: Go 1.22 或更高版本。

**Q: 测试覆盖率如何？**  
A: 平均测试覆盖率 75%+。

**Q: 是否支持流式输出？**  
A: 是的，所有 ChatModel 都支持流式输出。

---

## 📚 相关资源

- [快速开始](../getting-started/) - 新手入门
- [使用指南](../guides/) - 功能文档
- [开发文档](../development/) - 贡献指南

---

<div align="center">

**[⬆ 回到文档首页](../README.md)**

</div>

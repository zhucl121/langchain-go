# 开发文档

为 LangChain-Go 贡献者准备的开发文档。

---

## 📖 文档列表

### 项目信息
- [项目进度](./project-progress.md) - 开发进度跟踪
- 架构设计 - 系统架构说明（即将添加）
- 项目结构 - 代码组织方式（即将添加）

### 贡献指南
- 贡献指南 - 如何贡献代码（参见根目录 CONTRIBUTING.md）
- 测试指南 - 测试规范和实践（即将添加）
- 代码规范 - 编码标准（参见 .cursorrules）
- 发布流程 - 版本发布流程（即将添加）

---

## 🛠️ 开发工作流

### 1. 设置开发环境

```bash
# Clone 项目
git clone https://github.com/yourusername/langchain-go.git
cd langchain-go

# 安装依赖
go mod download

# 运行测试
go test ./...
```

### 2. 开发流程

1. 创建功能分支
2. 编写代码和测试
3. 运行测试确保通过
4. 提交 Pull Request

### 3. 测试规范

- 单元测试覆盖率 > 70%
- 所有测试必须通过
- 添加必要的集成测试

---

## 📊 项目统计

查看[项目进度](./project-progress.md)了解：
- 模块完成情况
- 代码统计
- 测试覆盖率
- 开发里程碑

---

## 🔗 相关资源

- [贡献指南](../../CONTRIBUTING.md) - 如何贡献
- [行为准则](../../CODE_OF_CONDUCT.md) - 社区规范
- [安全政策](../../SECURITY.md) - 安全问题报告

---

<div align="center">

**[⬆ 回到文档首页](../README.md)**

</div>

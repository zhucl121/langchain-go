# 剩余问题清单

## 🔴 严重问题（阻塞发布）

### 1. core/agents 包编译失败

**影响范围：** 
- agents 包完全不可用
- 所有示例文件无法编译运行
- 所有依赖 agents 的功能无法使用

**错误列表：**

#### 错误 1: types.Function 未定义
```
core/agents/openai_functions.go:149:68: undefined: types.Function
core/agents/openai_functions.go:206:20: undefined: types.Function
```
**需要：** 在 pkg/types 中定义 Function 类型或修复引用

#### 错误 2: chat 包未导入
```
core/agents/selfask.go:279:29: undefined: chat
core/agents/structured_chat.go:266:36: undefined: chat
core/agents/multi_agent.go:748:21: undefined: chat.Message
core/agents/multi_agent.go:748:34: undefined: chat.NewHumanMessage
```
**需要：** 添加正确的 import "langchain-go/core/chat"

#### 错误 3: tools 包未导入
```
core/agents/selfask.go:279:56: undefined: tools
core/agents/structured_chat.go:266:65: undefined: tools
```
**需要：** 添加正确的 import "langchain-go/core/tools"

#### 错误 4: 类型不匹配
```
core/agents/factory.go:36:17: cannot use templates.ReActPrompt (variable of type *prompts.PromptTemplate) as string value in struct literal
core/agents/factory.go:74:17: cannot use templates.ToolCallingPrompt (variable of type *prompts.PromptTemplate) as string value in struct literal
```
**需要：** 修改 factory.go 中的结构体定义，或调用 templates 的 String() 方法

---

## ⚠️ 中等问题（影响质量）

### 2. 测试覆盖率严重不足

**当前状态：**
- 原始测试: 451 行
- 简化后: 41 行
- **损失: 91% 的测试用例**

**缺失的测试：**
- ❌ CalculatorTool 完整测试（表达式计算、错误处理）
- ❌ JSON 工具测试套件（Parse, Stringify, Extract）
- ❌ String 工具测试套件（Length, Split, Join）
- ❌ 工具注册表功能测试
- ❌ 工具参数验证测试
- ❌ 工具执行错误处理测试
- ❌ 示例代码和文档

**建议：** 
1. 恢复 tools_test.go.bak
2. 逐个修复测试中引用的不存在工具
3. 用现有工具替换或删除个别测试，而不是整个文件

---

## 🟡 次要问题（不影响核心功能）

### 3. retrieval/loaders 测试编译错误
```
retrieval/loaders/loader_test.go:195:6: TestCSVLoader redeclared
retrieval/loaders/docx_test.go:184:15: undefined: NewCharacterTextSplitter
```
**状态：** loaders 包本身可以编译，只是测试有问题

### 4. 部分示例文件可能不完整
- 所有示例都依赖 agents 包，目前无法验证是否完整

---

## ✅ 已修复的问题

- ✅ HTTPRequestTool 重复声明
- ✅ 工具 GetParameters() API 更新为 types.Schema
- ✅ 示例文件 import 路径（ollama -> openai）
- ✅ registry.go 中的工具引用
- ✅ Multimodal 工具实现
- ✅ Go 版本设置

---

## 📋 修复优先级

### P0 - 必须修复（阻塞发布）
1. 修复 core/agents 编译错误
2. 验证所有示例可以编译运行

### P1 - 强烈建议修复（影响质量）
3. 恢复完整的测试覆盖率
4. 修复 retrieval 测试错误

### P2 - 可选修复
5. 优化文档和示例

---

## 🚀 快速修复建议

### 选项 A: 完整修复（推荐）
1. 修复 agents 包的所有编译错误（约30分钟）
2. 恢复并修复测试文件（约20分钟）
3. 验证所有功能（约10分钟）

### 选项 B: 最小可用版本
1. 仅修复 agents 包编译错误
2. 保持简化的测试
3. 在 README 中注明功能限制

### 选项 C: 暂时跳过 agents
1. 从构建中排除 agents 包
2. 在文档中标注 agents 功能暂不可用
3. 后续单独修复

---

## 💡 建议

**我的建议是选择「选项 A: 完整修复」**，因为：

1. **agents 是核心功能** - 7/11 个示例都依赖它
2. **修复时间可控** - 预计1小时内完成
3. **测试很重要** - 91%的测试缺失会导致未来难以维护
4. **一次性解决** - 避免技术债累积

**是否继续修复？**

# 贡献指南

🌍 **语言**: 中文 | [English](CONTRIBUTING.md)

首先，感谢您考虑为 LangChain-Go 做出贡献！正是像您这样的人让 LangChain-Go 成为一个优秀的工具。

## 行为准则

本项目及其所有参与者均受我们的行为准则约束。通过参与，您应遵守此准则。请向 [your.email@example.com] 报告不可接受的行为。

## 如何贡献？

### 报告 Bug

在创建 bug 报告之前，请检查问题列表，因为您可能会发现不需要创建新的报告。创建 bug 报告时，请尽可能包含详细信息：

* **使用清晰且描述性的标题**
* **描述重现问题的确切步骤**
* **提供具体示例来演示这些步骤**
* **描述执行步骤后观察到的行为**
* **解释您期望看到什么行为以及原因**
* **包含 Go 版本、操作系统和相关环境详细信息**

**Bug 报告模板：**

```markdown
## Bug 描述
清晰简洁的 bug 描述。

## 重现步骤
重现该行为的步骤：
1. 转到 '...'
2. 运行 '...'
3. 看到错误

## 预期行为
您期望发生什么。

## 实际行为
实际发生了什么。

## 环境
- Go 版本: [例如 1.22.0]
- 操作系统: [例如 macOS 14.0]
- LangChain-Go 版本: [例如 1.3.0]

## 其他信息
在此添加关于问题的任何其他信息。
```

### 建议增强功能

增强功能建议作为 GitHub issues 跟踪。创建增强功能建议时，请包括：

* **使用清晰且描述性的标题**
* **提供建议增强功能的分步描述**
* **提供具体示例来演示这些步骤**
* **描述当前行为并解释预期行为**
* **解释此增强功能为何有用**

**功能请求模板：**

```markdown
## 功能描述
清晰简洁地描述您希望发生的事情。

## 使用场景
描述您试图解决的问题。

## 建议的解决方案
描述您设想的功能如何工作。

## 考虑的替代方案
描述您考虑过的替代方案。

## 其他信息
在此添加关于功能请求的任何其他信息或截图。
```

### 拉取请求

* 填写所需的模板
* 遵循 Go 编码风格
* 包含经过深思熟虑、结构良好的测试
* 记录新代码
* 所有文件以换行符结尾

## 开发流程

### 设置开发环境

1. **Fork 仓库**

```bash
# 在 GitHub 上 Fork，然后克隆您的 fork
git clone https://github.com/YOUR_USERNAME/langchain-go.git
cd langchain-go
```

2. **安装依赖**

```bash
go mod download
```

3. **创建分支**

```bash
git checkout -b feature/my-new-feature
# 或
git checkout -b fix/issue-123
```

### 编码指南

#### Go 风格指南

我们遵循官方的 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) 和 [Effective Go](https://golang.org/doc/effective_go)。

**要点：**

1. **格式化**：使用 `gofmt` 或 `go fmt`
   ```bash
   go fmt ./...
   ```

2. **代码检查**：使用 `golangci-lint`
   ```bash
   golangci-lint run
   ```

3. **命名约定**：
   - 导出名称使用 MixedCaps
   - 未导出名称使用 camelCase
   - 保持名称简短且有意义
   - 避免口吃（例如，`http.HTTPServer` → `http.Server`）

4. **文档**：
   - 每个导出的函数、类型和包都必须有文档注释
   - 文档注释以被描述事物的名称开头
   - 使用完整的句子

   ```go
   // NewClient 使用给定配置创建一个新的 OpenAI 客户端。
   //
   // 该客户端对于多个 goroutine 的并发使用是安全的。
   func NewClient(config Config) (*Client, error) {
       // ...
   }
   ```

5. **错误处理**：
   - 返回错误，不要 panic（除非真正的异常情况）
   - 使用 `fmt.Errorf` 和 `%w` 进行错误包装
   - 在错误消息中提供上下文

   ```go
   if err != nil {
       return fmt.Errorf("加载文档失败: %w", err)
   }
   ```

6. **测试**：
   - 为所有新代码编写测试
   - 在适当的地方使用表驱动测试
   - 使用有意义的测试名称

   ```go
   func TestVectorStore_Search(t *testing.T) {
       tests := []struct {
           name    string
           query   string
           k       int
           want    int
           wantErr bool
       }{
           {
               name:  "基础搜索",
               query: "测试查询",
               k:     5,
               want:  5,
           },
           // ... 更多测试用例
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // 测试实现
           })
       }
   }
   ```

#### 项目特定指南

1. **阅读 `.cursorrules`**：我们详细的编码标准记录在 [.cursorrules](.cursorrules) 中

2. **适当使用泛型**：LangChain-Go 使用 Go 泛型提供类型安全
   ```go
   type Runnable[In, Out any] interface {
       Invoke(ctx context.Context, input In, opts ...Option) (Out, error)
   }
   ```

3. **Context 处理**：始终接受 `context.Context` 作为第一个参数
   ```go
   func Process(ctx context.Context, input string) error {
       // ...
   }
   ```

4. **并发**：明智地使用 goroutines 和 channels
   - 记录并发行为
   - 使用 `defer` 确保正确清理
   - 对共享状态使用 `sync.Mutex` 或 `sync.RWMutex`

5. **包组织**：
   - 保持包的专注和内聚
   - 避免循环依赖
   - 对实现细节使用内部包

### 测试

#### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行特定包的测试
go test ./core/chat/...

# 使用竞态检测器运行测试
go test -race ./...

# 运行基准测试
go test -bench=. ./...
```

#### 编写测试

1. **单元测试**：测试单个函数和方法
2. **集成测试**：测试组件之间的交互
3. **基准测试**：测量性能

```go
// 示例单元测试
func TestCalculator_Evaluate(t *testing.T) {
    calc := NewCalculator()
    result, err := calc.Evaluate("2 + 2")
    
    assert.NoError(t, err)
    assert.Equal(t, 4.0, result)
}

// 示例基准测试
func BenchmarkCalculator_Evaluate(b *testing.B) {
    calc := NewCalculator()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = calc.Evaluate("2 + 2 * 3")
    }
}
```

#### 测试覆盖率

- 新代码至少达到 70% 的覆盖率
- 关键路径应该有 90%+ 的覆盖率
- 不要为了覆盖率数字而牺牲测试质量

### 文档

#### 代码文档

- 对所有公共 API 使用 GoDoc 格式
- 在有帮助的情况下在文档注释中包含示例
- 记录并发行为
- 如有任何 panic，记录它们

```go
// Process 处理输入并返回结果。
//
// 此函数对并发使用是安全的。
//
// 示例：
//
//	result, err := Process(ctx, "input")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result)
func Process(ctx context.Context, input string) (string, error) {
    // ...
}
```

#### Markdown 文档

- 保持文档与代码更改同步
- 使用清晰、简洁的语言
- 包括代码示例
- 在有帮助的地方添加图表

### 提交消息

我们遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型：**
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更改
- `style`: 代码风格更改（格式化等）
- `refactor`: 代码重构
- `perf`: 性能改进
- `test`: 测试添加或更改
- `chore`: 构建过程或辅助工具更改

**示例：**

```
feat(vectorstore): 添加 Milvus 混合搜索支持

实现结合向量相似度和 BM25 关键词搜索的混合搜索，
用于 Milvus 2.6+。支持 RRF 和加权重排序。

Closes #123
```

```
fix(agent): 防止 ReAct agent 无限循环

添加最大步数检查以防止当 agent 无法得出结论时的无限循环。

Fixes #456
```

### 拉取请求流程

1. **更新文档**：更新 README.md、CHANGELOG.md 和相关文档
2. **添加测试**：确保您的 PR 包含新功能的测试
3. **运行测试**：确保所有测试通过
4. **更新 CHANGELOG**：在 [CHANGELOG.md](CHANGELOG.md) 中添加条目
5. **填写 PR 模板**：使用我们的 PR 模板

**PR 模板：**

```markdown
## 描述
简要描述更改。

## 更改类型
- [ ] Bug 修复（不会导致现有功能无法正常工作的非破坏性更改）
- [ ] 新功能（添加功能的非破坏性更改）
- [ ] 破坏性更改（会导致现有功能无法按预期工作的修复或功能）
- [ ] 文档更新

## 如何测试？
描述您运行的测试以及如何重现它们。

## 检查清单
- [ ] 我的代码遵循此项目的风格指南
- [ ] 我已对自己的代码进行了自我审查
- [ ] 我已对代码进行了注释，特别是在难以理解的地方
- [ ] 我已对文档进行了相应的更改
- [ ] 我的更改不会产生新的警告
- [ ] 我已添加证明我的修复有效或我的功能有效的测试
- [ ] 新的和现有的单元测试在本地通过我的更改
- [ ] 任何依赖的更改已被合并和发布
```

### 审查流程

1. **自动检查**：CI/CD 将运行测试和代码检查器
2. **同行审查**：至少一位维护者将审查您的 PR
3. **反馈**：处理审查意见
4. **批准**：一旦批准，您的 PR 将被合并

## 社区

### 获取帮助

- **GitHub Discussions**：提问和讨论想法
- **GitHub Issues**：报告 bug 和请求功能
- **文档**：查看我们的 [docs](./docs) 文件夹和 [QUICK_START.md](./QUICK_START.md)

### 认可

贡献者将在以下位置获得认可：
- [AUTHORS](AUTHORS) 文件
- 发布说明
- 项目文档

## 许可证

通过为 LangChain-Go 做出贡献，您同意您的贡献将根据 MIT 许可证进行许可。

---

感谢您为 LangChain-Go 做出贡献！🎉

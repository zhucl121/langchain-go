# 安装指南

本指南将帮助你安装和配置 LangChain-Go。

---

## 📋 环境要求

### 必需
- **Go**: 1.22 或更高版本
- **操作系统**: Linux、macOS 或 Windows

### 可选（根据使用的功能）
- **Docker**: 用于运行 Milvus、Chroma 等向量数据库
- **PostgreSQL**: 用于 Checkpoint 持久化
- **SQLite**: 用于 Checkpoint 持久化

---

## 🚀 快速安装

### 1. 安装 Go

如果还没有安装 Go，请按照以下步骤：

#### macOS
```bash
brew install go
```

#### Linux
```bash
# 下载 Go
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz

# 解压
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# 添加到 PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### Windows
从 [https://go.dev/dl/](https://go.dev/dl/) 下载安装包并安装。

### 2. 验证 Go 安装

```bash
go version
# 应该输出: go version go1.22.0 或更高
```

### 3. 安装 LangChain-Go

```bash
go get github.com/yourusername/langchain-go
```

---

## 🔧 配置

### API Keys

LangChain-Go 支持多个 LLM 提供商，你需要获取相应的 API Key：

#### OpenAI
1. 访问 [https://platform.openai.com/api-keys](https://platform.openai.com/api-keys)
2. 创建新的 API Key
3. 设置环境变量：
```bash
export OPENAI_API_KEY="sk-..."
```

#### Anthropic
1. 访问 [https://console.anthropic.com/](https://console.anthropic.com/)
2. 创建 API Key
3. 设置环境变量：
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

### 向量数据库（可选）

#### Milvus
```bash
# 使用 Docker 运行
docker run -d --name milvus \
  -p 19530:19530 \
  -p 9091:9091 \
  milvusdb/milvus:latest
```

#### Chroma
```bash
# 使用 Docker 运行
docker run -d --name chroma \
  -p 8000:8000 \
  chromadb/chroma:latest
```

#### Pinecone
1. 访问 [https://www.pinecone.io/](https://www.pinecone.io/)
2. 注册并创建 API Key
3. 设置环境变量：
```bash
export PINECONE_API_KEY="your-key"
```

### 数据库（用于 Checkpoint）

#### PostgreSQL
```bash
# macOS
brew install postgresql
brew services start postgresql

# Linux (Ubuntu/Debian)
sudo apt-get install postgresql
sudo systemctl start postgresql
```

#### SQLite
SQLite 通常已预装，或通过包管理器安装：
```bash
# macOS
brew install sqlite

# Linux
sudo apt-get install sqlite3
```

---

## 📦 依赖管理

### 初始化项目

```bash
# 创建新项目
mkdir my-langchain-app
cd my-langchain-app

# 初始化 Go module
go mod init my-langchain-app

# 安装 LangChain-Go
go get github.com/yourusername/langchain-go
```

### 可选依赖

根据使用的功能，安装相应的依赖：

```bash
# Milvus
go get github.com/milvus-io/milvus-sdk-go/v2

# Chroma
go get github.com/amikos-tech/chroma-go

# Pinecone
go get github.com/pinecone-io/go-pinecone

# PostgreSQL
go get github.com/lib/pq

# SQLite
go get github.com/mattn/go-sqlite3

# OpenTelemetry
go get go.opentelemetry.io/otel

# Prometheus
go get github.com/prometheus/client_golang
```

---

## ✅ 验证安装

创建一个简单的测试文件：

```go
// main.go
package main

import (
    "context"
    "fmt"
    "os"
    
    "langchain-go/core/chat/providers/openai"
    "langchain-go/pkg/types"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        fmt.Println("请设置 OPENAI_API_KEY 环境变量")
        return
    }
    
    model := openai.New(openai.Config{
        APIKey: apiKey,
        Model:  "gpt-3.5-turbo",
    })
    
    response, err := model.Invoke(context.Background(), []types.Message{
        types.NewUserMessage("Say hello!"),
    })
    
    if err != nil {
        fmt.Printf("错误: %v\n", err)
        return
    }
    
    fmt.Println("LangChain-Go 安装成功！")
    fmt.Println("回复:", response.Content)
}
```

运行测试：

```bash
go run main.go
```

如果看到 "LangChain-Go 安装成功！" 和 LLM 的回复，说明安装成功！

---

## 🆘 常见问题

### Q: 提示 "cannot find package"
**A**: 运行 `go mod tidy` 确保所有依赖已下载。

### Q: OpenAI API 调用失败
**A**: 
1. 检查 API Key 是否正确
2. 检查网络连接
3. 确认账户有足够的额度

### Q: 向量数据库连接失败
**A**: 
1. 确保 Docker 容器正在运行
2. 检查端口是否被占用
3. 验证连接配置（主机、端口）

### Q: Go 版本过低
**A**: 升级到 Go 1.22+
```bash
# macOS
brew upgrade go

# Linux - 下载新版本
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
```

---

## 🎓 学习资源

- [Go 官方教程](https://go.dev/tour/)
- [Go 语言规范](https://go.dev/ref/spec)
- [Effective Go](https://go.dev/doc/effective_go)

---

## ➡️ 下一步

安装完成后，继续学习：

1. [5分钟快速开始](./quickstart.md) - 开始使用 LangChain-Go
2. [ChatModel 快速开始](./quickstart-chat.md) - 学习对话模型
3. [使用指南](../guides/) - 深入了解各个功能

---

<div align="center">

**[⬆ 回到快速开始](./README.md)** | **[回到文档首页](../README.md)**

</div>

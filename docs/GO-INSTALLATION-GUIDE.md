# Go 环境安装指南 (macOS ARM64)

## 系统信息
- **操作系统**: macOS (Darwin 25.1.0)
- **架构**: ARM64 (Apple Silicon)
- **Shell**: zsh
- **状态**: Go 未安装 ❌

---

## 方法一：使用 Homebrew 安装 (推荐) ⭐

### 1. 检查是否已安装 Homebrew

```bash
which brew
```

如果未安装，先安装 Homebrew：

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 2. 使用 Homebrew 安装 Go

```bash
# 安装最新稳定版 Go
brew install go

# 或指定版本（如 1.22）
brew install go@1.22
```

### 3. 验证安装

```bash
go version
# 应该显示：go version go1.22.x darwin/arm64
```

### 4. 配置环境变量（Homebrew 通常会自动配置）

检查配置：
```bash
go env GOPATH
go env GOROOT
```

---

## 方法二：手动下载安装包

### 1. 下载 Go

访问官方下载页面：
https://go.dev/dl/

或直接下载 ARM64 版本：
```bash
# 下载最新版本（以 1.22.9 为例）
curl -O https://go.dev/dl/go1.22.9.darwin-arm64.pkg
```

### 2. 安装

双击下载的 `.pkg` 文件，按照安装向导操作。

或使用命令行：
```bash
sudo installer -pkg go1.22.9.darwin-arm64.pkg -target /
```

### 3. 配置环境变量

编辑 `~/.zshrc`：
```bash
vim ~/.zshrc
```

添加以下内容：
```bash
# Go 环境变量
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

重新加载配置：
```bash
source ~/.zshrc
```

### 4. 验证安装

```bash
go version
go env
```

---

## 方法三：使用版本管理工具 (适合多版本管理)

### 使用 gvm (Go Version Manager)

```bash
# 安装 gvm
bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)

# 重新加载
source ~/.gvm/scripts/gvm

# 安装 Go 1.22
gvm install go1.22 -B

# 使用指定版本
gvm use go1.22 --default
```

---

## 安装后配置

### 1. 设置 Go 模块代理（加速国内下载）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
```

或添加到 `~/.zshrc`：
```bash
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
```

### 2. 创建工作目录

```bash
mkdir -p $HOME/go/{bin,src,pkg}
```

### 3. 配置 VS Code / Cursor

安装 Go 扩展后，会自动提示安装工具：
- gopls (语言服务器)
- dlv (调试器)
- staticcheck (静态检查)

或手动安装：
```bash
go install golang.org/x/tools/gopls@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
```

---

## 验证安装完成

### 运行完整检查

```bash
# 1. 检查 Go 版本
go version

# 2. 检查 Go 环境
go env

# 3. 创建测试项目
cd /tmp
mkdir hello-go && cd hello-go
go mod init example.com/hello
echo 'package main

import "fmt"

func main() {
    fmt.Println("Hello, Go!")
}' > main.go

# 4. 运行测试
go run main.go

# 5. 编译
go build

# 6. 运行编译后的二进制
./hello

# 清理
cd .. && rm -rf hello-go
```

---

## 为 langchain-go 项目配置

安装完成后，回到项目目录：

```bash
cd /Users/zhuchenglong/Documents/workspace/随笔/langchain-go

# 1. 下载依赖
go mod download

# 2. 整理依赖
go mod tidy

# 3. 运行测试
go test ./...

# 4. 查看覆盖率
go test -cover ./...

# 5. 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 6. 运行基准测试
go test -bench=. ./...

# 7. 检查代码
go vet ./...

# 8. 格式化代码
go fmt ./...
```

---

## 推荐的开发工具

### 1. golangci-lint (代码检查)

```bash
# 使用 Homebrew 安装
brew install golangci-lint

# 或使用官方脚本
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 运行检查
cd /Users/zhuchenglong/Documents/workspace/随笔/langchain-go
golangci-lint run
```

### 2. goimports (自动导入管理)

```bash
go install golang.org/x/tools/cmd/goimports@latest

# 使用
goimports -w .
```

### 3. air (热重载)

```bash
go install github.com/air-verse/air@latest

# 创建配置文件
air init

# 运行
air
```

---

## 常见问题

### Q1: 提示 "command not found: go"

**解决**：环境变量未配置，按照上面步骤配置 PATH。

### Q2: go get 下载慢

**解决**：配置 GOPROXY 代理（见上面配置部分）。

### Q3: M1/M2 芯片兼容性问题

**解决**：确保下载 darwin-arm64 版本，不要用 amd64 版本。

### Q4: 权限问题

**解决**：
```bash
sudo chown -R $(whoami) /usr/local/go
sudo chown -R $(whoami) ~/go
```

---

## 快速安装命令（推荐）

如果你已有 Homebrew，直接运行：

```bash
# 一键安装和配置
brew install go && \
go env -w GOPROXY=https://goproxy.cn,direct && \
go env -w GOSUMDB=sum.golang.google.cn && \
mkdir -p $HOME/go/{bin,src,pkg} && \
echo "✅ Go 安装完成！" && \
go version
```

---

## 安装完成后

运行以下命令测试 langchain-go 项目：

```bash
cd /Users/zhuchenglong/Documents/workspace/随笔/langchain-go

# 测试所有包
make test

# 或直接用 go
go test -v ./pkg/types/
```

---

## 需要帮助？

安装完成后告诉我，我会帮你：
1. ✅ 验证安装
2. ✅ 运行项目测试
3. ✅ 继续实现下一个模块

---

*文档更新时间：2026-01-14*

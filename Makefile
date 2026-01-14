.PHONY: help build test lint clean install run-examples

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
GO := go
GOFLAGS := -v
BINARY_NAME := langchain-go
BUILD_DIR := ./bin
COVERAGE_FILE := coverage.out

## help: 显示帮助信息
help:
	@echo "LangChain-Go Makefile Commands:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: 编译项目
build:
	@echo "Building..."
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/...

## test: 运行所有测试
test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) ./...

## test-coverage: 运行测试并生成覆盖率报告
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -cover -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-verbose: 运行详细测试
test-verbose:
	@echo "Running verbose tests..."
	$(GO) test -v ./...

## test-short: 运行快速测试（跳过长时间测试）
test-short:
	@echo "Running short tests..."
	$(GO) test -short ./...

## bench: 运行基准测试
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

## lint: 运行代码检查
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin"; exit 1)
	golangci-lint run ./...

## fmt: 格式化代码
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	goimports -w .

## vet: 运行 go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## tidy: 整理依赖
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

## download: 下载依赖
download:
	@echo "Downloading dependencies..."
	$(GO) mod download

## clean: 清理构建文件
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) coverage.html
	$(GO) clean

## install: 安装项目
install:
	@echo "Installing..."
	$(GO) install ./...

## run-examples: 运行示例代码
run-examples:
	@echo "Running examples..."
	$(GO) run ./examples/simple_chat/main.go

## init-db: 初始化数据库（用于测试 checkpointing）
init-db:
	@echo "Initializing test database..."
	# PostgreSQL
	-psql -U postgres -c "CREATE DATABASE langchain_test;"
	# SQLite
	-rm -f test.db

## check: 运行所有检查（fmt, vet, lint, test）
check: fmt vet lint test

## pre-commit: 提交前检查
pre-commit: fmt vet lint test-short

## ci: CI 流程
ci: fmt vet lint test-coverage

## docs: 生成文档
docs:
	@echo "Generating documentation..."
	$(GO) doc -all

## upgrade-deps: 升级依赖到最新版本
upgrade-deps:
	@echo "Upgrading dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

## version: 显示版本信息
version:
	@echo "Go version:"
	@$(GO) version
	@echo ""
	@echo "Project version: 1.0.0-dev"

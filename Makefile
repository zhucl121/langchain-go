# Makefile for LangChain-Go

.PHONY: help test test-cover test-race lint fmt vet build clean deps check-deps bench

# Variables
GO := go
GOFLAGS :=
LDFLAGS :=
PACKAGES := $(shell $(GO) list ./...)
GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "LangChain-Go Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v $(PACKAGES)

## test-short: Run tests excluding integration tests
test-short:
	@echo "Running short tests..."
	$(GO) test -short -v $(PACKAGES)

## test-cover: Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out -covermode=atomic $(PACKAGES)
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GO) test -race $(PACKAGES)

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem $(PACKAGES)

## lint: Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found. Install it from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run --timeout=5m

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt $(PACKAGES)

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet $(PACKAGES)

## build: Build the project
build:
	@echo "Building..."
	$(GO) build $(GOFLAGS) $(PACKAGES)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GO) clean
	rm -f coverage.out coverage.html

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

## deps-verify: Verify dependencies
deps-verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify

## tidy: Tidy go.mod
tidy:
	@echo "Tidying go.mod..."
	$(GO) mod tidy

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

## ci: Run CI checks
ci: check test-race test-cover

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest

## docs: Generate documentation
docs:
	@echo "Generating documentation..."
	@which godoc > /dev/null 2>&1 || $(GO) install golang.org/x/tools/cmd/godoc@latest
	@echo "Documentation available at http://localhost:6060"
	godoc -http=:6060

## example: Run example
example:
	@echo "Running example..."
	$(GO) run examples/quickstart/main.go

## milvus-up: Start Milvus Docker container
milvus-up:
	@echo "Starting Milvus..."
	docker run -d --name milvus -p 19530:19530 -p 9091:9091 milvusdb/milvus:v2.6.0

## milvus-down: Stop Milvus Docker container
milvus-down:
	@echo "Stopping Milvus..."
	docker stop milvus
	docker rm milvus

## docker-test: Run tests in Docker
docker-test:
	@echo "Running tests in Docker..."
	docker run --rm -v $(PWD):/app -w /app golang:1.22 make test

## version: Display Go version
version:
	@$(GO) version

## info: Display project information
info:
	@echo "LangChain-Go Project Information"
	@echo "================================"
	@echo "Go Version:     $$(go version)"
	@echo "Packages:       $$(echo $(PACKAGES) | wc -w)"
	@echo "Go Files:       $$(echo $(GOFILES) | wc -w)"
	@echo ""
	@echo "Modules:"
	@$(GO) list -m all | head -20

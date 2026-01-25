# Testing Guide

🌍 **Language**: [中文](TESTING.md) | English

Complete testing guide for LangChain-Go to help you quickly configure the test environment and run tests.

---

## 🚀 Quick Start

### Prerequisites

- ✅ Docker Desktop (installed and running)
- ✅ Go 1.21+ (recommended Go 1.22+)
- ✅ At least 2GB available disk space

### One-Command Testing

```bash
# 1. Start test environment (Redis + Milvus)
make -f Makefile.test test-env-up

# 2. Run all tests
make -f Makefile.test test

# 3. Stop test environment
make -f Makefile.test test-env-down
```

---

## 📦 Test Environment Services

Available services after startup:

| Service | Address | Credentials | Purpose |
|---------|---------|-------------|---------|
| Redis | localhost:6379 | Password: redis123 | Cache testing |
| Milvus 2.6.1 | localhost:19530 | None | Vector store testing |

---

## 🔧 Common Test Commands

### Environment Management

```bash
# View all available commands
make -f Makefile.test help

# Start test environment
make -f Makefile.test test-env-up

# Stop test environment
make -f Makefile.test test-env-down

# Check service status
make -f Makefile.test test-env-status

# View service logs
docker compose -f docker-compose.test.yml logs -f
```

### Running Tests

```bash
# Run all tests
make -f Makefile.test test

# Run all tests (verbose output)
make -f Makefile.test test-verbose

# Run Redis tests only
make -f Makefile.test test-redis

# Run Milvus tests only
make -f Makefile.test test-milvus

# Generate coverage report
make -f Makefile.test test-coverage
```

### Specific Package Tests

```bash
# Test specific package
go test ./core/cache -v
go test ./retrieval/vectorstores -v

# Test specific function
go test ./core/cache -v -run TestRedisCache_Set
go test ./retrieval/vectorstores -v -run TestMilvusVectorStore

# Run benchmarks
go test ./core/cache -bench=. -benchmem
```

---

## 🧪 Test Configuration

### Redis Configuration

Tests use the following Redis configuration:

```go
config := cache.DefaultRedisCacheConfig()
config.Password = "redis123"  // Docker container password
cache, err := cache.NewRedisCache(config)
```

### Milvus Configuration

Tests use the following Milvus configuration:

```go
config := MilvusConfig{
    Address:              "localhost:19530",
    CollectionName:       "test_collection",
    Dimension:            1536,
    AutoCreateCollection: true,
}
store, err := NewMilvusVectorStore(config, embeddings)
```

---

## ❓ Common Issues

### 1. Docker Not Running

**Error**: `Cannot connect to the Docker daemon`

**Solution**:
- macOS/Windows: Start Docker Desktop
- Linux: `sudo systemctl start docker`

### 2. Port Already in Use

**Error**: `port is already allocated`

**Solution**:

```bash
# Check process using port
lsof -i :6379   # Redis
lsof -i :19530  # Milvus

# Stop conflicting containers
docker ps | grep redis
docker stop <container_id>

# Or use auto-fix script
bash scripts/fix-port-conflict.sh
```

### 3. Milvus Slow Startup

**Normal**: Milvus first startup takes 1-2 minutes

**Check status**:

```bash
# View Milvus logs
docker compose -f docker-compose.test.yml logs -f milvus

# Check health status
curl http://localhost:9091/healthz

# If timeout, restart container
docker compose -f docker-compose.test.yml restart milvus
```

### 4. Tests Skipped

**Symptom**: `t.Skip("Redis not available")`

**Reason**: Tests automatically skip when service is not running

**Solution**: Start test environment first

```bash
make -f Makefile.test test-env-up
```

### 5. Go Version Too Old

**Error**: `go: cannot find main module`

**Solution**: Upgrade to Go 1.21+

```bash
# Download latest Go: https://go.dev/dl/
# Verify version
go version
```

---

## 📊 Test Coverage

### Generate Coverage Report

```bash
# Generate coverage report
make -f Makefile.test test-coverage

# View in browser
open coverage.html
```

### Current Coverage

- **Overall Coverage**: 60%+
- **Core Packages**: 70%+
- **Test Packages**: 35+
- **Test Cases**: 500+

---

## 🔥 Performance Testing

Run benchmarks:

```bash
# Redis performance tests
go test ./core/cache -bench=BenchmarkRedisCache -benchmem -benchtime=10s

# Milvus performance tests
go test ./retrieval/vectorstores -bench=. -benchmem

# All benchmarks
go test ./... -bench=. -benchmem
```

---

## 🧹 Cleanup

```bash
# Stop services but keep data
make -f Makefile.test test-env-down

# Stop services and delete data volumes
docker compose -f docker-compose.test.yml down -v

# Complete cleanup (including images)
docker compose -f docker-compose.test.yml down -v --rmi all
```

---

## 💡 Best Practices

### 1. Daily Development Workflow

```bash
# Start work in the morning
make -f Makefile.test test-env-up

# Run tests frequently during development
make -f Makefile.test test

# Stop services before leaving
make -f Makefile.test test-env-down
```

### 2. Test-Driven Development

```bash
# Start services and keep running
make -f Makefile.test test-env-up

# Auto-test on file changes (requires entr)
ls **/*.go | entr -c go test ./... -v
```

### 3. Debug Failing Tests

```bash
# View service logs
docker compose -f docker-compose.test.yml logs

# Run single test
go test ./core/cache -v -run TestRedisCache_Set

# Enter container for debugging
docker exec -it langchain-go-redis redis-cli -a redis123
```

---

## 🎯 CI/CD Integration

Use in CI environment:

```yaml
# .github/workflows/test.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Start test environment
        run: make -f Makefile.test test-env-up
        
      - name: Run tests
        run: make -f Makefile.test test
        
      - name: Stop test environment
        if: always()
        run: make -f Makefile.test test-env-down
```

---

## 📚 Related Documentation

- [Contributing Guide](CONTRIBUTING.md) - Code contribution guidelines
- [Quick Start](docs/getting-started/quickstart.md) - Project quick start
- [Development Docs](docs/development/) - Developer documentation

---

## 🆘 Get Help

Having issues?

1. **Check help**: `make -f Makefile.test help`
2. **Check logs**: `docker compose -f docker-compose.test.yml logs`
3. **Verify environment**: `bash scripts/verify-setup.sh`
4. **Submit Issue**: [GitHub Issues](https://github.com/zhucl121/langchain-go/issues)

---

## ✅ Verify Installation

Run verification script to ensure everything is working:

```bash
bash scripts/verify-setup.sh
```

Expected output:

```
✅ Docker is running
✅ Found docker-compose
✅ docker-compose.test.yml exists
✅ Port 6379 (Redis) available
✅ Port 19530 (Milvus) available
✅ Verification complete!
```

---

**Happy testing! 🎉**

If you have any issues, please refer to the FAQ above or submit an issue.

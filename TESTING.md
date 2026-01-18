# 测试指南

LangChain-Go 的完整测试指南,帮助您快速配置测试环境并运行测试。

---

## 🚀 快速开始

### 前置要求

- ✅ Docker Desktop (已安装并运行)
- ✅ Go 1.21+ (推荐 Go 1.22+)
- ✅ 至少 2GB 可用磁盘空间

### 一键测试

```bash
# 1. 启动测试环境 (Redis + Milvus)
make -f Makefile.test test-env-up

# 2. 运行所有测试
make -f Makefile.test test

# 3. 停止测试环境
make -f Makefile.test test-env-down
```

---

## 📦 测试环境服务

启动后可用的服务:

| 服务 | 地址 | 凭证 | 用途 |
|------|------|------|------|
| Redis | localhost:6379 | 密码: redis123 | 缓存测试 |
| Milvus 2.6.1 | localhost:19530 | 无 | 向量存储测试 |

---

## 🔧 常用测试命令

### 环境管理

```bash
# 查看所有可用命令
make -f Makefile.test help

# 启动测试环境
make -f Makefile.test test-env-up

# 停止测试环境
make -f Makefile.test test-env-down

# 查看服务状态
make -f Makefile.test test-env-status

# 查看服务日志
docker compose -f docker-compose.test.yml logs -f
```

### 运行测试

```bash
# 运行所有测试
make -f Makefile.test test

# 运行所有测试 (详细输出)
make -f Makefile.test test-verbose

# 仅运行 Redis 测试
make -f Makefile.test test-redis

# 仅运行 Milvus 测试
make -f Makefile.test test-milvus

# 生成覆盖率报告
make -f Makefile.test test-coverage
```

### 特定包测试

```bash
# 测试特定包
go test ./core/cache -v
go test ./retrieval/vectorstores -v

# 测试特定函数
go test ./core/cache -v -run TestRedisCache_Set
go test ./retrieval/vectorstores -v -run TestMilvusVectorStore

# 运行基准测试
go test ./core/cache -bench=. -benchmem
```

---

## 🧪 测试配置

### Redis 配置

测试使用以下 Redis 配置:

```go
config := cache.DefaultRedisCacheConfig()
config.Password = "redis123"  // Docker 容器密码
cache, err := cache.NewRedisCache(config)
```

### Milvus 配置

测试使用以下 Milvus 配置:

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

## ❓ 常见问题

### 1. Docker 未运行

**错误**: `Cannot connect to the Docker daemon`

**解决方案**:
- macOS/Windows: 启动 Docker Desktop
- Linux: `sudo systemctl start docker`

### 2. 端口被占用

**错误**: `port is already allocated`

**解决方案**:

```bash
# 查看占用端口的进程
lsof -i :6379   # Redis
lsof -i :19530  # Milvus

# 停止冲突的容器
docker ps | grep redis
docker stop <container_id>

# 或使用自动修复脚本
bash scripts/fix-port-conflict.sh
```

### 3. Milvus 启动慢

**正常现象**: Milvus 首次启动需要 1-2 分钟

**检查状态**:

```bash
# 查看 Milvus 日志
docker compose -f docker-compose.test.yml logs -f milvus

# 检查健康状态
curl http://localhost:9091/healthz

# 如果超时,重启容器
docker compose -f docker-compose.test.yml restart milvus
```

### 4. 测试被跳过

**现象**: `t.Skip("Redis not available")`

**原因**: 服务未运行时,测试会自动跳过

**解决方案**: 先启动测试环境

```bash
make -f Makefile.test test-env-up
```

### 5. Go 版本过低

**错误**: `go: cannot find main module`

**解决方案**: 升级到 Go 1.21+

```bash
# 下载最新 Go: https://go.dev/dl/
# 验证版本
go version
```

---

## 📊 测试覆盖率

### 生成覆盖率报告

```bash
# 生成覆盖率报告
make -f Makefile.test test-coverage

# 在浏览器中查看
open coverage.html
```

### 当前覆盖率

- **总体覆盖率**: 60%+
- **核心包**: 70%+
- **测试包数**: 35+
- **测试用例**: 500+

---

## 🔥 性能测试

运行基准测试:

```bash
# Redis 性能测试
go test ./core/cache -bench=BenchmarkRedisCache -benchmem -benchtime=10s

# Milvus 性能测试
go test ./retrieval/vectorstores -bench=. -benchmem

# 所有基准测试
go test ./... -bench=. -benchmem
```

---

## 🧹 清理环境

```bash
# 停止服务但保留数据
make -f Makefile.test test-env-down

# 停止服务并删除数据卷
docker compose -f docker-compose.test.yml down -v

# 完全清理 (包括镜像)
docker compose -f docker-compose.test.yml down -v --rmi all
```

---

## 💡 最佳实践

### 1. 日常开发流程

```bash
# 早上开始工作
make -f Makefile.test test-env-up

# 开发过程中频繁运行测试
make -f Makefile.test test

# 下班前停止服务
make -f Makefile.test test-env-down
```

### 2. 测试驱动开发

```bash
# 启动服务后保持运行
make -f Makefile.test test-env-up

# 监听文件变化自动测试 (需要 entr)
ls **/*.go | entr -c go test ./... -v
```

### 3. 调试失败的测试

```bash
# 查看服务日志
docker compose -f docker-compose.test.yml logs

# 运行单个测试
go test ./core/cache -v -run TestRedisCache_Set

# 进入容器调试
docker exec -it langchain-go-redis redis-cli -a redis123
```

---

## 🎯 CI/CD 集成

在 CI 环境中使用:

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

## 📚 相关文档

- [贡献指南](CONTRIBUTING.md) - 代码贡献规范
- [快速开始](docs/getting-started/quickstart.md) - 项目快速开始
- [开发文档](docs/development/) - 开发者文档

---

## 🆘 获取帮助

遇到问题?

1. **查看帮助**: `make -f Makefile.test help`
2. **查看日志**: `docker compose -f docker-compose.test.yml logs`
3. **验证环境**: `bash scripts/verify-setup.sh`
4. **提交 Issue**: [GitHub Issues](https://github.com/zhucl121/langchain-go/issues)

---

## ✅ 验证安装

运行验证脚本确保一切正常:

```bash
bash scripts/verify-setup.sh
```

预期输出:

```
✅ Docker 已运行
✅ 找到 docker-compose
✅ docker-compose.test.yml 存在
✅ 端口 6379 (Redis) 可用
✅ 端口 19530 (Milvus) 可用
✅ 验证完成!
```

---

**祝测试顺利! 🎉**

如有问题,请参考上述常见问题部分或提交 Issue。

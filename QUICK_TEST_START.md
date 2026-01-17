# 🚀 LangChain-Go 测试环境快速启动指南

本指南帮助你快速配置 Redis 和 Milvus 2.6.1 测试环境，运行 langchain-go 的所有测试。

## 📋 前置要求

- ✅ Docker Desktop (或 Docker Engine)
- ✅ Go 1.21+ 
- ✅ 至少 2GB 可用磁盘空间

## ⚡ 快速开始 (3步)

### 方式一: 使用 Make (推荐)

```bash
# 1. 验证环境
bash scripts/verify-setup.sh

# 2. 启动测试环境
make -f Makefile.test test-env-up

# 3. 运行测试
make -f Makefile.test test
```

### 方式二: 一键运行

```bash
# 自动验证、启动和测试
bash scripts/quick-start-test.sh
```

### 方式三: 手动步骤

```bash
# 1. 验证环境
bash scripts/verify-setup.sh

# 2. 启动服务
bash scripts/test-env-setup.sh

# 3. 运行测试
bash scripts/run-tests.sh

# 4. 停止服务
bash scripts/test-env-stop.sh
```

## 📦 包含的服务

启动后会运行以下 Docker 容器:

| 服务 | 端口 | 用途 | 配置 |
|------|------|------|------|
| **Redis** | 6379 | 缓存测试 | 密码: redis123 |
| **Milvus** | 19530 | 向量存储测试 | v2.6.1 |
| Milvus-etcd | 2379 | Milvus 依赖 | 内部使用 |
| Milvus-minio | 9000 | Milvus 存储 | 内部使用 |

## 🔍 详细命令说明

### 环境管理

```bash
# 启动测试环境
make -f Makefile.test test-env-up
# 或
bash scripts/test-env-setup.sh

# 停止测试环境
make -f Makefile.test test-env-down
# 或
bash scripts/test-env-stop.sh

# 查看服务状态
make -f Makefile.test test-env-status
# 或
docker ps --filter "name=langchain-go-"

# 查看服务日志
docker compose -f docker-compose.test.yml logs -f redis
docker compose -f docker-compose.test.yml logs -f milvus
```

### 运行测试

```bash
# 运行所有测试
make -f Makefile.test test

# 运行所有测试 (详细输出)
make -f Makefile.test test-verbose

# 仅运行 Redis 测试
make -f Makefile.test test-redis
# 或
go test ./core/cache -v -run Redis

# 仅运行 Milvus 测试
make -f Makefile.test test-milvus
# 或
go test ./retrieval/vectorstores -v -run Milvus

# 生成测试覆盖率报告
make -f Makefile.test test-coverage
```

### 特定包测试

```bash
# 测试特定包
bash scripts/run-tests.sh ./core/cache
bash scripts/run-tests.sh ./retrieval/vectorstores

# 测试特定函数
go test ./core/cache -v -run TestRedisCache_Set
go test ./retrieval/vectorstores -v -run TestMilvusVectorStore
```

## 🔧 配置说明

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
    Dimension:            1536,  // 或你的 embedding 维度
    AutoCreateCollection: true,
}
store, err := NewMilvusVectorStore(config, embeddings)
```

## ❓ 常见问题

### 1. Docker 未运行

**问题**: `❌ Docker 未运行`

**解决**:
- macOS: 启动 Docker Desktop
- Linux: `sudo systemctl start docker`
- Windows: 启动 Docker Desktop

### 2. 端口被占用

**问题**: `port is already allocated`

**解决**:

```bash
# 查看占用端口的进程
lsof -i :6379   # Redis
lsof -i :19530  # Milvus

# 停止测试容器
docker compose -f docker-compose.test.yml down

# 或修改 docker-compose.test.yml 中的端口映射
```

### 3. Milvus 启动慢

**正常现象**: Milvus 首次启动需要 1-2 分钟

**检查**:

```bash
# 查看 Milvus 日志
docker compose -f docker-compose.test.yml logs -f milvus

# 检查健康状态
curl http://localhost:9091/healthz

# 如果超时，重启容器
docker compose -f docker-compose.test.yml restart milvus
```

### 4. 测试被跳过

**现象**: `t.Skip("Redis not available")`

这是正常行为 - 如果服务未运行，测试会自动跳过。

**解决**: 先启动测试环境

```bash
make -f Makefile.test test-env-up
```

### 5. 磁盘空间不足

**问题**: 下载镜像失败

**需要空间**:
- Redis: ~30MB
- Milvus: ~1-2GB
- 总计: ~2-3GB

**清理 Docker**:

```bash
# 删除未使用的镜像
docker image prune -a

# 清理所有未使用资源
docker system prune -a
```

## 📊 测试覆盖情况

运行覆盖率测试:

```bash
make -f Makefile.test test-coverage
```

会生成 `coverage.html`，用浏览器打开查看:

```bash
open coverage.html
```

## 🔥 性能测试

运行基准测试:

```bash
# Redis 性能测试
go test ./core/cache -bench=BenchmarkRedisCache -benchmem -benchtime=10s

# Milvus 性能测试
go test ./retrieval/vectorstores -bench=. -benchmem
```

## 🧹 清理环境

```bash
# 停止服务但保留数据
docker compose -f docker-compose.test.yml down

# 停止服务并删除数据卷
docker compose -f docker-compose.test.yml down -v

# 完全清理
docker compose -f docker-compose.test.yml down -v --rmi all
```

## 📚 相关文档

- 📘 [完整测试指南](TEST_GUIDE.md) - 详细的测试说明
- 📗 [Redis 缓存文档](docs/guides/redis-cache.md) - Redis 使用指南
- 📕 [Milvus 集成文档](docs/reference/enhancements.md) - Milvus 功能说明

## 🎯 CI/CD 集成

在 CI 环境中使用:

```yaml
# .github/workflows/test.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Docker
        run: |
          docker --version
          
      - name: Start test environment
        run: bash scripts/test-env-setup.sh
        
      - name: Run tests
        run: go test ./... -v
        
      - name: Stop test environment
        if: always()
        run: bash scripts/test-env-stop.sh
```

## 💡 最佳实践

1. **开发流程**:
   ```bash
   # 早上开始工作
   make -f Makefile.test test-env-up
   
   # 开发过程中频繁运行测试
   make -f Makefile.test test
   
   # 下班前停止服务
   make -f Makefile.test test-env-down
   ```

2. **测试驱动开发**:
   ```bash
   # 启动服务后保持运行
   make -f Makefile.test test-env-up
   
   # 监听文件变化自动测试 (需要 entr)
   ls **/*.go | entr -c go test ./... -v
   ```

3. **调试失败的测试**:
   ```bash
   # 查看服务日志
   docker compose -f docker-compose.test.yml logs
   
   # 运行单个测试
   go test ./core/cache -v -run TestRedisCache_Set
   
   # 进入容器调试
   docker exec -it langchain-go-redis redis-cli -a redis123
   ```

## 🆘 获取帮助

遇到问题？

1. **查看帮助**: `make -f Makefile.test help`
2. **查看日志**: `docker compose -f docker-compose.test.yml logs`
3. **检查状态**: `make -f Makefile.test test-env-status`
4. **提交 Issue**: [GitHub Issues](https://github.com/zhuchenglong/langchain-go/issues)

## ✅ 验证安装

运行验证脚本确保一切正常:

```bash
bash scripts/verify-setup.sh
```

应该看到:

```
✅ Docker 已运行
✅ 找到 docker-compose
✅ docker-compose.test.yml 存在
✅ 端口 6379 (Redis) 可用
✅ 端口 19530 (Milvus) 可用
✅ 验证完成！
```

---

**祝测试顺利! 🎉**

如有问题，请参考 [TEST_GUIDE.md](TEST_GUIDE.md) 获取更多详细信息。

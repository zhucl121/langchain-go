# 🧪 测试

## 快速开始

```bash
# 1. 启动测试环境 (Redis + Milvus)
make -f Makefile.test test-env-up

# 2. 运行测试
make -f Makefile.test test

# 3. 停止环境
make -f Makefile.test test-env-down
```

## 前置要求

- Docker Desktop (运行中)
- Go 1.21+ ([下载](https://go.dev/dl/))
- 2GB+ 可用磁盘空间

## 测试环境

自动配置的服务：
- **Redis** (localhost:6379, 密码: redis123)
- **Milvus 2.6.1** (localhost:19530)

## 常用命令

```bash
make -f Makefile.test help              # 查看所有命令
make -f Makefile.test test-env-status   # 查看服务状态
make -f Makefile.test test-redis        # Redis 测试
make -f Makefile.test test-milvus       # Milvus 测试
make -f Makefile.test test-coverage     # 覆盖率报告
```

## 问题排查

```bash
# 端口冲突
bash scripts/fix-port-conflict.sh

# 环境验证
bash scripts/verify-setup.sh

# 查看日志
docker compose -f docker-compose.test.yml logs -f
```

## 详细文档

- **TEST_GUIDE.md** - 完整测试指南
- **QUICK_TEST_START.md** - 快速开始（详细版）
- **QUICK_REFERENCE.txt** - 命令速查
- **PORT_CONFLICT_SOLUTION.md** - 端口冲突解决
- **TEST_SUCCESS_COMPLETE.md** - 测试报告和覆盖率

---

**快速测试**: `make -f Makefile.test test-env-up && make -f Makefile.test test`

# 🧪 LangChain-Go 测试指南

> 快速配置测试环境并运行所有测试

## 📋 前置要求

- ✅ Docker Desktop (已安装并运行)
- ✅ Go 1.21+ (推荐 Go 1.25+)
- ✅ 至少 2GB 可用磁盘空间

## 🚀 快速开始

### 1. 升级 Go（如果需要）

如果你的 Go 版本低于 1.21：

```bash
# 下载并安装最新 Go: https://go.dev/dl/
# 或使用官方安装器

# 验证版本
go version  # 应显示 go1.21 或更高
```

### 2. 启动测试环境

```bash
cd /path/to/langchain-go

# 方式 1: 使用 Make（推荐）
make -f Makefile.test test-env-up

# 方式 2: 使用脚本
bash scripts/test-env-setup.sh
```

等待约 1-2 分钟，直到看到：
```
✅ Redis 就绪
✅ Milvus 就绪
```

### 3. 运行测试

```bash
# 运行所有测试
make -f Makefile.test test

# 或直接使用 go test
go test $(go list ./... | grep -v '/examples') -short
```

## 📦 测试环境服务

启动后可用的服务：

| 服务 | 地址 | 凭证 | 用途 |
|------|------|------|------|
| Redis | localhost:6379 | 密码: redis123 | 缓存测试 |
| Milvus | localhost:19530 | 无 | 向量存储测试 |

## 🔧 常用命令

```bash
# 查看所有可用命令
make -f Makefile.test help

# 启动环境
make -f Makefile.test test-env-up

# 停止环境
make -f Makefile.test test-env-down

# 查看状态
make -f Makefile.test test-env-status

# 运行所有测试
make -f Makefile.test test

# 运行 Redis 测试
make -f Makefile.test test-redis

# 运行 Milvus 测试
make -f Makefile.test test-milvus

# 生成覆盖率报告
make -f Makefile.test test-coverage
```

## 🐛 常见问题

### 问题 1: 端口被占用

**错误**: `port is already allocated`

**解决**:
```bash
# 自动修复
bash scripts/fix-port-conflict.sh

# 或手动停止冲突容器
docker ps --filter "publish=6379"  # 查看占用 6379 的容器
docker stop <container-name>       # 停止它
```

### 问题 2: Go 版本过低

**错误**: `package xxx is not in GOROOT`

**解决**: 升级到 Go 1.21+
```bash
# 下载: https://go.dev/dl/
# 安装后运行配置脚本
bash scripts/configure-go-env.sh
```

### 问题 3: Milvus 启动失败

**原因**: 首次启动需要 1-2 分钟

**解决**: 
```bash
# 查看日志
docker compose -f docker-compose.test.yml logs -f milvus

# 如果失败，重启
docker compose -f docker-compose.test.yml restart milvus
```

## 📊 测试结果

运行测试后，你应该看到类似的结果：

```
✅ 所有测试通过
- 35 个包测试成功
- 平均覆盖率: 60%+
- 执行时间: ~1 分钟
```

## 🔍 特定测试

```bash
# 测试 Redis 功能
go test ./core/cache -v -run TestRedisCache

# 测试 Milvus 功能
go test ./retrieval/vectorstores -v

# 测试 Agent 功能
go test ./core/agents -v

# 测试 LangGraph 功能
go test ./graph/... -v
```

## 📚 详细文档

- **QUICK_TEST_START.md** - 详细的快速开始指南（推荐阅读）
- **QUICK_REFERENCE.txt** - 命令速查卡片
- **PORT_CONFLICT_SOLUTION.md** - 端口冲突解决方案
- **TEST_SUCCESS_COMPLETE.md** - 完整的测试报告和覆盖率统计

## 🔄 日常使用流程

```bash
# 早上开始工作
make -f Makefile.test test-env-up

# 开发中频繁运行测试
make -f Makefile.test test

# 下班前停止环境
make -f Makefile.test test-env-down
```

## 🧹 清理

```bash
# 停止并删除容器
make -f Makefile.test test-env-down

# 或使用 Docker Compose
docker compose -f docker-compose.test.yml down -v
```

## 🆘 获取帮助

1. **查看命令帮助**: `make -f Makefile.test help`
2. **查看详细指南**: `cat QUICK_TEST_START.md`
3. **查看快速参考**: `cat QUICK_REFERENCE.txt`
4. **运行诊断**: `bash scripts/verify-setup.sh`

## ✅ 验证安装

运行此命令验证环境配置：

```bash
bash scripts/verify-setup.sh
```

应该看到：
```
✅ Docker 已运行
✅ 找到 docker-compose
✅ docker-compose.test.yml 存在
✅ 端口可用
✅ 验证完成！
```

---

**快速开始**: `make -f Makefile.test test-env-up && make -f Makefile.test test` 🚀

**问题反馈**: [GitHub Issues](https://github.com/zhuchenglong/langchain-go/issues)

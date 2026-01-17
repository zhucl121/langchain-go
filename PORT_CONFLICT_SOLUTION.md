# 🔧 端口冲突解决方案

## 问题

启动测试环境时遇到以下错误：
```
Error response from daemon: driver failed programming external connectivity on endpoint langchain-go-redis: 
Bind for 0.0.0.0:6379 failed: port is already allocated
```

## 原因

你有一个名为 `optimus-redis` 的 Redis 容器正在运行，占用了 6379 端口。

## ✅ 快速解决

### 方案 1: 使用自动修复工具（推荐）

```bash
cd /Users/yunyuexingsheng/Documents/worksapce/随笔/langchain-go
bash scripts/fix-port-conflict.sh
```

这个脚本会：
1. 检查端口占用情况
2. 显示占用端口的容器
3. 提供多种解决方案
4. 可选自动停止冲突的容器

### 方案 2: 手动停止冲突的容器

```bash
# 停止 optimus-redis
docker stop optimus-redis

# 启动测试环境
make -f Makefile.test test-env-up
```

### 方案 3: 使用不同的端口

如果你需要保持 `optimus-redis` 运行，可以修改测试环境使用不同的端口。

编辑 `docker-compose.test.yml`，将 Redis 端口改为：

```yaml
redis:
  ports:
    - "6380:6379"  # 使用 6380 而不是 6379
```

然后更新测试代码中的连接配置：

```bash
# 在测试代码中使用
REDIS_ADDR=localhost:6380
```

### 方案 4: 查看所有占用端口的容器

```bash
# 查看占用 6379 的容器
docker ps --filter "publish=6379"

# 查看占用 19530 的容器
docker ps --filter "publish=19530"

# 查看所有 Redis 容器
docker ps | grep redis
```

## 🔍 诊断命令

```bash
# 检查端口占用
lsof -i :6379
lsof -i :19530

# 查看所有 Docker 容器
docker ps -a

# 查看测试环境容器状态
docker ps --filter "name=langchain-go-"
```

## 📋 当前情况

根据检查，你有以下容器：

| 容器名称 | 状态 | 端口映射 | 说明 |
|---------|------|---------|------|
| optimus-redis | 运行中 | 0.0.0.0:6379->6379/tcp | 占用 6379 端口 |
| langchain-go-redis | 已创建 | 无 | 测试环境容器（未启动）|

## 💡 推荐操作流程

1. **临时停止 optimus-redis**（如果不需要）:
   ```bash
   docker stop optimus-redis
   ```

2. **启动测试环境**:
   ```bash
   make -f Makefile.test test-env-up
   ```

3. **运行测试**:
   ```bash
   make -f Makefile.test test
   ```

4. **测试完成后，重启 optimus-redis**（如果需要）:
   ```bash
   docker start optimus-redis
   ```

## 🔄 长期解决方案

如果你经常需要同时运行两个 Redis 实例，建议：

### 选项 A: 使用不同端口

将测试环境的 Redis 端口改为 6380：

1. 修改 `docker-compose.test.yml`:
   ```yaml
   redis:
     ports:
       - "6380:6379"
   ```

2. 修改测试配置:
   ```go
   config := cache.DefaultRedisCacheConfig()
   config.Addr = "localhost:6380"  // 使用 6380
   config.Password = "redis123"
   ```

### 选项 B: 使用环境变量

创建 `.env.test` 文件：
```bash
REDIS_PORT=6380
```

然后在 `docker-compose.test.yml` 中使用：
```yaml
redis:
  ports:
    - "${REDIS_PORT:-6379}:6379"
```

## 🆘 仍然遇到问题？

运行完整的诊断：

```bash
# 运行修复工具
bash scripts/fix-port-conflict.sh

# 或查看详细状态
docker ps -a
docker compose -f docker-compose.test.yml ps
lsof -i :6379
lsof -i :19530
```

## 📞 获取帮助

如果问题仍然存在：

1. 查看 Docker 日志:
   ```bash
   docker compose -f docker-compose.test.yml logs
   ```

2. 重启 Docker Desktop

3. 查看 `QUICK_TEST_START.md` 中的故障排查部分

---

**快速修复**: `bash scripts/fix-port-conflict.sh`

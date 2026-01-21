# 集群管理示例

这个示例展示了 LangChain-Go 的集群管理功能，包括节点注册、服务发现、健康检查等。

## 功能演示

- ✅ Consul 服务发现
- ✅ 节点注册和注销
- ✅ 心跳机制
- ✅ 节点变化监听
- ✅ 健康检查（HTTP 和 TCP）
- ✅ 节点过滤

## 前置条件

### 安装 Consul（可选）

如果要演示真实的服务发现功能，需要安装 Consul：

```bash
# 使用 Docker 运行 Consul
docker run -d --name consul -p 8500:8500 consul:latest

# 验证 Consul 是否运行
curl http://localhost:8500/v1/status/leader
```

如果没有 Consul，程序会自动切换到模拟模式。

## 运行示例

### 方式 1: 使用 Consul（推荐）

```bash
# 启动 Consul
docker run -d --name consul -p 8500:8500 consul:latest

# 运行示例
cd examples/cluster_demo
go run main.go
```

### 方式 2: 模拟模式

```bash
# 直接运行（不需要 Consul）
cd examples/cluster_demo
go run main.go
```

程序会自动检测 Consul 是否可用，如果不可用会使用模拟模式。

## 示例输出

### Consul 模式

```
🚀 LangChain-Go 集群管理示例
========================================

📡 连接到 Consul: localhost:8500
✅ 成功连接到 Consul

📝 注册节点: worker-1737525600
✅ 节点注册成功

💓 启动心跳...

👀 监听集群节点变化...
➕ 节点加入: demo-worker (worker-1737525600)

📋 当前集群节点:
  1. demo-worker (worker-1737525600) - 127.0.0.1:8080 - 状态: online

🏥 健康检查示例:
  ⚠️  TCP 检查失败: TCP connection failed after 1 retries
  ⚠️  HTTP 检查失败: HTTP request failed

✅ 集群管理系统运行中...
按 Ctrl+C 退出

💓 心跳发送成功
💓 心跳发送成功
```

### 模拟模式

```
🚀 LangChain-Go 集群管理示例
========================================

📡 连接到 Consul: localhost:8500
⚠️  无法连接到 Consul: ...
运行模拟模式...

🎭 模拟模式
========================================

📋 模拟集群节点:

1. worker-1 (worker-1)
   地址: 192.168.1.10:8080
   状态: online
   角色: [worker]
   容量: 1000 连接, 500 QPS, 4096 MB 内存
   负载: 500 连接 (50.0%), CPU 45.0%, 内存 2048 MB
   健康: ✅ 健康

2. worker-2 (worker-2)
   地址: 192.168.1.11:8080
   状态: busy
   角色: [worker]
   容量: 1000 连接, 500 QPS, 4096 MB 内存
   负载: 850 连接 (85.0%), CPU 80.0%, 内存 3500 MB
   健康: ✅ 健康

3. cache-1 (cache-1)
   地址: 192.168.1.20:6379
   状态: online
   角色: [cache]
   容量: 10000 连接, 0 QPS, 8192 MB 内存
   负载: 2000 连接 (20.0%), CPU 0.0%, 内存 4096 MB
   健康: ✅ 健康

🔍 节点过滤示例:

在线节点 (2 个):
  - worker-1 (worker-1)
  - cache-1 (cache-1)

健康的工作节点 (2 个):
  - worker-1 (负载: 50.0%)
  - worker-2 (负载: 85.0%)

✅ 模拟完成
```

## 核心概念

### 1. 节点管理

节点是集群中的基本单位，包含：
- 唯一标识符（ID）
- 地址和端口
- 状态（online, offline, busy, draining, maintenance）
- 角色（master, worker, cache, gateway）
- 容量信息
- 负载信息

### 2. 服务发现

使用 Consul 实现自动服务发现：
- 节点自动注册
- 节点变化监听
- 健康检查集成
- 自动注销机制

### 3. 健康检查

支持多种健康检查方式：
- TCP 连接检查
- HTTP 端点检查
- 自定义检查
- 组合检查

### 4. 节点过滤

灵活的节点过滤机制：
- 按状态过滤
- 按角色过滤
- 按区域过滤
- 按负载过滤
- 组合条件

## 多节点测试

要测试多节点集群，可以启动多个实例：

```bash
# 终端 1
cd examples/cluster_demo
go run main.go

# 终端 2
cd examples/cluster_demo
go run main.go

# 终端 3
cd examples/cluster_demo
go run main.go
```

你会看到每个节点的加入和心跳日志。

## 访问 Consul UI

Consul 提供了 Web UI 来查看集群状态：

```bash
# 打开浏览器访问
open http://localhost:8500/ui
```

在 UI 中可以看到：
- 所有注册的服务
- 节点健康状态
- 服务标签和元数据

## 清理

```bash
# 停止 Consul 容器
docker stop consul
docker rm consul
```

## 下一步

查看更多示例：
- 负载均衡示例：`examples/load_balancer_demo/`（开发中）
- 故障转移示例：`examples/failover_demo/`（开发中）
- 分布式缓存示例：`examples/distributed_cache_demo/`（开发中）

## 相关文档

- [V0.5.0 实施计划](../../docs/V0.5.0_IMPLEMENTATION_PLAN.md)
- [V0.5.0 进度跟踪](../../docs/V0.5.0_PROGRESS.md)

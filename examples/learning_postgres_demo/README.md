# Learning Retrieval - PostgreSQL 存储示例

这个示例展示如何在生产环境中使用 PostgreSQL 存储用户反馈数据。

## 为什么需要 PostgreSQL？

| 特性 | 内存存储 | PostgreSQL 存储 |
|------|---------|----------------|
| **数据持久化** | ❌ 重启丢失 | ✅ 永久保存 |
| **数据规模** | 受内存限制 | ✅ 支持海量数据 |
| **查询能力** | 简单过滤 | ✅ 复杂 SQL 查询 |
| **并发能力** | 中等 | ✅ 高并发支持 |
| **生产环境** | ❌ 仅测试用 | ✅ 生产级可靠 |
| **使用场景** | 测试/演示 | 生产部署 |

## 前置条件

### 启动 PostgreSQL

**使用 Docker**:
```bash
docker run -d --name postgres-learning \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=langchain_learning \
  -p 5432:5432 \
  postgres:15
```

**或使用项目的 docker-compose**:
```bash
docker-compose -f docker-compose.test.yml up -d postgres
```

## 运行示例

```bash
# 设置数据库连接（可选，有默认值）
export POSTGRES_URL="postgres://postgres:password@localhost:5432/langchain_learning?sslmode=disable"

# 运行示例
cd examples/learning_postgres_demo
go run main.go
```

## 输出示例

```
=== LangChain-Go Learning Retrieval - PostgreSQL 存储示例 ===

✅ 成功连接到 PostgreSQL 数据库
🔧 初始化数据库表结构...
✅ 数据库表创建成功
   📋 创建了 4 张表:
      - learning_queries
      - learning_results
      - learning_explicit_feedback
      - learning_implicit_feedback

📝 保存测试数据到 PostgreSQL...
✅ 查询已保存 (ID: xxx)
✅ 检索结果已保存 (3 个文档)
✅ 用户反馈已保存 (5 星好评)
✅ 用户行为已保存 (阅读 90 秒)

📖 从 PostgreSQL 读取数据...

查询信息:
  📝 查询: PostgreSQL 存储示例查询
  👤 用户: demo-user
  🎯 策略: hybrid
  📊 结果数: 3
  ⭐ 平均评分: 5.0/5
  📈 点击率: 33.3%
  ⏱️  阅读时长: 1m30s

📊 数据库统计:
  📈 总查询数: 1
  ⭐ 平均评分: 5.00/5
  👍 正面率: 0.0%
  📊 平均 CTR: 33.3%

✅ PostgreSQL 存储示例完成！
```

## 数据库结构

### 表结构

```sql
-- 1. 查询表
CREATE TABLE learning_queries (
    id VARCHAR(255) PRIMARY KEY,
    text TEXT NOT NULL,
    user_id VARCHAR(255),
    strategy VARCHAR(100),
    timestamp TIMESTAMP NOT NULL,
    metadata JSONB
);

-- 2. 检索结果表
CREATE TABLE learning_results (
    id SERIAL PRIMARY KEY,
    query_id VARCHAR(255) REFERENCES learning_queries(id),
    document_id VARCHAR(255),
    rank INT,
    score FLOAT,
    document JSONB,
    timestamp TIMESTAMP NOT NULL
);

-- 3. 显式反馈表
CREATE TABLE learning_explicit_feedback (
    id SERIAL PRIMARY KEY,
    query_id VARCHAR(255) REFERENCES learning_queries(id),
    user_id VARCHAR(255),
    type VARCHAR(50),
    rating INT,
    comment TEXT,
    timestamp TIMESTAMP NOT NULL
);

-- 4. 隐式反馈表
CREATE TABLE learning_implicit_feedback (
    id SERIAL PRIMARY KEY,
    query_id VARCHAR(255) REFERENCES learning_queries(id),
    user_id VARCHAR(255),
    document_id VARCHAR(255),
    action VARCHAR(50),
    duration_ms BIGINT,
    timestamp TIMESTAMP NOT NULL
);
```

### 索引优化

```sql
-- 查询优化索引
CREATE INDEX idx_learning_queries_user ON learning_queries(user_id);
CREATE INDEX idx_learning_queries_timestamp ON learning_queries(timestamp);
CREATE INDEX idx_learning_queries_strategy ON learning_queries(strategy);

-- 反馈查询索引
CREATE INDEX idx_learning_explicit_query ON learning_explicit_feedback(query_id);
CREATE INDEX idx_learning_implicit_query ON learning_implicit_feedback(query_id);
```

## 代码示例

```go
package main

import (
    "database/sql"
    _ "github.com/lib/pq"
    "github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

func main() {
    // 1. 连接 PostgreSQL
    db, err := sql.Open("postgres", 
        "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 2. 创建 PostgreSQL 存储
    storage := feedback.NewPostgreSQLStorage(db)

    // 3. 初始化表结构（首次使用）
    pgStorage := storage.(*feedback.PostgreSQLStorage)
    if err := pgStorage.InitSchema(ctx); err != nil {
        panic(err)
    }

    // 4. 创建收集器（API 和内存存储完全相同！）
    collector := feedback.NewCollector(storage)

    // 5. 使用（和内存存储 API 一致）
    collector.RecordQuery(ctx, query)
    collector.CollectExplicitFeedback(ctx, feedback)
    collector.GetQueryFeedback(ctx, queryID)
}
```

## 性能特点

### 写入性能
- 单条插入: ~10-50ms（取决于网络和配置）
- 批量插入: 使用事务可大幅提升性能
- 建议: 批量操作使用事务

### 查询性能
- 索引查询: ~5-20ms
- 聚合统计: ~20-100ms
- 优化: 合理使用索引，避免全表扫描

### 优化建议

1. **批量操作**
   ```go
   // 使用事务批量插入
   tx, _ := db.Begin()
   for _, query := range queries {
       storage.SaveQuery(ctx, query)
   }
   tx.Commit()
   ```

2. **连接池配置**
   ```go
   db.SetMaxOpenConns(25)
   db.SetMaxIdleConns(5)
   db.SetConnMaxLifetime(5 * time.Minute)
   ```

3. **定期维护**
   ```sql
   -- 分析表统计信息
   ANALYZE learning_queries;
   
   -- 清理旧数据
   DELETE FROM learning_queries 
   WHERE timestamp < NOW() - INTERVAL '90 days';
   ```

## 生产环境配置

### 环境变量

```bash
# 数据库连接
export POSTGRES_URL="postgres://user:pass@host:port/db?sslmode=require"

# 连接池配置
export DB_MAX_OPEN_CONNS=25
export DB_MAX_IDLE_CONNS=5
export DB_CONN_MAX_LIFETIME=5m
```

### 监控指标

建议监控：
- 查询响应时间
- 连接池使用率
- 慢查询日志
- 表大小增长
- 索引效率

## 故障排除

### 连接失败

```bash
# 检查 PostgreSQL 是否运行
docker ps | grep postgres

# 检查端口
netstat -an | grep 5432

# 测试连接
psql -h localhost -U postgres -d langchain_learning
```

### 表已存在错误

```sql
-- 删除旧表（谨慎！）
DROP TABLE IF EXISTS learning_implicit_feedback CASCADE;
DROP TABLE IF EXISTS learning_explicit_feedback CASCADE;
DROP TABLE IF EXISTS learning_results CASCADE;
DROP TABLE IF EXISTS learning_queries CASCADE;
```

### 性能问题

```sql
-- 检查慢查询
SELECT * FROM pg_stat_statements 
ORDER BY total_time DESC LIMIT 10;

-- 检查索引使用
SELECT * FROM pg_stat_user_indexes 
WHERE schemaname = 'public';
```

## 对比总结

**内存存储** (`NewMemoryStorage()`):
- ✅ 零配置，开箱即用
- ✅ 极快性能（0.1ms）
- ✅ 适合测试和演示
- ❌ 数据不持久化
- ❌ 内存限制

**PostgreSQL 存储** (`NewPostgreSQLStorage(db)`):
- ✅ 数据持久化
- ✅ 支持大规模数据
- ✅ 生产级可靠性
- ✅ 强大的查询能力
- ⚠️ 需要部署数据库
- ⚠️ 略慢于内存（但可优化）

## 下一步

- 查看 `learning_feedback_demo` 了解基础用法
- 查看 `learning_evaluation_demo` 了解评估功能
- 集成到实际项目中使用 PostgreSQL 存储

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/learning/feedback"
)

func main() {
	fmt.Println("=== LangChain-Go Learning Retrieval - PostgreSQL 存储示例 ===")

	// 从环境变量获取数据库连接字符串
	// 格式: postgres://user:password@localhost:5432/dbname?sslmode=disable
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		// 使用默认配置
		connStr = "postgres://postgres:password@localhost:5432/langchain_learning?sslmode=disable"
		fmt.Printf("⚠️  未设置 POSTGRES_URL 环境变量，使用默认值\n")
		fmt.Printf("   默认连接: %s\n\n", connStr)
		fmt.Println("💡 提示: 如果没有 PostgreSQL，请先启动:")
		fmt.Println("   docker run -d --name postgres-learning \\")
		fmt.Println("     -e POSTGRES_PASSWORD=password \\")
		fmt.Println("     -e POSTGRES_DB=langchain_learning \\")
		fmt.Println("     -p 5432:5432 \\")
		fmt.Println("     postgres:15")
		fmt.Println()
		fmt.Println("📝 本示例将展示如何连接，实际运行需要数据库")
		fmt.Println("   如果使用内存存储，请运行 learning_feedback_demo")
		fmt.Println("=" + string(make([]byte, 60)) + "\n")
	}

	// 尝试连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		fmt.Println("\n💡 如果没有 PostgreSQL，可以:")
		fmt.Println("   1. 运行上述 docker 命令启动数据库")
		fmt.Println("   2. 或使用内存存储运行其他示例")
		demonstrateUsage()
		return
	}
	defer db.Close()

	// 测试连接
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		fmt.Printf("❌ 数据库连接测试失败: %v\n", err)
		fmt.Println("\n💡 请检查 PostgreSQL 是否正在运行")
		demonstrateUsage()
		return
	}

	fmt.Println("✅ 成功连接到 PostgreSQL 数据库")

	// 创建 PostgreSQL 存储
	storage := feedback.NewPostgreSQLStorage(db)

	// 初始化数据库 Schema
	fmt.Println("🔧 初始化数据库表结构...")
	pgStorage := storage.(*feedback.PostgreSQLStorage)
	if err := pgStorage.InitSchema(ctx); err != nil {
		fmt.Printf("❌ 初始化失败: %v\n", err)
		return
	}
	fmt.Println("✅ 数据库表创建成功")
	fmt.Println("   📋 创建了 4 张表:")
	fmt.Println("      - learning_queries")
	fmt.Println("      - learning_results")
	fmt.Println("      - learning_explicit_feedback")
	fmt.Println("      - learning_implicit_feedback")

	// 创建收集器
	collector := feedback.NewCollector(storage)

	// 示例：保存查询和反馈
	fmt.Println("📝 保存测试数据到 PostgreSQL...")
	
	queryID := uuid.New().String()
	query := &feedback.Query{
		ID:        queryID,
		Text:      "PostgreSQL 存储示例查询",
		UserID:    "demo-user",
		Strategy:  "hybrid",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "demo",
			"env":    "production",
		},
	}

	if err := collector.RecordQuery(ctx, query); err != nil {
		fmt.Printf("❌ 保存查询失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 查询已保存 (ID: %s)\n", queryID)

	// 保存检索结果
	results := []types.Document{
		{ID: "doc-1", Content: "PostgreSQL 是强大的关系数据库"},
		{ID: "doc-2", Content: "支持 JSONB、全文搜索等高级特性"},
		{ID: "doc-3", Content: "适合生产环境的数据持久化"},
	}

	if err := collector.RecordResults(ctx, queryID, results); err != nil {
		fmt.Printf("❌ 保存结果失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 检索结果已保存 (%d 个文档)\n", len(results))

	// 保存显式反馈
	if err := collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
		QueryID:   queryID,
		UserID:    "demo-user",
		Type:      feedback.FeedbackTypeRating,
		Rating:    5,
		Comment:   "PostgreSQL 存储测试",
		Timestamp: time.Now(),
	}); err != nil {
		fmt.Printf("❌ 保存反馈失败: %v\n", err)
		return
	}
	fmt.Println("✅ 用户反馈已保存 (5 星好评)")

	// 保存隐式反馈
	if err := collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
		QueryID:    queryID,
		UserID:     "demo-user",
		DocumentID: "doc-1",
		Action:     feedback.ActionRead,
		Duration:   90 * time.Second,
		Timestamp:  time.Now(),
	}); err != nil {
		fmt.Printf("❌ 保存行为失败: %v\n", err)
		return
	}
	fmt.Println("✅ 用户行为已保存 (阅读 90 秒)")

	// 从数据库读取反馈
	fmt.Println("📖 从 PostgreSQL 读取数据...")
	qf, err := collector.GetQueryFeedback(ctx, queryID)
	if err != nil {
		fmt.Printf("❌ 读取失败: %v\n", err)
		return
	}

	fmt.Printf("\n查询信息:\n")
	fmt.Printf("  📝 查询: %s\n", qf.Query.Text)
	fmt.Printf("  👤 用户: %s\n", qf.Query.UserID)
	fmt.Printf("  🎯 策略: %s\n", qf.Query.Strategy)
	fmt.Printf("  📊 结果数: %d\n", len(qf.Results))
	fmt.Printf("  ⭐ 平均评分: %.1f/5\n", qf.AvgRating)
	fmt.Printf("  📈 点击率: %.1f%%\n", qf.CTR*100)
	fmt.Printf("  ⏱️  阅读时长: %v\n", qf.AvgReadDuration)

	// 聚合统计
	fmt.Println("\n📊 数据库统计:")
	stats, err := collector.AggregateStats(ctx, feedback.AggregateOptions{
		TimeRange: 24 * time.Hour,
	})
	if err != nil {
		fmt.Printf("❌ 统计失败: %v\n", err)
		return
	}

	fmt.Printf("  📈 总查询数: %d\n", stats.TotalQueries)
	fmt.Printf("  ⭐ 平均评分: %.2f/5\n", stats.AvgRating)
	fmt.Printf("  👍 正面率: %.1f%%\n", stats.PositiveRate*100)
	fmt.Printf("  📊 平均 CTR: %.1f%%\n", stats.AvgCTR*100)

	fmt.Println("\n✅ PostgreSQL 存储示例完成！")
	fmt.Println("\n💡 生产环境优势:")
	fmt.Println("   • 数据持久化，重启不丢失")
	fmt.Println("   • 支持大规模数据存储")
	fmt.Println("   • 支持复杂查询和聚合")
	fmt.Println("   • 支持事务和数据一致性")
	fmt.Println("   • JSONB 高效存储元数据")
}

func demonstrateUsage() {
	fmt.Println("\n📖 PostgreSQL 存储使用示例:")
	
	fmt.Println("```go")
	fmt.Println("// 1. 连接 PostgreSQL")
	fmt.Println(`db, _ := sql.Open("postgres", "postgres://...")`)
	fmt.Println("")
	fmt.Println("// 2. 创建存储")
	fmt.Println("storage := feedback.NewPostgreSQLStorage(db)")
	fmt.Println("")
	fmt.Println("// 3. 初始化表结构")
	fmt.Println("storage.(*feedback.PostgreSQLStorage).InitSchema(ctx)")
	fmt.Println("")
	fmt.Println("// 4. 创建收集器（和内存存储用法完全相同）")
	fmt.Println("collector := feedback.NewCollector(storage)")
	fmt.Println("")
	fmt.Println("// 5. 使用（API 完全一致）")
	fmt.Println("collector.RecordQuery(ctx, query)")
	fmt.Println("collector.CollectExplicitFeedback(ctx, feedback)")
	fmt.Println("```")
	
	fmt.Println("\n🔑 关键优势:")
	fmt.Println("   ✅ API 完全一致 - 只需切换存储实现")
	fmt.Println("   ✅ 生产级可靠性 - PostgreSQL 的稳定性")
	fmt.Println("   ✅ 高性能索引 - 支持快速查询")
	fmt.Println("   ✅ JSONB 支持 - 灵活的元数据存储")
}

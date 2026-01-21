// Package feedback 提供用户反馈收集功能。
//
// 该包实现了学习型检索系统的反馈收集组件，支持：
//
// 1. 显式反馈收集（点赞、评分、评论）
// 2. 隐式反馈收集（点击、阅读、复制等行为）
// 3. 查询和结果记录
// 4. 反馈聚合统计
//
// 基本用法：
//
//	// 创建反馈收集器
//	collector := feedback.NewCollector(storage)
//
//	// 记录查询
//	query := &feedback.Query{
//	    ID:       uuid.New().String(),
//	    Text:     "什么是机器学习？",
//	    UserID:   "user-123",
//	    Strategy: "hybrid",
//	}
//	collector.RecordQuery(ctx, query)
//
//	// 记录检索结果
//	collector.RecordResults(ctx, query.ID, documents)
//
//	// 收集显式反馈
//	collector.CollectExplicitFeedback(ctx, &feedback.ExplicitFeedback{
//	    QueryID: query.ID,
//	    Type:    feedback.FeedbackTypePositive,
//	    Rating:  5,
//	})
//
//	// 收集隐式反馈
//	collector.CollectImplicitFeedback(ctx, &feedback.ImplicitFeedback{
//	    QueryID:    query.ID,
//	    DocumentID: "doc-1",
//	    Action:     feedback.ActionClick,
//	    Duration:   30 * time.Second,
//	})
//
//	// 聚合统计
//	stats, _ := collector.AggregateStats(ctx, feedback.AggregateOptions{
//	    TimeRange: 24 * time.Hour,
//	})
//
// 存储后端：
//
// 支持多种存储后端：
//   - PostgreSQL: 生产环境推荐
//   - Memory: 用于测试和原型开发
//
// 示例：
//
//	// PostgreSQL 存储
//	storage := feedback.NewPostgreSQLStorage(db)
//
//	// 内存存储
//	storage := feedback.NewMemoryStorage()
//
package feedback

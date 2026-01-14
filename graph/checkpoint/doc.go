// Package checkpoint 提供 LangGraph Checkpoint 系统。
//
// checkpoint 包负责管理图执行过程中的状态持久化和恢复。
//
// # Checkpoint 概念
//
// Checkpoint 是图执行过程中某个时刻的状态快照，包括：
//   - 当前状态
//   - 执行历史
//   - 元数据信息
//   - 配置信息
//
// # 核心组件
//
// 1. **Checkpoint** - 检查点数据结构
//   - 状态快照
//   - 版本信息
//   - 时间戳
//   - 父检查点引用
//
// 2. **CheckpointSaver** - 检查点保存器接口
//   - Save: 保存检查点
//   - Load: 加载检查点
//   - List: 列出检查点
//
// 3. **CheckpointConfig** - 检查点配置
//   - Thread ID: 线程标识
//   - Checkpoint ID: 检查点标识
//   - 元数据
//
// # 基本使用
//
// 保存检查点：
//
//	saver := checkpoint.NewMemoryCheckpointSaver[MyState]()
//	
//	config := checkpoint.NewCheckpointConfig("thread-1")
//	checkpoint := checkpoint.NewCheckpoint("cp-1", state, config)
//	
//	err := saver.Save(ctx, checkpoint)
//
// 加载检查点：
//
//	checkpoint, err := saver.Load(ctx, config)
//	if err != nil {
//	    // 处理错误
//	}
//	
//	state := checkpoint.GetState()
//
// 列出检查点：
//
//	checkpoints, err := saver.List(ctx, "thread-1")
//
// # 存储后端
//
// 系统支持多种存储后端：
//   - Memory: 内存存储（开发/测试）
//   - SQLite: SQLite 数据库（单机）
//   - Postgres: PostgreSQL 数据库（生产）
//
// # 时间旅行
//
// Checkpoint 支持时间旅行功能：
//   - 从任意检查点恢复执行
//   - 查看执行历史
//   - 分支执行路径
//
package checkpoint

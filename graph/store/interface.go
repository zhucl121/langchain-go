// Package store 提供 LangGraph 跨会话长期记忆存储层。
//
// 对标 LangGraph 1.0 LangGraph Store，支持按命名空间存储任意结构化数据，
// 用于跨线程/跨会话的长期记忆管理。
//
// 命名空间约定：
//   - ["user", userID, "memories"] - 用户记忆
//   - ["agent", agentID, "skills"] - Agent 技能
//   - ["org", orgID, "knowledge"]  - 组织知识库
//
// 使用示例：
//
//	store := graphstore.NewInMemoryStore()
//
//	// 存储用户记忆
//	store.Put(ctx, []string{"user", "alice", "preferences"}, "coding-style",
//	    map[string]any{"language": "Go", "style": "minimal"},
//	)
//
//	// 语义检索
//	results, _ := store.Search(ctx,
//	    []string{"user", "alice", "preferences"},
//	    "编程语言偏好",
//	    graphstore.SearchOptions{K: 5},
//	)
//
package store

import (
	"context"
	"time"
)

// StoreItem 存储条目。
type StoreItem struct {
	// Namespace 命名空间（层级路径）
	Namespace []string `json:"namespace"`

	// Key 条目键
	Key string `json:"key"`

	// Value 存储的值
	Value any `json:"value"`

	// Score 相关性分数（检索时设置）
	Score float64 `json:"score,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// SearchOptions 检索选项。
type SearchOptions struct {
	// K 最大返回条数（默认 10）
	K int

	// Filter 元数据过滤条件
	Filter map[string]any

	// MinScore 最低相关性分数（0 表示不过滤）
	MinScore float64
}

// DefaultSearchOptions 返回默认检索选项。
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{K: 10}
}

// ListOptions 列举命名空间选项。
type ListOptions struct {
	// Prefix 命名空间前缀过滤
	Prefix []string

	// MaxDepth 最大深度（0 表示不限）
	MaxDepth int

	// Limit 最大返回数
	Limit int
}

// Store LangGraph 风格的长期记忆存储接口。
//
// Store 使用命名空间（namespace）组织数据，类似文件系统的目录结构。
// 命名空间是一个字符串切片，如 ["user", "alice", "memories"]。
type Store interface {
	// Put 存储条目（不存在则创建，存在则更新）
	Put(ctx context.Context, namespace []string, key string, value any) error

	// Get 获取单个条目
	Get(ctx context.Context, namespace []string, key string) (*StoreItem, error)

	// Delete 删除条目
	Delete(ctx context.Context, namespace []string, key string) error

	// Search 在命名空间内检索（关键词或语义）
	Search(ctx context.Context, namespace []string, query string, opts SearchOptions) ([]*StoreItem, error)

	// List 列举命名空间中的所有条目
	List(ctx context.Context, namespace []string, opts ListOptions) ([]*StoreItem, error)

	// ListNamespaces 列举所有命名空间
	ListNamespaces(ctx context.Context, opts ListOptions) ([][]string, error)

	// BatchPut 批量存储
	BatchPut(ctx context.Context, items []*StoreItem) error

	// BatchGet 批量获取
	BatchGet(ctx context.Context, keys []NamespaceKey) ([]*StoreItem, error)
}

// NamespaceKey 命名空间 + 键的组合。
type NamespaceKey struct {
	Namespace []string
	Key       string
}

// ContextStoreKey context 中存储 Store 的键类型。
type contextStoreKey struct{}

// WithStore 将 Store 注入 context（在 StateGraph 节点中访问）。
func WithStore(ctx context.Context, store Store) context.Context {
	return context.WithValue(ctx, contextStoreKey{}, store)
}

// GetStore 从 context 中获取 Store。
//
// 在 StateGraph 节点函数内调用，获取当前执行上下文中的 Store。
//
// 示例：
//
//	func myNode(ctx context.Context, state MyState) (MyState, error) {
//	    store, ok := graphstore.GetStore(ctx)
//	    if ok {
//	        memories, _ := store.Search(ctx,
//	            []string{"user", state.UserID, "memories"},
//	            state.Query,
//	            graphstore.DefaultSearchOptions(),
//	        )
//	        // 使用 memories 增强响应...
//	    }
//	    return state, nil
//	}
func GetStore(ctx context.Context) (Store, bool) {
	store, ok := ctx.Value(contextStoreKey{}).(Store)
	return store, ok
}

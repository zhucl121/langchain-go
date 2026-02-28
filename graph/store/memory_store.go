package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// namespaceKey 内部存储键（namespace + key 组合）。
type internalKey struct {
	namespace string // namespace 路径，以 "/" 分隔
	key       string
}

// InMemoryStore 基于内存的 LangGraph Store 实现。
//
// 适用于开发和测试环境。生产环境推荐使用 PostgresStore 或 RedisStore。
//
// 示例：
//
//	store := graphstore.NewInMemoryStore()
//	graph.WithStore(store)  // 集成到 StateGraph
type InMemoryStore struct {
	mu    sync.RWMutex
	items map[internalKey]*StoreItem
}

// NewInMemoryStore 创建内存 Store。
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		items: make(map[internalKey]*StoreItem),
	}
}

// namespaceToString 将命名空间转换为字符串路径。
func namespaceToString(namespace []string) string {
	return strings.Join(namespace, "/")
}

// Put 存储条目。
func (s *InMemoryStore) Put(ctx context.Context, namespace []string, key string, value any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ikey := internalKey{
		namespace: namespaceToString(namespace),
		key:       key,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if existing, ok := s.items[ikey]; ok {
		existing.Value = value
		existing.UpdatedAt = now

		// 保留 namespace/key 的原始切片（深拷贝）
		ns := make([]string, len(namespace))
		copy(ns, namespace)
		existing.Namespace = ns
	} else {
		ns := make([]string, len(namespace))
		copy(ns, namespace)

		s.items[ikey] = &StoreItem{
			Namespace: ns,
			Key:       key,
			Value:     value,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return nil
}

// Get 获取单个条目。
func (s *InMemoryStore) Get(ctx context.Context, namespace []string, key string) (*StoreItem, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ikey := internalKey{
		namespace: namespaceToString(namespace),
		key:       key,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[ikey]
	if !ok {
		return nil, nil
	}
	return cloneItem(item), nil
}

// Delete 删除条目。
func (s *InMemoryStore) Delete(ctx context.Context, namespace []string, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ikey := internalKey{
		namespace: namespaceToString(namespace),
		key:       key,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, ikey)
	return nil
}

// Search 在命名空间内关键词检索。
func (s *InMemoryStore) Search(ctx context.Context, namespace []string, query string, opts SearchOptions) ([]*StoreItem, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if opts.K <= 0 {
		opts.K = 10
	}

	nsStr := namespaceToString(namespace)
	queryLower := strings.ToLower(query)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*StoreItem
	for ikey, item := range s.items {
		if ikey.namespace != nsStr {
			continue
		}

		// 简单关键词匹配（生产环境应使用向量检索）
		score := 0.0
		valueStr := fmt.Sprintf("%v", item.Value)
		if strings.Contains(strings.ToLower(valueStr), queryLower) {
			score = 1.0
		} else if strings.Contains(strings.ToLower(ikey.key), queryLower) {
			score = 0.8
		}

		if score >= opts.MinScore && (score > 0 || opts.MinScore == 0) {
			cloned := cloneItem(item)
			cloned.Score = score
			results = append(results, cloned)
		}

		if len(results) >= opts.K {
			break
		}
	}

	return results, nil
}

// List 列举命名空间中的所有条目。
func (s *InMemoryStore) List(ctx context.Context, namespace []string, opts ListOptions) ([]*StoreItem, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	nsStr := namespaceToString(namespace)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*StoreItem
	for ikey, item := range s.items {
		if ikey.namespace != nsStr {
			continue
		}

		results = append(results, cloneItem(item))

		if opts.Limit > 0 && len(results) >= opts.Limit {
			break
		}
	}

	return results, nil
}

// ListNamespaces 列举所有命名空间。
func (s *InMemoryStore) ListNamespaces(ctx context.Context, opts ListOptions) ([][]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// 收集唯一命名空间
	seen := make(map[string]struct{})
	var namespaces [][]string

	for ikey := range s.items {
		if _, ok := seen[ikey.namespace]; ok {
			continue
		}
		seen[ikey.namespace] = struct{}{}

		// 前缀过滤
		if len(opts.Prefix) > 0 {
			prefixStr := namespaceToString(opts.Prefix)
			if !strings.HasPrefix(ikey.namespace, prefixStr) {
				continue
			}
		}

		parts := strings.Split(ikey.namespace, "/")
		ns := make([]string, len(parts))
		copy(ns, parts)
		namespaces = append(namespaces, ns)

		if opts.Limit > 0 && len(namespaces) >= opts.Limit {
			break
		}
	}

	return namespaces, nil
}

// BatchPut 批量存储。
func (s *InMemoryStore) BatchPut(ctx context.Context, items []*StoreItem) error {
	for _, item := range items {
		if err := s.Put(ctx, item.Namespace, item.Key, item.Value); err != nil {
			return fmt.Errorf("batch put key %s: %w", item.Key, err)
		}
	}
	return nil
}

// BatchGet 批量获取。
func (s *InMemoryStore) BatchGet(ctx context.Context, keys []NamespaceKey) ([]*StoreItem, error) {
	results := make([]*StoreItem, len(keys))
	for i, nk := range keys {
		item, err := s.Get(ctx, nk.Namespace, nk.Key)
		if err != nil {
			return nil, fmt.Errorf("batch get key %s: %w", nk.Key, err)
		}
		results[i] = item
	}
	return results, nil
}

// Size 返回当前存储条目数。
func (s *InMemoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// cloneItem 浅拷贝条目（避免外部修改影响内部状态）。
func cloneItem(item *StoreItem) *StoreItem {
	ns := make([]string, len(item.Namespace))
	copy(ns, item.Namespace)
	return &StoreItem{
		Namespace: ns,
		Key:       item.Key,
		Value:     item.Value,
		Score:     item.Score,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

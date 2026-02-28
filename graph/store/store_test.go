package store

import (
	"context"
	"testing"
)

func TestInMemoryStore_PutGet(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	ns := []string{"user", "alice", "memories"}
	err := s.Put(ctx, ns, "pref-1", map[string]any{"language": "Go"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	item, err := s.Get(ctx, ns, "pref-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if item == nil {
		t.Fatal("expected non-nil item")
	}
	if item.Key != "pref-1" {
		t.Errorf("wrong key: %s", item.Key)
	}
}

func TestInMemoryStore_GetNonExistent(t *testing.T) {
	s := NewInMemoryStore()
	item, err := s.Get(context.Background(), []string{"x"}, "missing")
	if err != nil || item != nil {
		t.Errorf("expected nil item and no error for missing key: item=%v err=%v", item, err)
	}
}

func TestInMemoryStore_Update(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()
	ns := []string{"agent", "bot", "state"}

	_ = s.Put(ctx, ns, "counter", 1)
	_ = s.Put(ctx, ns, "counter", 42)

	item, _ := s.Get(ctx, ns, "counter")
	if item.Value != 42 {
		t.Errorf("expected updated value 42, got %v", item.Value)
	}
	if s.Size() != 1 {
		t.Errorf("size should remain 1 after update, got %d", s.Size())
	}
}

func TestInMemoryStore_Delete(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()
	ns := []string{"test"}

	_ = s.Put(ctx, ns, "key1", "val1")
	_ = s.Delete(ctx, ns, "key1")

	item, err := s.Get(ctx, ns, "key1")
	if err != nil || item != nil {
		t.Errorf("deleted item should not exist: item=%v err=%v", item, err)
	}
}

func TestInMemoryStore_Search(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()
	ns := []string{"user", "bob", "memories"}

	_ = s.Put(ctx, ns, "m1", "用户喜欢 Go 语言编程")
	_ = s.Put(ctx, ns, "m2", "用户在金融行业工作")
	_ = s.Put(ctx, ns, "m3", "用户偏好简洁的代码风格")

	results, err := s.Search(ctx, ns, "Go", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one search result for 'Go'")
	}
	t.Logf("found %d results for 'Go'", len(results))
}

func TestInMemoryStore_Search_NoMatch(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()
	ns := []string{"test"}

	_ = s.Put(ctx, ns, "k1", "hello world")

	opts := SearchOptions{K: 10, MinScore: 0.5}
	results, err := s.Search(ctx, ns, "golang", opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	t.Logf("no-match results: %d", len(results))
}

func TestInMemoryStore_List(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()
	ns := []string{"org", "acme", "docs"}

	for i := 0; i < 5; i++ {
		_ = s.Put(ctx, ns, string(rune('a'+i)), i)
	}

	items, err := s.List(ctx, ns, ListOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}
}

func TestInMemoryStore_List_Limit(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()
	ns := []string{"test", "limit"}

	for i := 0; i < 10; i++ {
		_ = s.Put(ctx, ns, string(rune('a'+i)), i)
	}

	items, err := s.List(ctx, ns, ListOptions{Limit: 3})
	if err != nil {
		t.Fatalf("List with limit failed: %v", err)
	}
	if len(items) > 3 {
		t.Errorf("expected max 3 items with limit, got %d", len(items))
	}
}

func TestInMemoryStore_ListNamespaces(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	_ = s.Put(ctx, []string{"user", "alice", "prefs"}, "k1", "v1")
	_ = s.Put(ctx, []string{"user", "bob", "prefs"}, "k2", "v2")
	_ = s.Put(ctx, []string{"agent", "bot", "state"}, "k3", "v3")

	ns, err := s.ListNamespaces(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("ListNamespaces failed: %v", err)
	}
	if len(ns) != 3 {
		t.Errorf("expected 3 namespaces, got %d", len(ns))
	}
}

func TestInMemoryStore_ListNamespaces_PrefixFilter(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	_ = s.Put(ctx, []string{"user", "alice", "prefs"}, "k1", "v1")
	_ = s.Put(ctx, []string{"user", "bob", "prefs"}, "k2", "v2")
	_ = s.Put(ctx, []string{"agent", "bot", "state"}, "k3", "v3")

	ns, err := s.ListNamespaces(ctx, ListOptions{Prefix: []string{"user"}})
	if err != nil {
		t.Fatalf("ListNamespaces with prefix failed: %v", err)
	}
	if len(ns) != 2 {
		t.Errorf("expected 2 user namespaces, got %d", len(ns))
	}
}

func TestInMemoryStore_BatchPutGet(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	items := []*StoreItem{
		{Namespace: []string{"batch"}, Key: "k1", Value: "v1"},
		{Namespace: []string{"batch"}, Key: "k2", Value: "v2"},
		{Namespace: []string{"batch"}, Key: "k3", Value: "v3"},
	}

	err := s.BatchPut(ctx, items)
	if err != nil {
		t.Fatalf("BatchPut failed: %v", err)
	}

	if s.Size() != 3 {
		t.Errorf("expected 3 items after batch put, got %d", s.Size())
	}

	keys := []NamespaceKey{
		{Namespace: []string{"batch"}, Key: "k1"},
		{Namespace: []string{"batch"}, Key: "k3"},
	}

	got, err := s.BatchGet(ctx, keys)
	if err != nil {
		t.Fatalf("BatchGet failed: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("expected 2 items from batch get, got %d", len(got))
	}
	if got[0] == nil || got[0].Value != "v1" {
		t.Errorf("wrong value for k1: %v", got[0])
	}
}

func TestInMemoryStore_ContextCancel(t *testing.T) {
	s := NewInMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Put(ctx, []string{"ns"}, "k", "v")
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestWithStore_GetStore(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	ctx = WithStore(ctx, s)

	retrieved, ok := GetStore(ctx)
	if !ok {
		t.Fatal("GetStore should return true")
	}
	if retrieved == nil {
		t.Fatal("retrieved store should not be nil")
	}

	// 验证是同一个 store
	_ = s.Put(context.Background(), []string{"test"}, "key", "value")
	item, _ := retrieved.Get(context.Background(), []string{"test"}, "key")
	if item == nil || item.Value != "value" {
		t.Error("retrieved store should be the same instance")
	}
}

func TestGetStore_NotSet(t *testing.T) {
	ctx := context.Background()
	_, ok := GetStore(ctx)
	if ok {
		t.Error("GetStore should return false when not set")
	}
}

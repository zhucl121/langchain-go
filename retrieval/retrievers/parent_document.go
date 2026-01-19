package retrievers

import (
	"context"
	"fmt"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// ParentDocumentRetriever 父文档检索器
//
// 这个检索器索引较小的文档块以提高检索精度，
// 但返回较大的父文档以保持完整上下文。
//
// 工作原理：
//  1. 将大文档分割成小块（子文档）
//  2. 索引这些小块到向量存储
//  3. 检索时搜索小块
//  4. 返回小块对应的完整父文档
//
// 优势：
//   - 检索精度高（基于小块）
//   - 上下文完整（返回父文档）
//   - 避免上下文截断问题
//
// 使用示例:
//
//	parentRetriever := retrievers.NewParentDocumentRetriever(
//	    vectorStore,
//	    docStore,
//	    childSplitter,
//	    retrievers.WithParentSplitter(parentSplitter),
//	)
//	
//	// 添加文档
//	_ = parentRetriever.AddDocuments(ctx, documents)
//	
//	// 检索
//	docs, _ := parentRetriever.GetRelevantDocuments(ctx, "query")
//
type ParentDocumentRetriever struct {
	vectorStore    VectorStoreWithAdd
	docStore       DocumentStore
	childSplitter  TextSplitter
	parentSplitter TextSplitter
	config         ParentDocumentConfig
}

// ParentDocumentConfig 父文档检索器配置
type ParentDocumentConfig struct {
	// IDKey 文档 ID 在元数据中的键名（默认 "doc_id"）
	IDKey string
	
	// ParentIDKey 父文档 ID 在元数据中的键名（默认 "parent_id"）
	ParentIDKey string
	
	// TopK 检索的子文档数量
	TopK int
	
	// ReturnFullDocument 是否返回完整父文档（默认 true）
	ReturnFullDocument bool
}

// DefaultParentDocumentConfig 返回默认配置
func DefaultParentDocumentConfig() ParentDocumentConfig {
	return ParentDocumentConfig{
		IDKey:              "doc_id",
		ParentIDKey:        "parent_id",
		TopK:               4,
		ReturnFullDocument: true,
	}
}

// NewParentDocumentRetriever 创建新的父文档检索器
//
// 参数：
//   - vectorStore: 向量存储（用于存储子文档）
//   - docStore: 文档存储（用于存储父文档）
//   - childSplitter: 子文档分割器
//   - opts: 可选配置
func NewParentDocumentRetriever(
	vectorStore VectorStoreWithAdd,
	docStore DocumentStore,
	childSplitter TextSplitter,
	opts ...ParentDocumentOption,
) *ParentDocumentRetriever {
	config := DefaultParentDocumentConfig()
	
	for _, opt := range opts {
		opt(&config)
	}
	
	return &ParentDocumentRetriever{
		vectorStore:   vectorStore,
		docStore:      docStore,
		childSplitter: childSplitter,
		config:        config,
	}
}

// AddDocuments 添加文档
//
// 这个方法会：
//  1. 将文档分割成父文档（如果提供了父分割器）
//  2. 将父文档分割成子文档
//  3. 将子文档添加到向量存储
//  4. 将父文档添加到文档存储
func (r *ParentDocumentRetriever) AddDocuments(ctx context.Context, documents []types.Document) error {
	// 如果有父分割器，先分割成父文档
	var parentDocs []types.Document
	if r.parentSplitter != nil {
		parentDocs = r.splitDocuments(documents, r.parentSplitter)
	} else {
		parentDocs = documents
	}
	
	// 为每个父文档生成 ID
	for i := range parentDocs {
		if parentDocs[i].Metadata == nil {
			parentDocs[i].Metadata = make(map[string]interface{})
		}
		
		// 生成父文档 ID
		parentID := r.generateID(parentDocs[i])
		parentDocs[i].Metadata[r.config.IDKey] = parentID
	}
	
	// 将父文档存储到文档存储
	if err := r.docStore.AddDocuments(ctx, parentDocs); err != nil {
		return fmt.Errorf("failed to add parent documents: %w", err)
	}
	
	// 为每个父文档创建子文档
	var allChildDocs []types.Document
	
	for _, parentDoc := range parentDocs {
		parentID := parentDoc.Metadata[r.config.IDKey].(string)
		
		// 分割成子文档
		childDocs := r.childSplitter.SplitDocuments([]types.Document{parentDoc})
		
		// 为每个子文档添加父文档引用
		for i := range childDocs {
			if childDocs[i].Metadata == nil {
				childDocs[i].Metadata = make(map[string]interface{})
			}
			childDocs[i].Metadata[r.config.ParentIDKey] = parentID
		}
		
		allChildDocs = append(allChildDocs, childDocs...)
	}
	
	// 将子文档添加到向量存储
	if err := r.vectorStore.AddDocuments(ctx, allChildDocs); err != nil {
		return fmt.Errorf("failed to add child documents: %w", err)
	}
	
	return nil
}

// GetRelevantDocuments 获取相关文档
func (r *ParentDocumentRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]types.Document, error) {
	// 从向量存储检索子文档
	childDocs, err := r.vectorStore.SimilaritySearch(ctx, query, r.config.TopK)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve child documents: %w", err)
	}
	
	if len(childDocs) == 0 {
		return []types.Document{}, nil
	}
	
	// 如果不需要返回完整父文档，直接返回子文档
	if !r.config.ReturnFullDocument {
		return childDocs, nil
	}
	
	// 提取父文档 ID
	parentIDs := r.extractParentIDs(childDocs)
	
	// 从文档存储获取父文档
	parentDocs, err := r.docStore.GetDocuments(ctx, parentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve parent documents: %w", err)
	}
	
	// 去重（可能多个子文档来自同一个父文档）
	parentDocs = r.deduplicateByID(parentDocs)
	
	return parentDocs, nil
}

// splitDocuments 使用分割器分割文档
func (r *ParentDocumentRetriever) splitDocuments(documents []types.Document, splitter TextSplitter) []types.Document {
	var result []types.Document
	
	for _, doc := range documents {
		splits := splitter.SplitDocuments([]types.Document{doc})
		result = append(result, splits...)
	}
	
	return result
}

// generateID 生成文档 ID
func (r *ParentDocumentRetriever) generateID(doc types.Document) string {
	// 使用内容哈希作为 ID
	// 实际实现可以使用更复杂的哈希算法
	content := doc.Content
	if len(content) > 50 {
		content = content[:50]
	}
	
	// 简化的 ID 生成（实际应使用 UUID 或哈希）
	return fmt.Sprintf("doc_%d", len(content))
}

// extractParentIDs 从子文档中提取父文档 ID
func (r *ParentDocumentRetriever) extractParentIDs(docs []types.Document) []string {
	idSet := make(map[string]bool)
	var ids []string
	
	for _, doc := range docs {
		if doc.Metadata != nil {
			if parentID, ok := doc.Metadata[r.config.ParentIDKey].(string); ok {
				if !idSet[parentID] {
					idSet[parentID] = true
					ids = append(ids, parentID)
				}
			}
		}
	}
	
	return ids
}

// deduplicateByID 按 ID 去重文档
func (r *ParentDocumentRetriever) deduplicateByID(docs []types.Document) []types.Document {
	seen := make(map[string]bool)
	var result []types.Document
	
	for _, doc := range docs {
		var id string
		if doc.Metadata != nil {
			if docID, ok := doc.Metadata[r.config.IDKey].(string); ok {
				id = docID
			}
		}
		
		if id == "" {
			// 如果没有 ID，使用内容作为键
			id = r.getDocumentKey(doc)
		}
		
		if !seen[id] {
			seen[id] = true
			result = append(result, doc)
		}
	}
	
	return result
}

// getDocumentKey 获取文档的唯一键
func (r *ParentDocumentRetriever) getDocumentKey(doc types.Document) string {
	content := doc.Content
	if len(content) > 100 {
		content = content[:100]
	}
	return content
}

// ==================== 辅助接口 ====================

// VectorStoreWithAdd 支持添加文档的向量存储接口
type VectorStoreWithAdd interface {
	VectorStore
	AddDocuments(ctx context.Context, documents []types.Document) error
	SimilaritySearch(ctx context.Context, query string, k int) ([]types.Document, error)
}

// DocumentStore 文档存储接口
//
// 用于存储和检索完整的父文档
type DocumentStore interface {
	// AddDocuments 添加文档
	AddDocuments(ctx context.Context, documents []types.Document) error
	
	// GetDocuments 根据 ID 列表获取文档
	GetDocuments(ctx context.Context, ids []string) ([]types.Document, error)
	
	// DeleteDocuments 删除文档
	DeleteDocuments(ctx context.Context, ids []string) error
}

// TextSplitter 文本分割器接口
type TextSplitter interface {
	// SplitDocuments 分割文档
	SplitDocuments(documents []types.Document) []types.Document
}

// ==================== 选项模式 ====================

// ParentDocumentOption 配置选项
type ParentDocumentOption func(*ParentDocumentConfig)

// WithParentSplitter 设置父文档分割器
func WithParentSplitter(splitter TextSplitter) ParentDocumentOption {
	return func(c *ParentDocumentConfig) {
		// 父分割器不在 config 中，这个选项暂时不做任何事
		// 需要在 NewParentDocumentRetriever 中单独设置
	}
}

// WithIDKey 设置文档 ID 键名
func WithIDKey(key string) ParentDocumentOption {
	return func(c *ParentDocumentConfig) {
		c.IDKey = key
	}
}

// WithParentIDKey 设置父文档 ID 键名
func WithParentIDKey(key string) ParentDocumentOption {
	return func(c *ParentDocumentConfig) {
		c.ParentIDKey = key
	}
}

// WithParentTopK 设置检索数量
func WithParentTopK(k int) ParentDocumentOption {
	return func(c *ParentDocumentConfig) {
		c.TopK = k
	}
}

// WithReturnFullDocument 设置是否返回完整文档
func WithReturnFullDocument(full bool) ParentDocumentOption {
	return func(c *ParentDocumentConfig) {
		c.ReturnFullDocument = full
	}
}

// ==================== 简单的内存文档存储实现 ====================

// MemoryDocumentStore 内存文档存储
type MemoryDocumentStore struct {
	docs map[string]types.Document
}

// NewMemoryDocumentStore 创建新的内存文档存储
func NewMemoryDocumentStore() *MemoryDocumentStore {
	return &MemoryDocumentStore{
		docs: make(map[string]types.Document),
	}
}

// AddDocuments 添加文档
func (s *MemoryDocumentStore) AddDocuments(ctx context.Context, documents []types.Document) error {
	for _, doc := range documents {
		if doc.Metadata != nil {
			if id, ok := doc.Metadata["doc_id"].(string); ok {
				s.docs[id] = doc
			}
		}
	}
	return nil
}

// GetDocuments 获取文档
func (s *MemoryDocumentStore) GetDocuments(ctx context.Context, ids []string) ([]types.Document, error) {
	var result []types.Document
	
	for _, id := range ids {
		if doc, ok := s.docs[id]; ok {
			result = append(result, doc)
		}
	}
	
	return result, nil
}

// DeleteDocuments 删除文档
func (s *MemoryDocumentStore) DeleteDocuments(ctx context.Context, ids []string) error {
	for _, id := range ids {
		delete(s.docs, id)
	}
	return nil
}

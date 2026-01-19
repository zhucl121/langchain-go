package types

// Document 表示一个文档。
//
// Document 是 RAG 系统中的基本单元，包含内容和元数据。
//
// 示例：
//
//	doc := types.NewDocument("文档内容", map[string]any{
//	    "source": "example.txt",
//	    "page": 1,
//	})
//
type Document struct {
	// Content 文档内容
	Content string `json:"content"`

	// Metadata 文档元数据
	Metadata map[string]any `json:"metadata,omitempty"`

	// Source 文档来源
	Source string `json:"source,omitempty"`

	// ID 文档唯一标识符（可选）
	ID string `json:"id,omitempty"`
}

// NewDocument 创建文档。
//
// 参数：
//   - content: 文档内容
//   - metadata: 元数据
//
// 返回：
//   - *Document: 文档实例
//
func NewDocument(content string, metadata map[string]any) *Document {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return &Document{
		Content:  content,
		Metadata: metadata,
	}
}

// WithSource 设置文档来源。
//
// 参数：
//   - source: 来源标识
//
// 返回：
//   - *Document: 返回自身，支持链式调用
//
func (d *Document) WithSource(source string) *Document {
	d.Source = source
	return d
}

// WithID 设置文档 ID。
//
// 参数：
//   - id: 文档 ID
//
// 返回：
//   - *Document: 返回自身，支持链式调用
//
func (d *Document) WithID(id string) *Document {
	d.ID = id
	return d
}

// AddMetadata 添加元数据。
//
// 参数：
//   - key: 元数据键
//   - value: 元数据值
//
// 返回：
//   - *Document: 返回自身，支持链式调用
//
func (d *Document) AddMetadata(key string, value any) *Document {
	if d.Metadata == nil {
		d.Metadata = make(map[string]any)
	}
	d.Metadata[key] = value
	return d
}

// Clone 创建文档的深拷贝。
//
// 返回：
//   - *Document: 文档副本
//
func (d *Document) Clone() *Document {
	clone := &Document{
		Content: d.Content,
		Source:  d.Source,
		ID:      d.ID,
	}

	// 深拷贝 Metadata
	if d.Metadata != nil {
		clone.Metadata = make(map[string]any, len(d.Metadata))
		for k, v := range d.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}

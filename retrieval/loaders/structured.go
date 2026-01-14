package loaders

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// JSONLoader 是 JSON 文件加载器。
//
// 支持加载 JSON 文件并提取特定字段作为文档内容。
//
type JSONLoader struct {
	*BaseLoader
	contentKey string // JSON 中的内容字段
	jqSchema   string // JQ 查询表达式（简化版）
}

// NewJSONLoader 创建 JSON 加载器。
//
// 参数：
//   - filePath: 文件路径
//
// 返回：
//   - *JSONLoader: JSON 加载器实例
//
func NewJSONLoader(filePath string) *JSONLoader {
	return &JSONLoader{
		BaseLoader: NewBaseLoader(filePath),
		contentKey: "content",
	}
}

// WithContentKey 设置内容字段名。
func (jl *JSONLoader) WithContentKey(key string) *JSONLoader {
	jl.contentKey = key
	return jl
}

// Load 实现 DocumentLoader 接口。
func (jl *JSONLoader) Load(ctx context.Context) ([]*Document, error) {
	// 读取文件
	data, err := os.ReadFile(jl.source)
	if err != nil {
		return nil, fmt.Errorf("json loader: read file failed: %w", err)
	}
	
	// 解析 JSON
	var jsonData any
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("json loader: unmarshal failed: %w", err)
	}
	
	// 提取文档
	docs := jl.extractDocuments(jsonData)
	
	return docs, nil
}

// extractDocuments 从 JSON 数据中提取文档。
func (jl *JSONLoader) extractDocuments(data any) []*Document {
	var docs []*Document
	
	switch v := data.(type) {
	case map[string]any:
		// 单个对象
		doc := jl.extractFromObject(v)
		if doc != nil {
			docs = append(docs, doc)
		}
		
	case []any:
		// 对象数组
		for _, item := range v {
			if obj, ok := item.(map[string]any); ok {
				doc := jl.extractFromObject(obj)
				if doc != nil {
					docs = append(docs, doc)
				}
			}
		}
	}
	
	return docs
}

// extractFromObject 从 JSON 对象中提取文档。
func (jl *JSONLoader) extractFromObject(obj map[string]any) *Document {
	// 获取内容
	content, ok := obj[jl.contentKey]
	if !ok {
		// 如果没有指定的内容字段，将整个对象序列化为内容
		jsonBytes, _ := json.Marshal(obj)
		content = string(jsonBytes)
	}
	
	contentStr := fmt.Sprintf("%v", content)
	
	// 创建元数据（复制所有字段）
	metadata := make(map[string]any)
	for k, v := range obj {
		if k != jl.contentKey {
			metadata[k] = v
		}
	}
	metadata["source"] = jl.source
	metadata["type"] = "json"
	
	doc := NewDocument(contentStr, metadata)
	doc.Source = jl.source
	
	return doc
}

// LoadAndSplit 实现 DocumentLoader 接口。
func (jl *JSONLoader) LoadAndSplit(ctx context.Context) ([]*Document, error) {
	// JSON 加载器已经自然分割为多个文档
	return jl.Load(ctx)
}

// CSVLoader 是 CSV 文件加载器。
//
// 将 CSV 的每一行作为一个文档。
//
type CSVLoader struct {
	*BaseLoader
	contentColumns []string // 用作内容的列
	separator      rune
}

// NewCSVLoader 创建 CSV 加载器。
//
// 参数：
//   - filePath: 文件路径
//
// 返回：
//   - *CSVLoader: CSV 加载器实例
//
func NewCSVLoader(filePath string) *CSVLoader {
	return &CSVLoader{
		BaseLoader:     NewBaseLoader(filePath),
		contentColumns: []string{},
		separator:      ',',
	}
}

// WithContentColumns 设置内容列。
//
// 参数：
//   - columns: 列名列表
//
// 返回：
//   - *CSVLoader: 返回自身
//
func (cl *CSVLoader) WithContentColumns(columns ...string) *CSVLoader {
	cl.contentColumns = columns
	return cl
}

// WithSeparator 设置分隔符。
func (cl *CSVLoader) WithSeparator(sep rune) *CSVLoader {
	cl.separator = sep
	return cl
}

// Load 实现 DocumentLoader 接口。
func (cl *CSVLoader) Load(ctx context.Context) ([]*Document, error) {
	// 打开文件
	file, err := os.Open(cl.source)
	if err != nil {
		return nil, fmt.Errorf("csv loader: open file failed: %w", err)
	}
	defer file.Close()
	
	// 创建 CSV reader
	reader := csv.NewReader(file)
	reader.Comma = cl.separator
	
	// 读取所有记录
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv loader: read csv failed: %w", err)
	}
	
	if len(records) == 0 {
		return []*Document{}, nil
	}
	
	// 第一行作为表头
	headers := records[0]
	
	// 转换为文档
	docs := make([]*Document, 0, len(records)-1)
	for i, record := range records[1:] {
		doc := cl.recordToDocument(headers, record, i)
		docs = append(docs, doc)
	}
	
	return docs, nil
}

// recordToDocument 将 CSV 记录转换为文档。
func (cl *CSVLoader) recordToDocument(headers, record []string, rowNum int) *Document {
	// 创建元数据
	metadata := make(map[string]any)
	metadata["source"] = cl.source
	metadata["row"] = rowNum
	metadata["type"] = "csv"
	
	// 构建内容
	var contentParts []string
	
	for i, header := range headers {
		if i < len(record) {
			value := record[i]
			
			// 添加到元数据
			metadata[header] = value
			
			// 如果指定了内容列，只使用这些列
			if len(cl.contentColumns) > 0 {
				for _, col := range cl.contentColumns {
					if header == col {
						contentParts = append(contentParts, value)
						break
					}
				}
			} else {
				// 否则使用所有列
				contentParts = append(contentParts, fmt.Sprintf("%s: %s", header, value))
			}
		}
	}
	
	content := strings.Join(contentParts, "\n")
	
	doc := NewDocument(content, metadata)
	doc.Source = cl.source
	
	return doc
}

// LoadAndSplit 实现 DocumentLoader 接口。
func (cl *CSVLoader) LoadAndSplit(ctx context.Context) ([]*Document, error) {
	// CSV 加载器已经自然分割为多个文档（每行一个）
	return cl.Load(ctx)
}

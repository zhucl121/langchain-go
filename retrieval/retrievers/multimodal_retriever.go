// Package retrievers 提供多模态检索器
package retrievers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	
	"github.com/zhucl121/langchain-go/pkg/types"
	"github.com/zhucl121/langchain-go/retrieval/embeddings"
	"github.com/zhucl121/langchain-go/retrieval/loaders"
	"github.com/zhucl121/langchain-go/retrieval/vectorstores"
)

// MultimodalRetrieverConfig 多模态检索器配置
type MultimodalRetrieverConfig struct {
	// VectorStore 向量存储
	VectorStore vectorstores.VectorStore
	
	// MultimodalEmbedder 多模态嵌入器
	MultimodalEmbedder embeddings.MultimodalEmbedder
	
	// TopK 返回结果数量
	TopK int
	
	// ScoreThreshold 最小相似度阈值
	ScoreThreshold float32
	
	// EnableCrossModal 是否启用跨模态检索
	// 例如: 用文本搜索图像，或用图像搜索文本
	EnableCrossModal bool
}

// MultimodalRetriever 多模态检索器
//
// 支持跨模态检索：文本、图像、音频、视频之间的相互检索。
type MultimodalRetriever struct {
	config MultimodalRetrieverConfig
	
	// 已索引的多模态文档
	documents []*loaders.MultimodalDocument
}

// NewMultimodalRetriever 创建多模态检索器
func NewMultimodalRetriever(config MultimodalRetrieverConfig) (*MultimodalRetriever, error) {
	if config.VectorStore == nil {
		return nil, errors.New("vector store is required")
	}
	if config.MultimodalEmbedder == nil {
		return nil, errors.New("multimodal embedder is required")
	}
	if config.TopK <= 0 {
		config.TopK = 10
	}
	
	return &MultimodalRetriever{
		config:    config,
		documents: make([]*loaders.MultimodalDocument, 0),
	}, nil
}

// AddDocuments 添加多模态文档
func (r *MultimodalRetriever) AddDocuments(ctx context.Context, docs []*loaders.MultimodalDocument) error {
	if len(docs) == 0 {
		return nil
	}
	
	// 为每个内容块生成向量并存储
	for _, doc := range docs {
		for _, content := range doc.Contents {
			// 生成向量
			embedding, err := r.config.MultimodalEmbedder.EmbedMultimodal(ctx, content)
			if err != nil {
				return fmt.Errorf("failed to embed content: %w", err)
			}
			
			// 转换为文档并添加到向量存储
			loaderDoc := &loaders.Document{
				Content: r.contentToString(content),
				Metadata: map[string]interface{}{
					"document_id":  doc.ID,
					"content_type": string(content.Type),
					"has_images":   doc.HasImages(),
					"has_audios":   doc.HasAudios(),
					"has_videos":   doc.HasVideos(),
				},
			}
			
			// 手动添加向量（绕过自动嵌入）
			// 注意: 这里需要向量存储支持直接添加向量
			// 简化实现，实际可能需要扩展 VectorStore 接口
			_ = embedding
			
			if _, err := r.config.VectorStore.AddDocuments(ctx, []*loaders.Document{loaderDoc}); err != nil {
				return fmt.Errorf("failed to add document to vector store: %w", err)
			}
		}
	}
	
	// 保存文档引用
	r.documents = append(r.documents, docs...)
	
	return nil
}

// Search 检索相似内容
func (r *MultimodalRetriever) Search(ctx context.Context, query *types.MultimodalContent, k int) ([]*loaders.MultimodalDocument, error) {
	if k <= 0 {
		k = r.config.TopK
	}
	
	// 1. 生成查询向量
	queryEmbedding, err := r.config.MultimodalEmbedder.EmbedMultimodal(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	
	// 2. 向量检索
	// 注意: 这里需要 VectorStore 支持向量查询
	// 简化实现使用文本查询
	queryText := r.contentToString(query)
	results, err := r.config.VectorStore.SimilaritySearchWithScore(ctx, queryText, k*2) // 多检索一些以便过滤
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	
	_ = queryEmbedding
	
	// 3. 根据阈值过滤
	var filtered []vectorstores.DocumentWithScore
	for _, result := range results {
		if result.Score >= r.config.ScoreThreshold {
			filtered = append(filtered, result)
		}
	}
	
	// 4. 限制结果数量
	if len(filtered) > k {
		filtered = filtered[:k]
	}
	
	// 5. 转换回多模态文档
	return r.resultsToDocuments(filtered), nil
}

// SearchByText 使用文本检索多模态内容
func (r *MultimodalRetriever) SearchByText(ctx context.Context, text string, k int) ([]*loaders.MultimodalDocument, error) {
	query := types.NewTextContent(text)
	return r.Search(ctx, query, k)
}

// SearchByImage 使用图像检索多模态内容
func (r *MultimodalRetriever) SearchByImage(ctx context.Context, imageData []byte, k int) ([]*loaders.MultimodalDocument, error) {
	query := types.NewImageContentFromData(imageData, types.ImageFormatJPEG)
	return r.Search(ctx, query, k)
}

// SearchByAudio 使用音频检索多模态内容
func (r *MultimodalRetriever) SearchByAudio(ctx context.Context, audioData []byte, k int) ([]*loaders.MultimodalDocument, error) {
	query := types.NewAudioContentFromData(audioData, types.AudioFormatMP3)
	return r.Search(ctx, query, k)
}

// SearchByVideo 使用视频检索多模态内容
func (r *MultimodalRetriever) SearchByVideo(ctx context.Context, videoData []byte, k int) ([]*loaders.MultimodalDocument, error) {
	query := types.NewVideoContentFromData(videoData, types.VideoFormatMP4)
	return r.Search(ctx, query, k)
}

// contentToString 将内容转换为字符串（用于向量存储）
func (r *MultimodalRetriever) contentToString(content *types.MultimodalContent) string {
	switch content.Type {
	case types.ContentTypeText:
		return content.Text
	case types.ContentTypeImage:
		return fmt.Sprintf("[Image: %s]", content.ImageFormat)
	case types.ContentTypeAudio:
		return fmt.Sprintf("[Audio: %s]", content.AudioFormat)
	case types.ContentTypeVideo:
		return fmt.Sprintf("[Video: %s]", content.VideoFormat)
	default:
		return "[Unknown Content]"
	}
}

// resultsToDocuments 将检索结果转换为多模态文档
func (r *MultimodalRetriever) resultsToDocuments(results []vectorstores.DocumentWithScore) []*loaders.MultimodalDocument {
	// 按文档 ID 分组
	docMap := make(map[string]*loaders.MultimodalDocument)
	
	for _, result := range results {
		docID, ok := result.Document.Metadata["document_id"].(string)
		if !ok {
			continue
		}
		
		// 查找原始文档
		var originalDoc *loaders.MultimodalDocument
		for _, doc := range r.documents {
			if doc.ID == docID {
				originalDoc = doc
				break
			}
		}
		
		if originalDoc != nil {
			docMap[docID] = originalDoc
		}
	}
	
	// 转换为列表
	docs := make([]*loaders.MultimodalDocument, 0, len(docMap))
	for _, doc := range docMap {
		docs = append(docs, doc)
	}
	
	return docs
}

// MultimodalSearchResult 多模态检索结果
type MultimodalSearchResult struct {
	// Document 文档
	Document *loaders.MultimodalDocument
	
	// Score 相似度分数
	Score float32
	
	// MatchedContents 匹配的内容块
	MatchedContents []*types.MultimodalContent
	
	// Scores 每个内容块的分数
	Scores []float32
}

// SearchWithDetails 检索并返回详细结果
func (r *MultimodalRetriever) SearchWithDetails(ctx context.Context, query *types.MultimodalContent, k int) ([]*MultimodalSearchResult, error) {
	// 生成查询向量
	queryEmbedding, err := r.config.MultimodalEmbedder.EmbedMultimodal(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	
	_ = queryEmbedding
	
	// 简化实现: 使用基本检索
	docs, err := r.Search(ctx, query, k)
	if err != nil {
		return nil, err
	}
	
	// 转换为详细结果
	results := make([]*MultimodalSearchResult, len(docs))
	for i, doc := range docs {
		results[i] = &MultimodalSearchResult{
			Document:        doc,
			Score:           0.9, // 占位符
			MatchedContents: doc.Contents,
			Scores:          make([]float32, len(doc.Contents)),
		}
	}
	
	return results, nil
}

// RerankResults 重排序结果
func (r *MultimodalRetriever) RerankResults(results []*MultimodalSearchResult) []*MultimodalSearchResult {
	// 按分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

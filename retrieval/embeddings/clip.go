package embeddings

import (
	"context"
	"errors"
	"fmt"
)

// CLIPEmbedder CLIP 多模态嵌入器
//
// CLIP (Contrastive Language-Image Pre-training) 是一个多模态模型，
// 可以将文本和图像映射到同一个向量空间。
type CLIPEmbedder struct {
	textEmbedder  Embeddings
	imageEmbedder ImageEmbedder
	dimension     int
	name          string
}

// NewCLIPEmbedder 创建 CLIP 嵌入器
func NewCLIPEmbedder(
	textEmbed Embeddings,
	imageEmbed ImageEmbedder,
	dimension int,
	name string,
) *CLIPEmbedder {
	return &CLIPEmbedder{
		textEmbedder:  textEmbed,
		imageEmbedder: imageEmbed,
		dimension:     dimension,
		name:          name,
	}
}

func (e *CLIPEmbedder) GetDimension() int {
	return e.dimension
}

func (e *CLIPEmbedder) GetName() string {
	return e.name
}

func (e *CLIPEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if e.textEmbedder == nil {
		return nil, errors.New("text embedder not configured")
	}
	return e.textEmbedder.EmbedQuery(ctx, text)
}

func (e *CLIPEmbedder) EmbedImage(ctx context.Context, imageData []byte) ([]float32, error) {
	if e.imageEmbedder == nil {
		return nil, errors.New("image embedder not configured")
	}
	return e.imageEmbedder.EmbedImage(ctx, imageData)
}

// ComputeSimilarity 计算文本和图像的相似度
//
// 返回值范围: [-1, 1]，值越大表示越相似
func (e *CLIPEmbedder) ComputeSimilarity(ctx context.Context, text string, imageData []byte) (float32, error) {
	textEmbed, err := e.EmbedText(ctx, text)
	if err != nil {
		return 0, fmt.Errorf("failed to embed text: %w", err)
	}
	
	imageEmbed, err := e.EmbedImage(ctx, imageData)
	if err != nil {
		return 0, fmt.Errorf("failed to embed image: %w", err)
	}
	
	return cosineSimilarity(textEmbed, imageEmbed), nil
}

// SearchImageByText 根据文本检索图像
//
// 参数：
//   - ctx: 上下文
//   - text: 查询文本
//   - imageEmbeddings: 候选图像的向量
//   - k: 返回数量
//
// 返回：
//   - []int: Top-k 图像的索引
//   - []float64: 对应的相似度分数
//   - error: 错误
func (e *CLIPEmbedder) SearchImageByText(
	ctx context.Context,
	text string,
	imageEmbeddings [][]float32,
	k int,
) ([]int, []float32, error) {
	// 嵌入文本
	textEmbed, err := e.EmbedText(ctx, text)
	if err != nil {
		return nil, nil, err
	}
	
	// 计算相似度
	type scored struct {
		index int
		score float32
	}
	
	scores := make([]scored, len(imageEmbeddings))
	for i, imageEmbed := range imageEmbeddings {
		scores[i] = scored{
			index: i,
			score: cosineSimilarity(textEmbed, imageEmbed),
		}
	}
	
	// 排序
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score < scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	// 取 Top-k
	if k > len(scores) {
		k = len(scores)
	}
	
	indices := make([]int, k)
	similarities := make([]float32, k)
	for i := 0; i < k; i++ {
		indices[i] = scores[i].index
		similarities[i] = scores[i].score
	}
	
	return indices, similarities, nil
}

// SearchTextByImage 根据图像检索文本
func (e *CLIPEmbedder) SearchTextByImage(
	ctx context.Context,
	imageData []byte,
	textEmbeddings [][]float32,
	k int,
) ([]int, []float32, error) {
	// 嵌入图像
	imageEmbed, err := e.EmbedImage(ctx, imageData)
	if err != nil {
		return nil, nil, err
	}
	
	// 计算相似度
	type scored struct {
		index int
		score float32
	}
	
	scores := make([]scored, len(textEmbeddings))
	for i, textEmbed := range textEmbeddings {
		scores[i] = scored{
			index: i,
			score: cosineSimilarity(imageEmbed, textEmbed),
		}
	}
	
	// 排序
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score < scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	// 取 Top-k
	if k > len(scores) {
		k = len(scores)
	}
	
	indices := make([]int, k)
	similarities := make([]float32, k)
	for i := 0; i < k; i++ {
		indices[i] = scores[i].index
		similarities[i] = scores[i].score
	}
	
	return indices, similarities, nil
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	
	var dotProduct float32
	var normA float32
	var normB float32
	
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (normA * normB)
}

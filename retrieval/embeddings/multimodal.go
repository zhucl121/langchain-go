// Package embeddings 提供多模态嵌入器接口
package embeddings

import (
	"context"
	
	"github.com/zhucl121/langchain-go/pkg/types"
)

// MultimodalEmbedder 多模态嵌入器接口
//
// 支持文本、图像、音频、视频等多种模态的向量嵌入。
type MultimodalEmbedder interface {
	// EmbedText 对文本进行向量化
	EmbedText(ctx context.Context, text string) ([]float32, error)
	
	// EmbedImage 对图像进行向量化
	EmbedImage(ctx context.Context, imageData []byte) ([]float32, error)
	
	// EmbedAudio 对音频进行向量化
	EmbedAudio(ctx context.Context, audioData []byte) ([]float32, error)
	
	// EmbedVideo 对视频进行向量化
	EmbedVideo(ctx context.Context, videoData []byte) ([]float32, error)
	
	// EmbedMultimodal 对多模态内容进行向量化
	EmbedMultimodal(ctx context.Context, content *types.MultimodalContent) ([]float32, error)
	
	// GetDimension 返回嵌入向量的维度
	GetDimension() int
	
	// GetName 返回嵌入器名称
	GetName() string
	
	// SupportsModality 是否支持指定模态
	SupportsModality(contentType types.ContentType) bool
}

// ImageEmbedder 图像嵌入器接口
type ImageEmbedder interface {
	// EmbedImage 对图像进行向量化
	EmbedImage(ctx context.Context, imageData []byte) ([]float32, error)
	
	// EmbedImageBatch 批量对图像进行向量化
	EmbedImageBatch(ctx context.Context, images [][]byte) ([][]float32, error)
	
	// GetDimension 返回嵌入向量的维度
	GetDimension() int
	
	// GetName 返回嵌入器名称
	GetName() string
}

// AudioEmbedder 音频嵌入器接口
type AudioEmbedder interface {
	// EmbedAudio 对音频进行向量化
	EmbedAudio(ctx context.Context, audioData []byte) ([]float32, error)
	
	// EmbedAudioBatch 批量对音频进行向量化
	EmbedAudioBatch(ctx context.Context, audios [][]byte) ([][]float32, error)
	
	// GetDimension 返回嵌入向量的维度
	GetDimension() int
	
	// GetName 返回嵌入器名称
	GetName() string
}

// VideoEmbedder 视频嵌入器接口
type VideoEmbedder interface {
	// EmbedVideo 对视频进行向量化
	EmbedVideo(ctx context.Context, videoData []byte) ([]float32, error)
	
	// EmbedVideoBatch 批量对视频进行向量化
	EmbedVideoBatch(ctx context.Context, videos [][]byte) ([][]float32, error)
	
	// GetDimension 返回嵌入向量的维度
	GetDimension() int
	
	// GetName 返回嵌入器名称
	GetName() string
}

// BaseMultimodalEmbedder 基础多模态嵌入器
//
// 组合不同的单模态嵌入器来实现多模态支持。
type BaseMultimodalEmbedder struct {
	textEmbedder  Embeddings     // 文本嵌入器
	imageEmbedder ImageEmbedder  // 图像嵌入器
	audioEmbedder AudioEmbedder  // 音频嵌入器
	videoEmbedder VideoEmbedder  // 视频嵌入器
	
	// 统一的嵌入维度（所有模态必须输出相同维度）
	dimension int
	name      string
}

// NewBaseMultimodalEmbedder 创建基础多模态嵌入器
func NewBaseMultimodalEmbedder(
	textEmbed Embeddings,
	imageEmbed ImageEmbedder,
	audioEmbed AudioEmbedder,
	videoEmbed VideoEmbedder,
	dimension int,
	name string,
) *BaseMultimodalEmbedder {
	return &BaseMultimodalEmbedder{
		textEmbedder:  textEmbed,
		imageEmbedder: imageEmbed,
		audioEmbedder: audioEmbed,
		videoEmbedder: videoEmbed,
		dimension:     dimension,
		name:          name,
	}
}

func (e *BaseMultimodalEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if e.textEmbedder == nil {
		return nil, ErrUnsupportedModality("text")
	}
	return e.textEmbedder.EmbedQuery(ctx, text)
}

func (e *BaseMultimodalEmbedder) EmbedImage(ctx context.Context, imageData []byte) ([]float32, error) {
	if e.imageEmbedder == nil {
		return nil, ErrUnsupportedModality("image")
	}
	return e.imageEmbedder.EmbedImage(ctx, imageData)
}

func (e *BaseMultimodalEmbedder) EmbedAudio(ctx context.Context, audioData []byte) ([]float32, error) {
	if e.audioEmbedder == nil {
		return nil, ErrUnsupportedModality("audio")
	}
	return e.audioEmbedder.EmbedAudio(ctx, audioData)
}

func (e *BaseMultimodalEmbedder) EmbedVideo(ctx context.Context, videoData []byte) ([]float32, error) {
	if e.videoEmbedder == nil {
		return nil, ErrUnsupportedModality("video")
	}
	return e.videoEmbedder.EmbedVideo(ctx, videoData)
}

func (e *BaseMultimodalEmbedder) EmbedMultimodal(ctx context.Context, content *types.MultimodalContent) ([]float32, error) {
	switch content.Type {
	case types.ContentTypeText:
		return e.EmbedText(ctx, content.Text)
	case types.ContentTypeImage:
		data, err := content.GetImageData()
		if err != nil {
			return nil, err
		}
		return e.EmbedImage(ctx, data)
	case types.ContentTypeAudio:
		data, err := content.GetAudioData()
		if err != nil {
			return nil, err
		}
		return e.EmbedAudio(ctx, data)
	case types.ContentTypeVideo:
		data, err := content.GetVideoData()
		if err != nil {
			return nil, err
		}
		return e.EmbedVideo(ctx, data)
	default:
		return nil, ErrUnsupportedModality(string(content.Type))
	}
}

func (e *BaseMultimodalEmbedder) GetDimension() int {
	return e.dimension
}

func (e *BaseMultimodalEmbedder) GetName() string {
	return e.name
}

func (e *BaseMultimodalEmbedder) SupportsModality(contentType types.ContentType) bool {
	switch contentType {
	case types.ContentTypeText:
		return e.textEmbedder != nil
	case types.ContentTypeImage:
		return e.imageEmbedder != nil
	case types.ContentTypeAudio:
		return e.audioEmbedder != nil
	case types.ContentTypeVideo:
		return e.videoEmbedder != nil
	default:
		return false
	}
}

// ErrUnsupportedModality 不支持的模态错误
func ErrUnsupportedModality(modality string) error {
	return &UnsupportedModalityError{Modality: modality}
}

// UnsupportedModalityError 不支持的模态错误类型
type UnsupportedModalityError struct {
	Modality string
}

func (e *UnsupportedModalityError) Error() string {
	return "unsupported modality: " + e.Modality
}

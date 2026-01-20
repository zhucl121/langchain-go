package loaders

import (
	"github.com/zhucl121/langchain-go/pkg/types"
)

// MultimodalDocument 多模态文档
//
// 扩展 Document 类型以支持多模态内容。
type MultimodalDocument struct {
	// ID 文档 ID
	ID string
	
	// Contents 多个内容块
	Contents []*types.MultimodalContent
	
	// Metadata 元数据
	Metadata map[string]interface{}
}

// NewMultimodalDocument 创建多模态文档
func NewMultimodalDocument(id string, contents ...*types.MultimodalContent) *MultimodalDocument {
	return &MultimodalDocument{
		ID:       id,
		Contents: contents,
		Metadata: make(map[string]interface{}),
	}
}

// AddContent 添加内容
func (d *MultimodalDocument) AddContent(content *types.MultimodalContent) {
	d.Contents = append(d.Contents, content)
}

// GetTextContents 获取所有文本内容
func (d *MultimodalDocument) GetTextContents() []string {
	var texts []string
	for _, content := range d.Contents {
		if content.IsText() {
			texts = append(texts, content.Text)
		}
	}
	return texts
}

// GetImageContents 获取所有图像内容
func (d *MultimodalDocument) GetImageContents() []*types.MultimodalContent {
	var images []*types.MultimodalContent
	for _, content := range d.Contents {
		if content.IsImage() {
			images = append(images, content)
		}
	}
	return images
}

// GetAudioContents 获取所有音频内容
func (d *MultimodalDocument) GetAudioContents() []*types.MultimodalContent {
	var audios []*types.MultimodalContent
	for _, content := range d.Contents {
		if content.IsAudio() {
			audios = append(audios, content)
		}
	}
	return audios
}

// GetVideoContents 获取所有视频内容
func (d *MultimodalDocument) GetVideoContents() []*types.MultimodalContent {
	var videos []*types.MultimodalContent
	for _, content := range d.Contents {
		if content.IsVideo() {
			videos = append(videos, content)
		}
	}
	return videos
}

// HasImages 是否包含图像
func (d *MultimodalDocument) HasImages() bool {
	for _, content := range d.Contents {
		if content.IsImage() {
			return true
		}
	}
	return false
}

// HasAudios 是否包含音频
func (d *MultimodalDocument) HasAudios() bool {
	for _, content := range d.Contents {
		if content.IsAudio() {
			return true
		}
	}
	return false
}

// HasVideos 是否包含视频
func (d *MultimodalDocument) HasVideos() bool {
	for _, content := range d.Contents {
		if content.IsVideo() {
			return true
		}
	}
	return false
}

// ToDocument 转换为普通 Document（仅保留文本）
func (d *MultimodalDocument) ToDocument() *Document {
	texts := d.GetTextContents()
	content := ""
	if len(texts) > 0 {
		// 合并所有文本
		for i, text := range texts {
			if i > 0 {
				content += "\n"
			}
			content += text
		}
	}
	
	return &Document{
		Content:  content,
		Metadata: d.Metadata,
	}
}

// ContentCount 返回内容数量
func (d *MultimodalDocument) ContentCount() int {
	return len(d.Contents)
}

// TotalSize 返回总大小（字节）
func (d *MultimodalDocument) TotalSize() int {
	total := 0
	for _, content := range d.Contents {
		total += content.Size()
	}
	return total
}

// GetContent 获取指定索引的内容
func (d *MultimodalDocument) GetContent(index int) *types.MultimodalContent {
	if index < 0 || index >= len(d.Contents) {
		return nil
	}
	return d.Contents[index]
}

// FilterByType 按类型过滤内容
func (d *MultimodalDocument) FilterByType(contentType types.ContentType) []*types.MultimodalContent {
	var filtered []*types.MultimodalContent
	for _, content := range d.Contents {
		if content.Type == contentType {
			filtered = append(filtered, content)
		}
	}
	return filtered
}

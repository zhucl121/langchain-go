// Package types 提供多模态内容类型定义
package types

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ContentType 内容类型
type ContentType string

const (
	// ContentTypeText 文本内容
	ContentTypeText ContentType = "text"
	
	// ContentTypeImage 图像内容
	ContentTypeImage ContentType = "image"
	
	// ContentTypeAudio 音频内容
	ContentTypeAudio ContentType = "audio"
	
	// ContentTypeVideo 视频内容
	ContentTypeVideo ContentType = "video"
	
	// ContentTypeFile 文件内容
	ContentTypeFile ContentType = "file"
)

// ImageFormat 图像格式
type ImageFormat string

const (
	ImageFormatJPEG ImageFormat = "jpeg"
	ImageFormatPNG  ImageFormat = "png"
	ImageFormatGIF  ImageFormat = "gif"
	ImageFormatWebP ImageFormat = "webp"
	ImageFormatBMP  ImageFormat = "bmp"
)

// AudioFormat 音频格式
type AudioFormat string

const (
	AudioFormatMP3  AudioFormat = "mp3"
	AudioFormatWAV  AudioFormat = "wav"
	AudioFormatFLAC AudioFormat = "flac"
	AudioFormatM4A  AudioFormat = "m4a"
	AudioFormatOGG  AudioFormat = "ogg"
)

// VideoFormat 视频格式
type VideoFormat string

const (
	VideoFormatMP4  VideoFormat = "mp4"
	VideoFormatAVI  VideoFormat = "avi"
	VideoFormatMKV  VideoFormat = "mkv"
	VideoFormatMOV  VideoFormat = "mov"
	VideoFormatWebM VideoFormat = "webm"
)

// MultimodalContent 多模态内容
//
// 支持文本、图像、音频、视频等多种模态的内容。
type MultimodalContent struct {
	// Type 内容类型
	Type ContentType `json:"type"`
	
	// Text 文本内容（当 Type 为 text 时）
	Text string `json:"text,omitempty"`
	
	// ImageURL 图像 URL（当 Type 为 image 时）
	ImageURL string `json:"image_url,omitempty"`
	
	// ImageData 图像数据（Base64 或原始字节）
	ImageData []byte `json:"image_data,omitempty"`
	
	// ImageFormat 图像格式
	ImageFormat ImageFormat `json:"image_format,omitempty"`
	
	// AudioURL 音频 URL（当 Type 为 audio 时）
	AudioURL string `json:"audio_url,omitempty"`
	
	// AudioData 音频数据
	AudioData []byte `json:"audio_data,omitempty"`
	
	// AudioFormat 音频格式
	AudioFormat AudioFormat `json:"audio_format,omitempty"`
	
	// VideoURL 视频 URL（当 Type 为 video 时）
	VideoURL string `json:"video_url,omitempty"`
	
	// VideoData 视频数据
	VideoData []byte `json:"video_data,omitempty"`
	
	// VideoFormat 视频格式
	VideoFormat VideoFormat `json:"video_format,omitempty"`
	
	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewTextContent 创建文本内容
func NewTextContent(text string) *MultimodalContent {
	return &MultimodalContent{
		Type: ContentTypeText,
		Text: text,
	}
}

// NewImageContent 创建图像内容（从 URL）
func NewImageContent(imageURL string, format ImageFormat) *MultimodalContent {
	return &MultimodalContent{
		Type:        ContentTypeImage,
		ImageURL:    imageURL,
		ImageFormat: format,
	}
}

// NewImageContentFromData 创建图像内容（从数据）
func NewImageContentFromData(imageData []byte, format ImageFormat) *MultimodalContent {
	return &MultimodalContent{
		Type:        ContentTypeImage,
		ImageData:   imageData,
		ImageFormat: format,
	}
}

// NewImageContentFromFile 创建图像内容（从文件）
func NewImageContentFromFile(filePath string) (*MultimodalContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}
	
	// 推断格式
	format := inferImageFormat(filePath)
	
	return &MultimodalContent{
		Type:        ContentTypeImage,
		ImageData:   data,
		ImageFormat: format,
		Metadata: map[string]interface{}{
			"file_path": filePath,
		},
	}, nil
}

// NewAudioContent 创建音频内容（从 URL）
func NewAudioContent(audioURL string, format AudioFormat) *MultimodalContent {
	return &MultimodalContent{
		Type:        ContentTypeAudio,
		AudioURL:    audioURL,
		AudioFormat: format,
	}
}

// NewAudioContentFromData 创建音频内容（从数据）
func NewAudioContentFromData(audioData []byte, format AudioFormat) *MultimodalContent {
	return &MultimodalContent{
		Type:        ContentTypeAudio,
		AudioData:   audioData,
		AudioFormat: format,
	}
}

// NewAudioContentFromFile 创建音频内容（从文件）
func NewAudioContentFromFile(filePath string) (*MultimodalContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}
	
	format := inferAudioFormat(filePath)
	
	return &MultimodalContent{
		Type:        ContentTypeAudio,
		AudioData:   data,
		AudioFormat: format,
		Metadata: map[string]interface{}{
			"file_path": filePath,
		},
	}, nil
}

// NewVideoContent 创建视频内容（从 URL）
func NewVideoContent(videoURL string, format VideoFormat) *MultimodalContent {
	return &MultimodalContent{
		Type:        ContentTypeVideo,
		VideoURL:    videoURL,
		VideoFormat: format,
	}
}

// NewVideoContentFromData 创建视频内容（从数据）
func NewVideoContentFromData(videoData []byte, format VideoFormat) *MultimodalContent {
	return &MultimodalContent{
		Type:        ContentTypeVideo,
		VideoData:   videoData,
		VideoFormat: format,
	}
}

// NewVideoContentFromFile 创建视频内容（从文件）
func NewVideoContentFromFile(filePath string) (*MultimodalContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read video file: %w", err)
	}
	
	format := inferVideoFormat(filePath)
	
	return &MultimodalContent{
		Type:        ContentTypeVideo,
		VideoData:   data,
		VideoFormat: format,
		Metadata: map[string]interface{}{
			"file_path": filePath,
		},
	}, nil
}

// IsText 是否为文本内容
func (c *MultimodalContent) IsText() bool {
	return c.Type == ContentTypeText
}

// IsImage 是否为图像内容
func (c *MultimodalContent) IsImage() bool {
	return c.Type == ContentTypeImage
}

// IsAudio 是否为音频内容
func (c *MultimodalContent) IsAudio() bool {
	return c.Type == ContentTypeAudio
}

// IsVideo 是否为视频内容
func (c *MultimodalContent) IsVideo() bool {
	return c.Type == ContentTypeVideo
}

// GetText 获取文本内容
func (c *MultimodalContent) GetText() (string, error) {
	if !c.IsText() {
		return "", errors.New("content is not text")
	}
	return c.Text, nil
}

// GetImageData 获取图像数据
func (c *MultimodalContent) GetImageData() ([]byte, error) {
	if !c.IsImage() {
		return nil, errors.New("content is not image")
	}
	
	if len(c.ImageData) > 0 {
		return c.ImageData, nil
	}
	
	if c.ImageURL != "" {
		return nil, errors.New("image URL provided but data not loaded")
	}
	
	return nil, errors.New("no image data available")
}

// GetImageDataBase64 获取 Base64 编码的图像数据
func (c *MultimodalContent) GetImageDataBase64() (string, error) {
	data, err := c.GetImageData()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// GetAudioData 获取音频数据
func (c *MultimodalContent) GetAudioData() ([]byte, error) {
	if !c.IsAudio() {
		return nil, errors.New("content is not audio")
	}
	
	if len(c.AudioData) > 0 {
		return c.AudioData, nil
	}
	
	if c.AudioURL != "" {
		return nil, errors.New("audio URL provided but data not loaded")
	}
	
	return nil, errors.New("no audio data available")
}

// GetVideoData 获取视频数据
func (c *MultimodalContent) GetVideoData() ([]byte, error) {
	if !c.IsVideo() {
		return nil, errors.New("content is not video")
	}
	
	if len(c.VideoData) > 0 {
		return c.VideoData, nil
	}
	
	if c.VideoURL != "" {
		return nil, errors.New("video URL provided but data not loaded")
	}
	
	return nil, errors.New("no video data available")
}

// Size 返回内容大小（字节）
func (c *MultimodalContent) Size() int {
	switch c.Type {
	case ContentTypeText:
		return len(c.Text)
	case ContentTypeImage:
		return len(c.ImageData)
	case ContentTypeAudio:
		return len(c.AudioData)
	case ContentTypeVideo:
		return len(c.VideoData)
	default:
		return 0
	}
}

// LoadFromReader 从 Reader 加载数据
func (c *MultimodalContent) LoadFromReader(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}
	
	switch c.Type {
	case ContentTypeImage:
		c.ImageData = data
	case ContentTypeAudio:
		c.AudioData = data
	case ContentTypeVideo:
		c.VideoData = data
	default:
		return fmt.Errorf("unsupported content type for LoadFromReader: %s", c.Type)
	}
	
	return nil
}

// inferImageFormat 从文件扩展名推断图像格式
func inferImageFormat(filePath string) ImageFormat {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		return ImageFormatJPEG
	case ".png":
		return ImageFormatPNG
	case ".gif":
		return ImageFormatGIF
	case ".webp":
		return ImageFormatWebP
	case ".bmp":
		return ImageFormatBMP
	default:
		return ImageFormatJPEG // 默认
	}
}

// inferAudioFormat 从文件扩展名推断音频格式
func inferAudioFormat(filePath string) AudioFormat {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".mp3":
		return AudioFormatMP3
	case ".wav":
		return AudioFormatWAV
	case ".flac":
		return AudioFormatFLAC
	case ".m4a":
		return AudioFormatM4A
	case ".ogg":
		return AudioFormatOGG
	default:
		return AudioFormatMP3 // 默认
	}
}

// inferVideoFormat 从文件扩展名推断视频格式
func inferVideoFormat(filePath string) VideoFormat {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".mp4":
		return VideoFormatMP4
	case ".avi":
		return VideoFormatAVI
	case ".mkv":
		return VideoFormatMKV
	case ".mov":
		return VideoFormatMOV
	case ".webm":
		return VideoFormatWebM
	default:
		return VideoFormatMP4 // 默认
	}
}

// MultimodalMessage 多模态消息
//
// 扩展 Message 类型以支持多模态内容。
type MultimodalMessage struct {
	// Role 消息角色
	Role string `json:"role"`
	
	// Contents 多个内容块
	Contents []*MultimodalContent `json:"contents"`
	
	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewMultimodalMessage 创建多模态消息
func NewMultimodalMessage(role string, contents ...*MultimodalContent) *MultimodalMessage {
	return &MultimodalMessage{
		Role:     role,
		Contents: contents,
		Metadata: make(map[string]interface{}),
	}
}

// AddContent 添加内容
func (m *MultimodalMessage) AddContent(content *MultimodalContent) {
	m.Contents = append(m.Contents, content)
}

// GetTextContents 获取所有文本内容
func (m *MultimodalMessage) GetTextContents() []string {
	var texts []string
	for _, content := range m.Contents {
		if content.IsText() {
			texts = append(texts, content.Text)
		}
	}
	return texts
}

// GetImageContents 获取所有图像内容
func (m *MultimodalMessage) GetImageContents() []*MultimodalContent {
	var images []*MultimodalContent
	for _, content := range m.Contents {
		if content.IsImage() {
			images = append(images, content)
		}
	}
	return images
}

// GetAudioContents 获取所有音频内容
func (m *MultimodalMessage) GetAudioContents() []*MultimodalContent {
	var audios []*MultimodalContent
	for _, content := range m.Contents {
		if content.IsAudio() {
			audios = append(audios, content)
		}
	}
	return audios
}

// GetVideoContents 获取所有视频内容
func (m *MultimodalMessage) GetVideoContents() []*MultimodalContent {
	var videos []*MultimodalContent
	for _, content := range m.Contents {
		if content.IsVideo() {
			videos = append(videos, content)
		}
	}
	return videos
}

// HasImages 是否包含图像
func (m *MultimodalMessage) HasImages() bool {
	for _, content := range m.Contents {
		if content.IsImage() {
			return true
		}
	}
	return false
}

// HasAudios 是否包含音频
func (m *MultimodalMessage) HasAudios() bool {
	for _, content := range m.Contents {
		if content.IsAudio() {
			return true
		}
	}
	return false
}

// HasVideos 是否包含视频
func (m *MultimodalMessage) HasVideos() bool {
	for _, content := range m.Contents {
		if content.IsVideo() {
			return true
		}
	}
	return false
}

// ToMessage 转换为普通 Message（仅保留文本）
func (m *MultimodalMessage) ToMessage() *Message {
	texts := m.GetTextContents()
	content := ""
	if len(texts) > 0 {
		content = texts[0] // 取第一个文本
	}
	
	// 将字符串角色转换为 Role 类型
	role := RoleUser
	switch m.Role {
	case "user":
		role = RoleUser
	case "assistant":
		role = RoleAssistant
	case "system":
		role = RoleSystem
	case "function", "tool":
		role = RoleTool
	}
	
	return &Message{
		Role:    role,
		Content: content,
	}
}

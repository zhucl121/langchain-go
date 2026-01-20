package types

import (
	"testing"
)

func TestNewTextContent(t *testing.T) {
	text := "Hello, world!"
	content := NewTextContent(text)
	
	if content.Type != ContentTypeText {
		t.Errorf("Type = %v, want %v", content.Type, ContentTypeText)
	}
	if content.Text != text {
		t.Errorf("Text = %v, want %v", content.Text, text)
	}
	if !content.IsText() {
		t.Error("IsText() = false, want true")
	}
}

func TestNewImageContent(t *testing.T) {
	url := "https://example.com/image.jpg"
	content := NewImageContent(url, ImageFormatJPEG)
	
	if content.Type != ContentTypeImage {
		t.Errorf("Type = %v, want %v", content.Type, ContentTypeImage)
	}
	if content.ImageURL != url {
		t.Errorf("ImageURL = %v, want %v", content.ImageURL, url)
	}
	if content.ImageFormat != ImageFormatJPEG {
		t.Errorf("ImageFormat = %v, want %v", content.ImageFormat, ImageFormatJPEG)
	}
	if !content.IsImage() {
		t.Error("IsImage() = false, want true")
	}
}

func TestNewImageContentFromData(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG 头部
	content := NewImageContentFromData(data, ImageFormatJPEG)
	
	if content.Type != ContentTypeImage {
		t.Errorf("Type = %v, want %v", content.Type, ContentTypeImage)
	}
	if len(content.ImageData) != len(data) {
		t.Errorf("ImageData length = %d, want %d", len(content.ImageData), len(data))
	}
	
	// 测试获取图像数据
	retrievedData, err := content.GetImageData()
	if err != nil {
		t.Fatalf("GetImageData() error = %v", err)
	}
	if len(retrievedData) != len(data) {
		t.Errorf("retrieved data length = %d, want %d", len(retrievedData), len(data))
	}
}

func TestMultimodalContent_Size(t *testing.T) {
	tests := []struct {
		name    string
		content *MultimodalContent
		want    int
	}{
		{
			name:    "text content",
			content: NewTextContent("Hello"),
			want:    5,
		},
		{
			name:    "image content",
			content: NewImageContentFromData([]byte{1, 2, 3}, ImageFormatJPEG),
			want:    3,
		},
		{
			name: "audio content",
			content: NewAudioContentFromData([]byte{1, 2, 3, 4}, AudioFormatMP3),
			want: 4,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.content.Size(); got != tt.want {
				t.Errorf("Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultimodalMessage(t *testing.T) {
	text := NewTextContent("Hello")
	image := NewImageContent("image.jpg", ImageFormatJPEG)
	audio := NewAudioContent("audio.mp3", AudioFormatMP3)
	
	message := NewMultimodalMessage("user", text, image, audio)
	
	// 测试角色
	if message.Role != "user" {
		t.Errorf("Role = %v, want user", message.Role)
	}
	
	// 测试内容数量
	if len(message.Contents) != 3 {
		t.Errorf("Contents length = %d, want 3", len(message.Contents))
	}
	
	// 测试文本内容
	texts := message.GetTextContents()
	if len(texts) != 1 {
		t.Errorf("text contents length = %d, want 1", len(texts))
	}
	if texts[0] != "Hello" {
		t.Errorf("text = %v, want Hello", texts[0])
	}
	
	// 测试图像内容
	images := message.GetImageContents()
	if len(images) != 1 {
		t.Errorf("image contents length = %d, want 1", len(images))
	}
	
	// 测试音频内容
	audios := message.GetAudioContents()
	if len(audios) != 1 {
		t.Errorf("audio contents length = %d, want 1", len(audios))
	}
	
	// 测试 Has 方法
	if !message.HasImages() {
		t.Error("HasImages() = false, want true")
	}
	if !message.HasAudios() {
		t.Error("HasAudios() = false, want true")
	}
	if message.HasVideos() {
		t.Error("HasVideos() = true, want false")
	}
}

func TestMultimodalMessage_AddContent(t *testing.T) {
	message := NewMultimodalMessage("user")
	
	if len(message.Contents) != 0 {
		t.Errorf("initial contents length = %d, want 0", len(message.Contents))
	}
	
	message.AddContent(NewTextContent("Hello"))
	if len(message.Contents) != 1 {
		t.Errorf("contents length after add = %d, want 1", len(message.Contents))
	}
	
	message.AddContent(NewImageContent("image.jpg", ImageFormatJPEG))
	if len(message.Contents) != 2 {
		t.Errorf("contents length after add = %d, want 2", len(message.Contents))
	}
}

func TestMultimodalMessage_ToMessage(t *testing.T) {
	text1 := NewTextContent("First")
	text2 := NewTextContent("Second")
	image := NewImageContent("image.jpg", ImageFormatJPEG)
	
	message := NewMultimodalMessage("user", text1, image, text2)
	
	// 转换为普通消息
	msg := message.ToMessage()
	
	if msg.Role != "user" {
		t.Errorf("Role = %v, want user", msg.Role)
	}
	
	// 应该只保留第一个文本
	if msg.Content != "First" {
		t.Errorf("Content = %v, want First", msg.Content)
	}
}

func TestInferImageFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     ImageFormat
	}{
		{"image.jpg", ImageFormatJPEG},
		{"image.jpeg", ImageFormatJPEG},
		{"image.png", ImageFormatPNG},
		{"image.gif", ImageFormatGIF},
		{"image.webp", ImageFormatWebP},
		{"image.bmp", ImageFormatBMP},
		{"image.unknown", ImageFormatJPEG}, // 默认
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := inferImageFormat(tt.filename)
			if got != tt.want {
				t.Errorf("inferImageFormat(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestInferAudioFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     AudioFormat
	}{
		{"audio.mp3", AudioFormatMP3},
		{"audio.wav", AudioFormatWAV},
		{"audio.flac", AudioFormatFLAC},
		{"audio.m4a", AudioFormatM4A},
		{"audio.ogg", AudioFormatOGG},
		{"audio.unknown", AudioFormatMP3}, // 默认
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := inferAudioFormat(tt.filename)
			if got != tt.want {
				t.Errorf("inferAudioFormat(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestInferVideoFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     VideoFormat
	}{
		{"video.mp4", VideoFormatMP4},
		{"video.avi", VideoFormatAVI},
		{"video.mkv", VideoFormatMKV},
		{"video.mov", VideoFormatMOV},
		{"video.webm", VideoFormatWebM},
		{"video.unknown", VideoFormatMP4}, // 默认
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := inferVideoFormat(tt.filename)
			if got != tt.want {
				t.Errorf("inferVideoFormat(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestMultimodalContent_GetImageDataBase64(t *testing.T) {
	data := []byte("test image data")
	content := NewImageContentFromData(data, ImageFormatJPEG)
	
	base64Str, err := content.GetImageDataBase64()
	if err != nil {
		t.Fatalf("GetImageDataBase64() error = %v", err)
	}
	
	if base64Str == "" {
		t.Error("GetImageDataBase64() returned empty string")
	}
	
	// 验证 base64 编码
	// "test image data" 的 base64 是 "dGVzdCBpbWFnZSBkYXRh"
	expected := "dGVzdCBpbWFnZSBkYXRh"
	if base64Str != expected {
		t.Errorf("GetImageDataBase64() = %v, want %v", base64Str, expected)
	}
}

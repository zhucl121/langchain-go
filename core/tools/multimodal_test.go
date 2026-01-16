package tools

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ============================================
// 图像分析工具测试
// ============================================

func TestImageAnalysisTool_GetName(t *testing.T) {
	tool := NewImageAnalysisTool(nil)
	if tool.GetName() != "image_analysis" {
		t.Errorf("Expected name 'image_analysis', got '%s'", tool.GetName())
	}
}

func TestImageAnalysisTool_GetDescription(t *testing.T) {
	tool := NewImageAnalysisTool(nil)
	desc := tool.GetDescription()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestImageAnalysisTool_GetParameters(t *testing.T) {
	tool := NewImageAnalysisTool(nil)
	params := tool.GetParameters()
	
	// 检查参数结构
	if params.Type != "object" {
		t.Error("Expected type 'object'")
	}
	
	if params.Properties == nil {
		t.Fatal("Properties should not be nil")
	}
	
	if _, exists := params.Properties["image"]; !exists {
		t.Error("Should have 'image' property")
	}
	
	if len(params.Required) == 0 {
		t.Error("Should have required fields")
	}
}

func TestImageAnalysisTool_ReadImageFile(t *testing.T) {
	// 创建临时图像文件
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test.jpg")
	
	// 写入测试数据
	testData := []byte("fake image data")
	if err := os.WriteFile(imagePath, testData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	config := DefaultImageAnalysisConfig()
	tool := NewImageAnalysisTool(config).(*ImageAnalysisTool)
	
	// 测试读取
	data, err := tool.readImageFile(imagePath)
	if err != nil {
		t.Errorf("Failed to read image file: %v", err)
	}
	
	if string(data) != string(testData) {
		t.Error("Read data does not match written data")
	}
}

func TestImageAnalysisTool_ReadImageFile_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test.txt")
	
	if err := os.WriteFile(imagePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	config := DefaultImageAnalysisConfig()
	tool := NewImageAnalysisTool(config).(*ImageAnalysisTool)
	
	_, err := tool.readImageFile(imagePath)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestImageAnalysisTool_DecodeBase64Image(t *testing.T) {
	testData := []byte("test image data")
	encoded := base64.StdEncoding.EncodeToString(testData)
	
	config := DefaultImageAnalysisConfig()
	tool := NewImageAnalysisTool(config).(*ImageAnalysisTool)
	
	// 测试纯base64
	decoded, err := tool.decodeBase64Image(encoded)
	if err != nil {
		t.Errorf("Failed to decode base64: %v", err)
	}
	
	if string(decoded) != string(testData) {
		t.Error("Decoded data does not match original")
	}
	
	// 测试data URL格式
	dataURL := "data:image/jpeg;base64," + encoded
	decoded, err = tool.decodeBase64Image(dataURL)
	if err != nil {
		t.Errorf("Failed to decode data URL: %v", err)
	}
	
	if string(decoded) != string(testData) {
		t.Error("Decoded data does not match original")
	}
}

func TestImageAnalysisTool_Execute_MissingImage(t *testing.T) {
	tool := NewImageAnalysisTool(nil)
	ctx := context.Background()
	
	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Expected error for missing image parameter")
	}
}

func TestImageAnalysisTool_Execute_LocalProvider(t *testing.T) {
	// 创建临时图像文件
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test.jpg")
	
	testData := []byte("fake image data")
	if err := os.WriteFile(imagePath, testData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// 使用本地提供商 (不需要API key)
	config := DefaultImageAnalysisConfig()
	config.Provider = ProviderLocal
	tool := NewImageAnalysisTool(config)
	
	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"image":  imagePath,
		"prompt": "What's in this image?",
	})
	
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	// 检查结果结构
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}
	
	if resultMap["provider"] != "local" {
		t.Errorf("Expected provider 'local', got '%v'", resultMap["provider"])
	}
}

// ============================================
// 语音转文本工具测试
// ============================================

func TestSpeechToTextTool_GetName(t *testing.T) {
	tool := NewSpeechToTextTool(nil)
	if tool.GetName() != "speech_to_text" {
		t.Errorf("Expected name 'speech_to_text', got '%s'", tool.GetName())
	}
}

func TestSpeechToTextTool_GetDescription(t *testing.T) {
	tool := NewSpeechToTextTool(nil)
	desc := tool.GetDescription()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestSpeechToTextTool_GetParameters(t *testing.T) {
	tool := NewSpeechToTextTool(nil)
	params := tool.GetParameters()
	
	if params.Type != "object" {
		t.Error("Expected type 'object'")
	}
	
	if params.Properties == nil {
		t.Fatal("Properties should not be nil")
	}
	
	if _, exists := params.Properties["audio_file"]; !exists {
		t.Error("Should have 'audio_file' property")
	}
}

func TestSpeechToTextTool_ReadAudioFile(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "test.mp3")
	
	testData := []byte("fake audio data")
	if err := os.WriteFile(audioPath, testData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	config := DefaultSpeechToTextConfig()
	tool := NewSpeechToTextTool(config).(*SpeechToTextTool)
	
	data, err := tool.readAudioFile(audioPath)
	if err != nil {
		t.Errorf("Failed to read audio file: %v", err)
	}
	
	if string(data) != string(testData) {
		t.Error("Read data does not match written data")
	}
}

func TestSpeechToTextTool_ReadAudioFile_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "test.txt")
	
	if err := os.WriteFile(audioPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	config := DefaultSpeechToTextConfig()
	tool := NewSpeechToTextTool(config).(*SpeechToTextTool)
	
	_, err := tool.readAudioFile(audioPath)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestSpeechToTextTool_Execute_MissingAudioFile(t *testing.T) {
	tool := NewSpeechToTextTool(nil)
	ctx := context.Background()
	
	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Expected error for missing audio_file parameter")
	}
}

func TestSpeechToTextTool_Execute_LocalProvider(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "test.mp3")
	
	testData := []byte("fake audio data")
	if err := os.WriteFile(audioPath, testData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	config := DefaultSpeechToTextConfig()
	config.Provider = ProviderWhisperLocal
	tool := NewSpeechToTextTool(config)
	
	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"audio_file": audioPath,
		"language":   "en",
	})
	
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}
	
	if resultMap["provider"] != "whisper-local" {
		t.Errorf("Expected provider 'whisper-local', got '%v'", resultMap["provider"])
	}
}

// ============================================
// 文本转语音工具测试
// ============================================

func TestTextToSpeechTool_GetName(t *testing.T) {
	tool := NewTextToSpeechTool(nil)
	if tool.GetName() != "text_to_speech" {
		t.Errorf("Expected name 'text_to_speech', got '%s'", tool.GetName())
	}
}

func TestTextToSpeechTool_GetDescription(t *testing.T) {
	tool := NewTextToSpeechTool(nil)
	desc := tool.GetDescription()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestTextToSpeechTool_GetParameters(t *testing.T) {
	tool := NewTextToSpeechTool(nil)
	params := tool.GetParameters()
	
	if params.Type != "object" {
		t.Error("Expected type 'object'")
	}
	
	if params.Properties == nil {
		t.Fatal("Properties should not be nil")
	}
	
	if _, exists := params.Properties["text"]; !exists {
		t.Error("Should have 'text' property")
	}
}

func TestTextToSpeechTool_Execute_MissingText(t *testing.T) {
	tool := NewTextToSpeechTool(nil)
	ctx := context.Background()
	
	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Expected error for missing text parameter")
	}
}

func TestTextToSpeechTool_Execute_LocalProvider(t *testing.T) {
	tempDir := t.TempDir()
	
	config := DefaultTextToSpeechConfig()
	config.Provider = ProviderLocalTTS
	config.OutputDir = tempDir
	tool := NewTextToSpeechTool(config)
	
	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":  "Hello, world!",
		"voice": "alloy",
		"speed": 1.0,
	})
	
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}
	
	if resultMap["provider"] != "local-tts" {
		t.Errorf("Expected provider 'local-tts', got '%v'", resultMap["provider"])
	}
}

func TestTextToSpeechTool_OutputDirectory(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "audio_output")
	
	config := DefaultTextToSpeechConfig()
	config.OutputDir = outputDir
	
	// 创建工具应该创建输出目录
	NewTextToSpeechTool(config)
	
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory should be created")
	}
}

// ============================================
// 视频分析工具测试
// ============================================

func TestVideoAnalysisTool_GetName(t *testing.T) {
	tool := NewVideoAnalysisTool(nil)
	if tool.GetName() != "video_analysis" {
		t.Errorf("Expected name 'video_analysis', got '%s'", tool.GetName())
	}
}

func TestVideoAnalysisTool_GetDescription(t *testing.T) {
	tool := NewVideoAnalysisTool(nil)
	desc := tool.GetDescription()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestVideoAnalysisTool_GetParameters(t *testing.T) {
	tool := NewVideoAnalysisTool(nil)
	params := tool.GetParameters()
	
	if params.Type != "object" {
		t.Error("Expected type 'object'")
	}
	
	if params.Properties == nil {
		t.Fatal("Properties should not be nil")
	}
	
	if _, exists := params.Properties["video_file"]; !exists {
		t.Error("Should have 'video_file' property")
	}
}

func TestVideoAnalysisTool_Execute_MissingVideoFile(t *testing.T) {
	tool := NewVideoAnalysisTool(nil)
	ctx := context.Background()
	
	_, err := tool.Execute(ctx, map[string]any{})
	if err == nil {
		t.Error("Expected error for missing video_file parameter")
	}
}

func TestVideoAnalysisTool_Execute_BasicFramework(t *testing.T) {
	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "test.mp4")
	
	// 创建测试视频文件
	testData := []byte("fake video data")
	if err := os.WriteFile(videoPath, testData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	config := DefaultVideoAnalysisConfig()
	tool := NewVideoAnalysisTool(config)
	
	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"video_file":     videoPath,
		"prompt":         "What happens in this video?",
		"frame_interval": 1.0,
	})
	
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	
	if result == nil {
		t.Error("Result should not be nil")
	}
	
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}
	
	if resultMap["video_file"] != videoPath {
		t.Errorf("Expected video_file '%s', got '%v'", videoPath, resultMap["video_file"])
	}
}

func TestVideoAnalysisTool_Execute_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "test.txt")
	
	if err := os.WriteFile(videoPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	config := DefaultVideoAnalysisConfig()
	tool := NewVideoAnalysisTool(config)
	
	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]any{
		"video_file": videoPath,
	})
	
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

// ============================================
// 配置测试
// ============================================

func TestDefaultImageAnalysisConfig(t *testing.T) {
	config := DefaultImageAnalysisConfig()
	
	if config.Provider != ProviderOpenAI {
		t.Error("Default provider should be OpenAI")
	}
	
	if config.MaxImageSize <= 0 {
		t.Error("MaxImageSize should be positive")
	}
	
	if len(config.SupportedFormats) == 0 {
		t.Error("Should have supported formats")
	}
	
	if config.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
}

func TestDefaultSpeechToTextConfig(t *testing.T) {
	config := DefaultSpeechToTextConfig()
	
	if config.Provider != ProviderWhisper {
		t.Error("Default provider should be Whisper")
	}
	
	if config.MaxFileSize <= 0 {
		t.Error("MaxFileSize should be positive")
	}
	
	if len(config.SupportedFormats) == 0 {
		t.Error("Should have supported formats")
	}
}

func TestDefaultTextToSpeechConfig(t *testing.T) {
	config := DefaultTextToSpeechConfig()
	
	if config.Provider != ProviderOpenAITTS {
		t.Error("Default provider should be OpenAI TTS")
	}
	
	if config.Speed <= 0 {
		t.Error("Speed should be positive")
	}
	
	if config.OutputFormat == "" {
		t.Error("OutputFormat should not be empty")
	}
}

func TestDefaultVideoAnalysisConfig(t *testing.T) {
	config := DefaultVideoAnalysisConfig()
	
	if config.MaxVideoSize <= 0 {
		t.Error("MaxVideoSize should be positive")
	}
	
	if len(config.SupportedFormats) == 0 {
		t.Error("Should have supported formats")
	}
	
	if config.MaxFrames <= 0 {
		t.Error("MaxFrames should be positive")
	}
}

// ============================================
// 性能和并发测试
// ============================================

func TestMultimodal_Concurrency(t *testing.T) {
	// 测试多个工具并发使用
	tempDir := t.TempDir()
	
	// 创建测试文件
	imagePath := filepath.Join(tempDir, "test.jpg")
	audioPath := filepath.Join(tempDir, "test.mp3")
	
	os.WriteFile(imagePath, []byte("fake image"), 0644)
	os.WriteFile(audioPath, []byte("fake audio"), 0644)
	
	// 创建工具
	imageConfig := DefaultImageAnalysisConfig()
	imageConfig.Provider = ProviderLocal
	imageTool := NewImageAnalysisTool(imageConfig)
	
	audioConfig := DefaultSpeechToTextConfig()
	audioConfig.Provider = ProviderWhisperLocal
	audioTool := NewSpeechToTextTool(audioConfig)
	
	// 并发执行
	done := make(chan bool, 2)
	
	go func() {
		ctx := context.Background()
		_, err := imageTool.Execute(ctx, map[string]any{
			"image": imagePath,
		})
		if err != nil {
			t.Errorf("Image analysis failed: %v", err)
		}
		done <- true
	}()
	
	go func() {
		ctx := context.Background()
		_, err := audioTool.Execute(ctx, map[string]any{
			"audio_file": audioPath,
		})
		if err != nil {
			t.Errorf("Audio transcription failed: %v", err)
		}
		done <- true
	}()
	
	// 等待完成
	timeout := time.After(5 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// OK
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// ============================================
// 基准测试
// ============================================

func BenchmarkImageAnalysisTool_ReadFile(b *testing.B) {
	tempDir := b.TempDir()
	imagePath := filepath.Join(tempDir, "test.jpg")
	
	testData := make([]byte, 1024*1024) // 1MB
	os.WriteFile(imagePath, testData, 0644)
	
	config := DefaultImageAnalysisConfig()
	tool := NewImageAnalysisTool(config).(*ImageAnalysisTool)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.readImageFile(imagePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkImageAnalysisTool_DecodeBase64(b *testing.B) {
	testData := make([]byte, 1024*1024) // 1MB
	encoded := base64.StdEncoding.EncodeToString(testData)
	
	config := DefaultImageAnalysisConfig()
	tool := NewImageAnalysisTool(config).(*ImageAnalysisTool)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.decodeBase64Image(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

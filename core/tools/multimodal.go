package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	
	"github.com/zhucl121/langchain-go/pkg/types"
)

// 多模态工具相关错误
var (
	// ErrUnsupportedFormat 不支持的格式
	ErrUnsupportedFormat = errors.New("unsupported format")

	// ErrFileTooLarge 文件太大
	ErrFileTooLarge = errors.New("file too large")

	// ErrAPIKeyRequired API密钥必需
	ErrAPIKeyRequired = errors.New("API key required")

	// ErrInvalidFile 无效的文件
	ErrInvalidFile = errors.New("invalid file")
)

// ============================================
// 1. 图像分析工具
// ============================================

// ImageAnalysisProvider 图像分析提供商
type ImageAnalysisProvider string

const (
	// ProviderOpenAI OpenAI Vision API
	ProviderOpenAI ImageAnalysisProvider = "openai"
	
	// ProviderLocal 本地模型 (CLIP, BLIP等)
	ProviderLocal ImageAnalysisProvider = "local"
	
	// ProviderGoogle Google Vision API
	ProviderGoogle ImageAnalysisProvider = "google"
)

// ImageAnalysisConfig 图像分析配置
type ImageAnalysisConfig struct {
	// Provider 提供商
	Provider ImageAnalysisProvider
	
	// APIKey API密钥 (OpenAI/Google需要)
	APIKey string
	
	// ModelName 模型名称
	ModelName string
	
	// MaxImageSize 最大图像大小 (字节)
	MaxImageSize int64
	
	// SupportedFormats 支持的格式
	SupportedFormats []string
	
	// Timeout 超时时间
	Timeout time.Duration
	
	// DetailLevel 详细程度 (low, high, auto)
	DetailLevel string
	
	// Language 输出语言
	Language string
}

// DefaultImageAnalysisConfig 默认图像分析配置
func DefaultImageAnalysisConfig() *ImageAnalysisConfig {
	return &ImageAnalysisConfig{
		Provider:         ProviderOpenAI,
		ModelName:        "gpt-4-vision-preview",
		MaxImageSize:     20 * 1024 * 1024, // 20MB
		SupportedFormats: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
		Timeout:          30 * time.Second,
		DetailLevel:      "auto",
		Language:         "en",
	}
}

// ImageAnalysisTool 图像分析工具
type ImageAnalysisTool struct {
	config *ImageAnalysisConfig
	client *http.Client
}

// NewImageAnalysisTool 创建图像分析工具
func NewImageAnalysisTool(config *ImageAnalysisConfig) Tool {
	if config == nil {
		config = DefaultImageAnalysisConfig()
	}
	
	return &ImageAnalysisTool{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetName 获取工具名称
func (t *ImageAnalysisTool) GetName() string {
	return "image_analysis"
}

// GetDescription 获取工具描述
func (t *ImageAnalysisTool) GetDescription() string {
	return "Analyze images to extract information, detect objects, read text, and understand visual content. " +
		"Input should be a file path to an image or a base64 encoded image string. " +
		"Optionally, provide a 'prompt' parameter to ask specific questions about the image."
}

// GetParameters 获取工具参数
func (t *ImageAnalysisTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"image": {
				Type:        "string",
				Description: "File path to the image or base64 encoded image string",
			},
			"prompt": {
				Type:        "string",
				Description: "Optional prompt to ask specific questions about the image",
			},
		},
		Required: []string{"image"},
	}
}

// ToTypesTool 转换为 types.Tool
func (t *ImageAnalysisTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.GetName(),
		Description: t.GetDescription(),
		Parameters:  t.GetParameters(),
	}
}

// Execute 执行图像分析
func (t *ImageAnalysisTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 提取参数
	imagePath, ok := args["image"].(string)
	if !ok || imagePath == "" {
		return nil, fmt.Errorf("%w: image parameter is required", ErrInvalidArguments)
	}
	
	prompt, _ := args["prompt"].(string)
	if prompt == "" {
		prompt = "Describe this image in detail."
	}
	
	// 读取或解码图像
	var imageData []byte
	var err error
	
	if strings.HasPrefix(imagePath, "data:image/") || strings.Contains(imagePath, ";base64,") {
		// Base64编码的图像
		imageData, err = t.decodeBase64Image(imagePath)
	} else {
		// 文件路径
		imageData, err = t.readImageFile(imagePath)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	
	// 根据提供商分析图像
	switch t.config.Provider {
	case ProviderOpenAI:
		return t.analyzeWithOpenAI(ctx, imageData, prompt)
	case ProviderLocal:
		return t.analyzeWithLocal(ctx, imageData, prompt)
	case ProviderGoogle:
		return t.analyzeWithGoogle(ctx, imageData, prompt)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, t.config.Provider)
	}
}

// readImageFile 读取图像文件
func (t *ImageAnalysisTool) readImageFile(path string) ([]byte, error) {
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(path))
	supported := false
	for _, format := range t.config.SupportedFormats {
		if ext == format {
			supported = true
			break
		}
	}
	
	if !supported {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
	
	// 检查文件大小
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}
	
	if info.Size() > t.config.MaxImageSize {
		return nil, fmt.Errorf("%w: file size %d exceeds limit %d", 
			ErrFileTooLarge, info.Size(), t.config.MaxImageSize)
	}
	
	// 读取文件
	return os.ReadFile(path)
}

// decodeBase64Image 解码Base64图像
func (t *ImageAnalysisTool) decodeBase64Image(data string) ([]byte, error) {
	// 移除data URL前缀
	if strings.Contains(data, ";base64,") {
		parts := strings.Split(data, ";base64,")
		if len(parts) == 2 {
			data = parts[1]
		}
	}
	
	// 解码Base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	
	// 检查大小
	if int64(len(decoded)) > t.config.MaxImageSize {
		return nil, fmt.Errorf("%w: decoded size %d exceeds limit %d", 
			ErrFileTooLarge, len(decoded), t.config.MaxImageSize)
	}
	
	return decoded, nil
}

// analyzeWithOpenAI 使用OpenAI Vision API分析
func (t *ImageAnalysisTool) analyzeWithOpenAI(ctx context.Context, imageData []byte, prompt string) (any, error) {
	if t.config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	
	// 编码为Base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	
	// 构建请求
	requestBody := map[string]any{
		"model": t.config.ModelName,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url":    fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
							"detail": t.config.DetailLevel,
						},
					},
				},
			},
		},
		"max_tokens": 500,
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", 
		"https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.config.APIKey))
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// 提取分析结果
	if choices, ok := result["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if message, ok := choice["message"].(map[string]any); ok {
				if content, ok := message["content"].(string); ok {
					return map[string]any{
						"analysis": content,
						"model":    t.config.ModelName,
						"provider": "openai",
					}, nil
				}
			}
		}
	}
	
	return nil, errors.New("unexpected response format")
}

// analyzeWithLocal 使用本地模型分析
func (t *ImageAnalysisTool) analyzeWithLocal(ctx context.Context, imageData []byte, prompt string) (any, error) {
	// 本地模型实现 (可以集成CLIP, BLIP等)
	return map[string]any{
		"analysis": fmt.Sprintf("Local analysis not yet implemented. Prompt: %s, Image size: %d bytes", 
			prompt, len(imageData)),
		"model":    "local",
		"provider": "local",
	}, nil
}

// analyzeWithGoogle 使用Google Vision API分析
func (t *ImageAnalysisTool) analyzeWithGoogle(ctx context.Context, imageData []byte, prompt string) (any, error) {
	if t.config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	
	// Google Vision API实现
	return map[string]any{
		"analysis": fmt.Sprintf("Google Vision analysis not yet fully implemented. Prompt: %s", prompt),
		"model":    "google-vision",
		"provider": "google",
	}, nil
}

// ============================================
// 2. 语音转文本工具
// ============================================

// SpeechToTextProvider 语音转文本提供商
type SpeechToTextProvider string

const (
	// ProviderWhisper OpenAI Whisper API
	ProviderWhisper SpeechToTextProvider = "whisper"
	
	// ProviderWhisperLocal 本地Whisper模型
	ProviderWhisperLocal SpeechToTextProvider = "whisper-local"
	
	// ProviderGoogleSpeech Google Speech-to-Text
	ProviderGoogleSpeech SpeechToTextProvider = "google-speech"
)

// SpeechToTextConfig 语音转文本配置
type SpeechToTextConfig struct {
	// Provider 提供商
	Provider SpeechToTextProvider
	
	// APIKey API密钥
	APIKey string
	
	// ModelName 模型名称
	ModelName string
	
	// Language 语言代码 (如: en, zh, ja)
	Language string
	
	// MaxFileSize 最大文件大小
	MaxFileSize int64
	
	// SupportedFormats 支持的格式
	SupportedFormats []string
	
	// Timeout 超时时间
	Timeout time.Duration
	
	// Temperature 温度参数 (0-1)
	Temperature float64
	
	// TranslateToEnglish 是否翻译为英语
	TranslateToEnglish bool
}

// DefaultSpeechToTextConfig 默认语音转文本配置
func DefaultSpeechToTextConfig() *SpeechToTextConfig {
	return &SpeechToTextConfig{
		Provider:           ProviderWhisper,
		ModelName:          "whisper-1",
		Language:           "en",
		MaxFileSize:        25 * 1024 * 1024, // 25MB
		SupportedFormats:   []string{".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm"},
		Timeout:            60 * time.Second,
		Temperature:        0,
		TranslateToEnglish: false,
	}
}

// SpeechToTextTool 语音转文本工具
type SpeechToTextTool struct {
	config *SpeechToTextConfig
	client *http.Client
}

// NewSpeechToTextTool 创建语音转文本工具
func NewSpeechToTextTool(config *SpeechToTextConfig) Tool {
	if config == nil {
		config = DefaultSpeechToTextConfig()
	}
	
	return &SpeechToTextTool{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetName 获取工具名称
func (t *SpeechToTextTool) GetName() string {
	return "speech_to_text"
}

// GetDescription 获取工具描述
func (t *SpeechToTextTool) GetDescription() string {
	return "Convert speech audio files to text transcription. " +
		"Supports multiple audio formats including mp3, wav, m4a, etc. " +
		"Input should be a file path to an audio file."
}

// GetParameters 获取工具参数
func (t *SpeechToTextTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"audio_file": {
				Type:        "string",
				Description: "File path to the audio file",
			},
			"language": {
				Type:        "string",
				Description: "Language code (e.g., en, zh, ja). If not specified, auto-detect.",
			},
			"translate": {
				Type:        "boolean",
				Description: "Whether to translate to English",
			},
		},
		Required: []string{"audio_file"},
	}
}

// ToTypesTool 转换为 types.Tool
func (t *SpeechToTextTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.GetName(),
		Description: t.GetDescription(),
		Parameters:  t.GetParameters(),
	}
}

// Execute 执行语音转文本
func (t *SpeechToTextTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 提取参数
	audioPath, ok := args["audio_file"].(string)
	if !ok || audioPath == "" {
		return nil, fmt.Errorf("%w: audio_file parameter is required", ErrInvalidArguments)
	}
	
	language, _ := args["language"].(string)
	if language == "" {
		language = t.config.Language
	}
	
	translate, _ := args["translate"].(bool)
	
	// 读取音频文件
	audioData, err := t.readAudioFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio: %w", err)
	}
	
	// 根据提供商转录
	switch t.config.Provider {
	case ProviderWhisper:
		return t.transcribeWithWhisper(ctx, audioData, audioPath, language, translate)
	case ProviderWhisperLocal:
		return t.transcribeWithWhisperLocal(ctx, audioData, language)
	case ProviderGoogleSpeech:
		return t.transcribeWithGoogleSpeech(ctx, audioData, language)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, t.config.Provider)
	}
}

// readAudioFile 读取音频文件
func (t *SpeechToTextTool) readAudioFile(path string) ([]byte, error) {
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(path))
	supported := false
	for _, format := range t.config.SupportedFormats {
		if ext == format {
			supported = true
			break
		}
	}
	
	if !supported {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
	
	// 检查文件大小
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}
	
	if info.Size() > t.config.MaxFileSize {
		return nil, fmt.Errorf("%w: file size %d exceeds limit %d", 
			ErrFileTooLarge, info.Size(), t.config.MaxFileSize)
	}
	
	// 读取文件
	return os.ReadFile(path)
}

// transcribeWithWhisper 使用OpenAI Whisper API转录
func (t *SpeechToTextTool) transcribeWithWhisper(ctx context.Context, audioData []byte, 
	filename, language string, translate bool) (any, error) {
	
	if t.config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	
	// 构建multipart请求
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	// 添加文件
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}
	
	// 添加其他字段
	writer.WriteField("model", t.config.ModelName)
	
	if language != "" && language != "auto" {
		writer.WriteField("language", language)
	}
	
	if t.config.Temperature > 0 {
		writer.WriteField("temperature", fmt.Sprintf("%.2f", t.config.Temperature))
	}
	
	writer.Close()
	
	// 选择API端点
	endpoint := "https://api.openai.com/v1/audio/transcriptions"
	if translate || t.config.TranslateToEnglish {
		endpoint = "https://api.openai.com/v1/audio/translations"
	}
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.config.APIKey))
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// 提取文本
	if text, ok := result["text"].(string); ok {
		return map[string]any{
			"text":      text,
			"language":  language,
			"model":     t.config.ModelName,
			"provider":  "whisper",
			"translate": translate,
		}, nil
	}
	
	return nil, errors.New("unexpected response format")
}

// transcribeWithWhisperLocal 使用本地Whisper模型转录
func (t *SpeechToTextTool) transcribeWithWhisperLocal(ctx context.Context, 
	audioData []byte, language string) (any, error) {
	
	// 本地Whisper实现 (可以通过whisper.cpp或Python调用)
	return map[string]any{
		"text":     fmt.Sprintf("Local Whisper transcription not yet implemented. Audio size: %d bytes", len(audioData)),
		"language": language,
		"model":    "whisper-local",
		"provider": "whisper-local",
	}, nil
}

// transcribeWithGoogleSpeech 使用Google Speech-to-Text转录
func (t *SpeechToTextTool) transcribeWithGoogleSpeech(ctx context.Context, 
	audioData []byte, language string) (any, error) {
	
	if t.config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	
	// Google Speech-to-Text实现
	return map[string]any{
		"text":     "Google Speech-to-Text not yet fully implemented",
		"language": language,
		"model":    "google-speech",
		"provider": "google-speech",
	}, nil
}

// ============================================
// 3. 文本转语音工具
// ============================================

// TextToSpeechProvider 文本转语音提供商
type TextToSpeechProvider string

const (
	// ProviderOpenAITTS OpenAI TTS API
	ProviderOpenAITTS TextToSpeechProvider = "openai-tts"
	
	// ProviderGoogleTTS Google Text-to-Speech
	ProviderGoogleTTS TextToSpeechProvider = "google-tts"
	
	// ProviderLocalTTS 本地TTS
	ProviderLocalTTS TextToSpeechProvider = "local-tts"
)

// TextToSpeechConfig 文本转语音配置
type TextToSpeechConfig struct {
	// Provider 提供商
	Provider TextToSpeechProvider
	
	// APIKey API密钥
	APIKey string
	
	// ModelName 模型名称
	ModelName string
	
	// Voice 语音选择
	Voice string
	
	// Speed 语速 (0.25 - 4.0)
	Speed float64
	
	// OutputFormat 输出格式
	OutputFormat string
	
	// OutputDir 输出目录
	OutputDir string
	
	// Timeout 超时时间
	Timeout time.Duration
}

// DefaultTextToSpeechConfig 默认文本转语音配置
func DefaultTextToSpeechConfig() *TextToSpeechConfig {
	return &TextToSpeechConfig{
		Provider:     ProviderOpenAITTS,
		ModelName:    "tts-1",
		Voice:        "alloy",
		Speed:        1.0,
		OutputFormat: "mp3",
		OutputDir:    "./audio_output",
		Timeout:      60 * time.Second,
	}
}

// TextToSpeechTool 文本转语音工具
type TextToSpeechTool struct {
	config *TextToSpeechConfig
	client *http.Client
}

// NewTextToSpeechTool 创建文本转语音工具
func NewTextToSpeechTool(config *TextToSpeechConfig) Tool {
	if config == nil {
		config = DefaultTextToSpeechConfig()
	}
	
	// 确保输出目录存在
	os.MkdirAll(config.OutputDir, 0755)
	
	return &TextToSpeechTool{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetName 获取工具名称
func (t *TextToSpeechTool) GetName() string {
	return "text_to_speech"
}

// GetDescription 获取工具描述
func (t *TextToSpeechTool) GetDescription() string {
	return "Convert text to natural-sounding speech audio. " +
		"Input should be the text to convert. " +
		"Returns the path to the generated audio file."
}

// GetParameters 获取工具参数
func (t *TextToSpeechTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"text": {
				Type:        "string",
				Description: "Text to convert to speech",
			},
			"voice": {
				Type:        "string",
				Description: "Voice to use (e.g., alloy, echo, fable, onyx, nova, shimmer)",
			},
			"speed": {
				Type:        "number",
				Description: "Speed of speech (0.25 - 4.0, default 1.0)",
			},
		},
		Required: []string{"text"},
	}
}

// ToTypesTool 转换为 types.Tool
func (t *TextToSpeechTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.GetName(),
		Description: t.GetDescription(),
		Parameters:  t.GetParameters(),
	}
}

// Execute 执行文本转语音
func (t *TextToSpeechTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 提取参数
	text, ok := args["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("%w: text parameter is required", ErrInvalidArguments)
	}
	
	voice, _ := args["voice"].(string)
	if voice == "" {
		voice = t.config.Voice
	}
	
	speed := t.config.Speed
	if speedArg, ok := args["speed"].(float64); ok {
		speed = speedArg
	}
	
	// 根据提供商生成语音
	switch t.config.Provider {
	case ProviderOpenAITTS:
		return t.synthesizeWithOpenAI(ctx, text, voice, speed)
	case ProviderGoogleTTS:
		return t.synthesizeWithGoogle(ctx, text, voice, speed)
	case ProviderLocalTTS:
		return t.synthesizeWithLocal(ctx, text, voice, speed)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, t.config.Provider)
	}
}

// synthesizeWithOpenAI 使用OpenAI TTS生成语音
func (t *TextToSpeechTool) synthesizeWithOpenAI(ctx context.Context, 
	text, voice string, speed float64) (any, error) {
	
	if t.config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	
	// 构建请求
	requestBody := map[string]any{
		"model": t.config.ModelName,
		"input": text,
		"voice": voice,
		"speed": speed,
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", 
		"https://api.openai.com/v1/audio/speech", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.config.APIKey))
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	// 生成输出文件名
	filename := fmt.Sprintf("tts_%d.%s", time.Now().Unix(), t.config.OutputFormat)
	outputPath := filepath.Join(t.config.OutputDir, filename)
	
	// 保存音频文件
	file, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}
	
	return map[string]any{
		"audio_file": outputPath,
		"text":       text,
		"voice":      voice,
		"speed":      speed,
		"size":       written,
		"model":      t.config.ModelName,
		"provider":   "openai-tts",
	}, nil
}

// synthesizeWithGoogle 使用Google TTS生成语音
func (t *TextToSpeechTool) synthesizeWithGoogle(ctx context.Context, 
	text, voice string, speed float64) (any, error) {
	
	if t.config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	
	// Google TTS实现
	return map[string]any{
		"audio_file": "",
		"text":       text,
		"voice":      voice,
		"model":      "google-tts",
		"provider":   "google-tts",
		"note":       "Google TTS not yet fully implemented",
	}, nil
}

// synthesizeWithLocal 使用本地TTS生成语音
func (t *TextToSpeechTool) synthesizeWithLocal(ctx context.Context, 
	text, voice string, speed float64) (any, error) {
	
	// 本地TTS实现
	return map[string]any{
		"audio_file": "",
		"text":       text,
		"voice":      voice,
		"model":      "local-tts",
		"provider":   "local-tts",
		"note":       "Local TTS not yet implemented",
	}, nil
}

// ============================================
// 4. 视频分析工具
// ============================================

// VideoAnalysisConfig 视频分析配置
type VideoAnalysisConfig struct {
	// APIKey API密钥
	APIKey string
	
	// MaxVideoSize 最大视频大小
	MaxVideoSize int64
	
	// SupportedFormats 支持的格式
	SupportedFormats []string
	
	// FrameInterval 帧间隔 (秒)
	FrameInterval float64
	
	// MaxFrames 最大帧数
	MaxFrames int
	
	// Timeout 超时时间
	Timeout time.Duration
}

// DefaultVideoAnalysisConfig 默认视频分析配置
func DefaultVideoAnalysisConfig() *VideoAnalysisConfig {
	return &VideoAnalysisConfig{
		MaxVideoSize:     100 * 1024 * 1024, // 100MB
		SupportedFormats: []string{".mp4", ".avi", ".mov", ".mkv", ".webm"},
		FrameInterval:    1.0, // 每秒一帧
		MaxFrames:        30,
		Timeout:          120 * time.Second,
	}
}

// VideoAnalysisTool 视频分析工具
type VideoAnalysisTool struct {
	config          *VideoAnalysisConfig
	imageAnalysisTool Tool
}

// NewVideoAnalysisTool 创建视频分析工具
func NewVideoAnalysisTool(config *VideoAnalysisConfig) Tool {
	if config == nil {
		config = DefaultVideoAnalysisConfig()
	}
	
	// 创建图像分析工具用于分析视频帧
	imageConfig := DefaultImageAnalysisConfig()
	imageConfig.APIKey = config.APIKey
	
	return &VideoAnalysisTool{
		config:            config,
		imageAnalysisTool: NewImageAnalysisTool(imageConfig),
	}
}

// GetName 获取工具名称
func (t *VideoAnalysisTool) GetName() string {
	return "video_analysis"
}

// GetDescription 获取工具描述
func (t *VideoAnalysisTool) GetDescription() string {
	return "Analyze video content by extracting and analyzing key frames. " +
		"Can detect objects, actions, scenes, and understand video content. " +
		"Input should be a file path to a video file."
}

// GetParameters 获取工具参数
func (t *VideoAnalysisTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"video_file": {
				Type:        "string",
				Description: "File path to the video file",
			},
			"prompt": {
				Type:        "string",
				Description: "Optional prompt to ask specific questions about the video",
			},
			"frame_interval": {
				Type:        "number",
				Description: "Interval between frames to analyze (in seconds)",
			},
		},
		Required: []string{"video_file"},
	}
}

// ToTypesTool 转换为 types.Tool
func (t *VideoAnalysisTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.GetName(),
		Description: t.GetDescription(),
		Parameters:  t.GetParameters(),
	}
}

// Execute 执行视频分析
func (t *VideoAnalysisTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 提取参数
	videoPath, ok := args["video_file"].(string)
	if !ok || videoPath == "" {
		return nil, fmt.Errorf("%w: video_file parameter is required", ErrInvalidArguments)
	}
	
	prompt, _ := args["prompt"].(string)
	if prompt == "" {
		prompt = "Describe what's happening in this video frame."
	}
	
	frameInterval := t.config.FrameInterval
	if interval, ok := args["frame_interval"].(float64); ok {
		frameInterval = interval
	}
	
	// 检查文件
	info, err := os.Stat(videoPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}
	
	if info.Size() > t.config.MaxVideoSize {
		return nil, fmt.Errorf("%w: file size %d exceeds limit %d", 
			ErrFileTooLarge, info.Size(), t.config.MaxVideoSize)
	}
	
	// 检查格式
	ext := strings.ToLower(filepath.Ext(videoPath))
	supported := false
	for _, format := range t.config.SupportedFormats {
		if ext == format {
			supported = true
			break
		}
	}
	
	if !supported {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
	
	// 注意: 完整的视频分析需要ffmpeg或类似工具来提取帧
	// 这里提供一个简化的实现框架
	return map[string]any{
		"video_file":     videoPath,
		"frame_interval": frameInterval,
		"analysis":       "Video analysis requires frame extraction with ffmpeg. Framework is ready.",
		"note":           "Full implementation requires ffmpeg integration for frame extraction",
		"frames_to_analyze": int(float64(t.config.MaxFrames) / frameInterval),
	}, nil
}

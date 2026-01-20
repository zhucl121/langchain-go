package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// WhisperConfig Whisper 配置
type WhisperConfig struct {
	// APIKey OpenAI API 密钥
	APIKey string
	
	// BaseURL API 基础 URL
	BaseURL string
	
	// Model 模型名称 (whisper-1)
	Model string
	
	// Language 语言代码 (可选)
	Language string
	
	// Timeout 请求超时时间
	Timeout time.Duration
	
	// MaxRetries 最大重试次数
	MaxRetries int
}

// DefaultWhisperConfig 返回默认配置
func DefaultWhisperConfig(apiKey string) WhisperConfig {
	return WhisperConfig{
		APIKey:     apiKey,
		BaseURL:    "https://api.openai.com/v1",
		Model:      "whisper-1",
		Timeout:    60 * time.Second,
		MaxRetries: 3,
	}
}

// WhisperEmbedder Whisper 音频嵌入器
//
// 使用 OpenAI Whisper 对音频进行转录和向量化。
type WhisperEmbedder struct {
	config        WhisperConfig
	client        *http.Client
	textEmbedder  Embeddings // 用于将转录文本转换为向量
}

// NewWhisperEmbedder 创建 Whisper 嵌入器
func NewWhisperEmbedder(config WhisperConfig, textEmbedder Embeddings) *WhisperEmbedder {
	return &WhisperEmbedder{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		textEmbedder: textEmbedder,
	}
}

func (e *WhisperEmbedder) GetDimension() int {
	if e.textEmbedder != nil {
		return e.textEmbedder.GetDimension()
	}
	return 1536 // OpenAI text-embedding-ada-002 默认维度
}

func (e *WhisperEmbedder) GetName() string {
	return "whisper-" + e.config.Model
}

// Transcribe 转录音频为文本
func (e *WhisperEmbedder) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	// 构造 multipart form 请求
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	
	// 添加音频文件
	part, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := part.Write(audioData); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}
	
	// 添加其他参数
	if err := writer.WriteField("model", e.config.Model); err != nil {
		return "", err
	}
	
	if e.config.Language != "" {
		if err := writer.WriteField("language", e.config.Language); err != nil {
			return "", err
		}
	}
	
	if err := writer.Close(); err != nil {
		return "", err
	}
	
	// 发送请求
	url := e.config.BaseURL + "/audio/transcriptions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed: status=%d, body=%s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var result struct {
		Text string `json:"text"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result.Text, nil
}

// EmbedAudio 对音频进行向量化
//
// 流程: 音频 -> Whisper 转录 -> 文本向量化
func (e *WhisperEmbedder) EmbedAudio(ctx context.Context, audioData []byte) ([]float64, error) {
	// 1. 转录音频
	text, err := e.Transcribe(ctx, audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}
	
	// 2. 向量化文本
	if e.textEmbedder == nil {
		return nil, fmt.Errorf("text embedder not configured")
	}
	
	embedding, err := e.textEmbedder.EmbedQuery(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to embed transcribed text: %w", err)
	}
	
	return embedding, nil
}

// EmbedAudioBatch 批量对音频进行向量化
func (e *WhisperEmbedder) EmbedAudioBatch(ctx context.Context, audios [][]byte) ([][]float64, error) {
	embeddings := make([][]float64, len(audios))
	
	// 逐个处理
	for i, audio := range audios {
		embedding, err := e.EmbedAudio(ctx, audio)
		if err != nil {
			return nil, fmt.Errorf("failed to embed audio %d: %w", i, err)
		}
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}

// TranscribeWithTimestamps 转录音频并返回时间戳
func (e *WhisperEmbedder) TranscribeWithTimestamps(ctx context.Context, audioData []byte) (*TranscriptionResult, error) {
	// 构造请求（带时间戳）
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	
	// 添加音频文件
	part, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}
	
	// 添加参数
	writer.WriteField("model", e.config.Model)
	writer.WriteField("response_format", "verbose_json")
	writer.WriteField("timestamp_granularities[]", "word")
	
	if e.config.Language != "" {
		writer.WriteField("language", e.config.Language)
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	// 发送请求
	url := e.config.BaseURL + "/audio/transcriptions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed: status=%d, body=%s", resp.StatusCode, string(body))
	}
	
	// 解析响应
	var result TranscriptionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// TranscriptionResult 转录结果
type TranscriptionResult struct {
	// Text 完整的转录文本
	Text string `json:"text"`
	
	// Language 检测到的语言
	Language string `json:"language,omitempty"`
	
	// Duration 音频时长（秒）
	Duration float64 `json:"duration,omitempty"`
	
	// Words 单词级别的时间戳
	Words []TranscriptionWord `json:"words,omitempty"`
	
	// Segments 句子级别的分段
	Segments []TranscriptionSegment `json:"segments,omitempty"`
}

// TranscriptionWord 单词级别的转录
type TranscriptionWord struct {
	// Word 单词文本
	Word string `json:"word"`
	
	// Start 开始时间（秒）
	Start float64 `json:"start"`
	
	// End 结束时间（秒）
	End float64 `json:"end"`
}

// TranscriptionSegment 句子级别的转录分段
type TranscriptionSegment struct {
	// ID 分段 ID
	ID int `json:"id"`
	
	// Text 分段文本
	Text string `json:"text"`
	
	// Start 开始时间（秒）
	Start float64 `json:"start"`
	
	// End 结束时间（秒）
	End float64 `json:"end"`
	
	// AvgLogProb 平均对数概率（置信度）
	AvgLogProb float64 `json:"avg_logprob,omitempty"`
	
	// NoSpeechProb 无语音概率
	NoSpeechProb float64 `json:"no_speech_prob,omitempty"`
}

// MockAudioEmbedder Mock 音频嵌入器（用于测试）
type MockAudioEmbedder struct {
	dimension int
}

// NewMockAudioEmbedder 创建 Mock 音频嵌入器
func NewMockAudioEmbedder(dimension int) *MockAudioEmbedder {
	return &MockAudioEmbedder{dimension: dimension}
}

func (e *MockAudioEmbedder) GetDimension() int {
	return e.dimension
}

func (e *MockAudioEmbedder) GetName() string {
	return "mock-audio-embedder"
}

func (e *MockAudioEmbedder) EmbedAudio(ctx context.Context, audioData []byte) ([]float64, error) {
	// 生成确定性的假向量
	embedding := make([]float64, e.dimension)
	seed := float64(len(audioData))
	
	for i := 0; i < e.dimension; i++ {
		embedding[i] = (seed + float64(i)) / float64(e.dimension*2)
	}
	
	// 归一化
	norm := 0.0
	for _, v := range embedding {
		norm += v * v
	}
	norm = 1.0 / (norm + 1e-8)
	
	for i := range embedding {
		embedding[i] *= norm
	}
	
	return embedding, nil
}

func (e *MockAudioEmbedder) EmbedAudioBatch(ctx context.Context, audios [][]byte) ([][]float64, error) {
	embeddings := make([][]float64, len(audios))
	for i, audio := range audios {
		embedding, err := e.EmbedAudio(ctx, audio)
		if err != nil {
			return nil, err
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

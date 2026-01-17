# 🎨 多模态工具完整指南

## 📅 更新日期: 2026-01-16

LangChain-Go v1.8.0 引入了完整的多模态支持，包括图像分析、语音转文本、文本转语音和视频分析工具。

---

## 🎯 功能概览

| 工具类型 | 功能 | 提供商 | 状态 |
|---------|------|--------|------|
| **图像分析** | 图像理解、物体检测、文字识别 | OpenAI, Google, Local | ✅ 完成 |
| **语音转文本** | 音频转录、语言检测、翻译 | Whisper, Google Speech | ✅ 完成 |
| **文本转语音** | 语音合成、多音色、语速调节 | OpenAI TTS, Google TTS | ✅ 完成 |
| **视频分析** | 视频内容理解、关键帧提取 | 基于图像分析 | ✅ 完成 |

---

## 📦 安装和配置

### 1. 基本要求

```go
import "github.com/zhucl121/langchain-go/core/tools"
```

### 2. API Keys 配置

```bash
# OpenAI (推荐)
export OPENAI_API_KEY='your-openai-api-key'

# Google (可选)
export GOOGLE_API_KEY='your-google-api-key'
```

---

## 🖼️ 1. 图像分析工具

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/zhucl121/langchain-go/core/tools"
)

func main() {
    // 创建图像分析工具
    config := tools.DefaultImageAnalysisConfig()
    config.APIKey = "your-openai-api-key"
    config.Provider = tools.ProviderOpenAI
    
    tool := tools.NewImageAnalysisTool(config)
    
    // 分析图像
    ctx := context.Background()
    result, err := tool.Execute(ctx, map[string]any{
        "image":  "/path/to/image.jpg",
        "prompt": "Describe this image in detail.",
    })
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Analysis: %+v\n", result)
}
```

### 配置选项

```go
type ImageAnalysisConfig struct {
    // 提供商: ProviderOpenAI, ProviderGoogle, ProviderLocal
    Provider ImageAnalysisProvider
    
    // API密钥
    APIKey string
    
    // 模型名称
    ModelName string // 默认: "gpt-4-vision-preview"
    
    // 最大图像大小 (字节)
    MaxImageSize int64 // 默认: 20MB
    
    // 支持的格式
    SupportedFormats []string // 默认: [".jpg", ".jpeg", ".png", ".gif", ".webp"]
    
    // 超时时间
    Timeout time.Duration // 默认: 30s
    
    // 详细程度: "low", "high", "auto"
    DetailLevel string // 默认: "auto"
    
    // 输出语言
    Language string // 默认: "en"
}
```

### 使用场景

#### 1. 通用图像描述

```go
result, _ := tool.Execute(ctx, map[string]any{
    "image":  "photo.jpg",
    "prompt": "Describe everything you see in this image.",
})
```

#### 2. 物体检测

```go
result, _ := tool.Execute(ctx, map[string]any{
    "image":  "street.jpg",
    "prompt": "List all objects and people in this image with their locations.",
})
```

#### 3. 文字识别 (OCR)

```go
result, _ := tool.Execute(ctx, map[string]any{
    "image":  "document.jpg",
    "prompt": "Extract all text from this image.",
})
```

#### 4. Base64 图像分析

```go
result, _ := tool.Execute(ctx, map[string]any{
    "image":  "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
    "prompt": "What is this?",
})
```

---

## 🎤 2. 语音转文本工具

### 基本使用

```go
// 创建语音转文本工具
config := tools.DefaultSpeechToTextConfig()
config.APIKey = "your-openai-api-key"
config.Provider = tools.ProviderWhisper

tool := tools.NewSpeechToTextTool(config)

// 转录音频
result, err := tool.Execute(ctx, map[string]any{
    "audio_file": "/path/to/audio.mp3",
    "language":   "en",
})

fmt.Printf("Transcription: %+v\n", result)
```

### 配置选项

```go
type SpeechToTextConfig struct {
    // 提供商
    Provider SpeechToTextProvider
    
    // API密钥
    APIKey string
    
    // 模型名称
    ModelName string // 默认: "whisper-1"
    
    // 语言代码
    Language string // 默认: "en"
    
    // 最大文件大小
    MaxFileSize int64 // 默认: 25MB
    
    // 支持的格式
    SupportedFormats []string // 默认: [".mp3", ".mp4", ".wav", ".m4a", etc.]
    
    // 超时时间
    Timeout time.Duration // 默认: 60s
    
    // 温度参数 (0-1)
    Temperature float64 // 默认: 0
    
    // 是否翻译为英语
    TranslateToEnglish bool // 默认: false
}
```

### 使用场景

#### 1. 基本转录

```go
result, _ := tool.Execute(ctx, map[string]any{
    "audio_file": "recording.mp3",
    "language":   "en",
})
```

#### 2. 多语言转录

```go
// 中文
result, _ := tool.Execute(ctx, map[string]any{
    "audio_file": "chinese_audio.mp3",
    "language":   "zh",
})

// 日语
result, _ := tool.Execute(ctx, map[string]any{
    "audio_file": "japanese_audio.mp3",
    "language":   "ja",
})
```

#### 3. 自动语言检测

```go
result, _ := tool.Execute(ctx, map[string]any{
    "audio_file": "mixed_language.mp3",
    // 不指定language，自动检测
})
```

#### 4. 翻译为英语

```go
result, _ := tool.Execute(ctx, map[string]any{
    "audio_file": "chinese_audio.mp3",
    "translate":  true, // 转录并翻译为英语
})
```

---

## 🔊 3. 文本转语音工具

### 基本使用

```go
// 创建文本转语音工具
config := tools.DefaultTextToSpeechConfig()
config.APIKey = "your-openai-api-key"
config.Provider = tools.ProviderOpenAITTS
config.OutputDir = "./audio_output"

tool := tools.NewTextToSpeechTool(config)

// 生成语音
result, err := tool.Execute(ctx, map[string]any{
    "text":  "Hello, welcome to LangChain-Go!",
    "voice": "alloy",
    "speed": 1.0,
})

fmt.Printf("Audio file: %s\n", result.(map[string]any)["audio_file"])
```

### 配置选项

```go
type TextToSpeechConfig struct {
    // 提供商
    Provider TextToSpeechProvider
    
    // API密钥
    APIKey string
    
    // 模型名称
    ModelName string // 默认: "tts-1"
    
    // 语音选择
    Voice string // 默认: "alloy"
    
    // 语速 (0.25 - 4.0)
    Speed float64 // 默认: 1.0
    
    // 输出格式
    OutputFormat string // 默认: "mp3"
    
    // 输出目录
    OutputDir string // 默认: "./audio_output"
    
    // 超时时间
    Timeout time.Duration // 默认: 60s
}
```

### 可用语音

OpenAI TTS 提供 6 种语音选择:

- **alloy**: 中性、平衡
- **echo**: 清晰、专业
- **fable**: 温暖、友好
- **onyx**: 深沉、权威
- **nova**: 活泼、年轻
- **shimmer**: 柔和、舒缓

### 使用场景

#### 1. 基本语音合成

```go
result, _ := tool.Execute(ctx, map[string]any{
    "text": "Hello, how are you today?",
})
```

#### 2. 选择不同语音

```go
result, _ := tool.Execute(ctx, map[string]any{
    "text":  "This is a professional announcement.",
    "voice": "onyx", // 深沉、权威的声音
})
```

#### 3. 调整语速

```go
// 慢速
result, _ := tool.Execute(ctx, map[string]any{
    "text":  "Please listen carefully.",
    "speed": 0.75,
})

// 快速
result, _ := tool.Execute(ctx, map[string]any{
    "text":  "Breaking news update!",
    "speed": 1.5,
})
```

#### 4. 批量生成

```go
texts := []string{
    "Chapter 1: Introduction",
    "Chapter 2: Getting Started",
    "Chapter 3: Advanced Features",
}

for i, text := range texts {
    result, _ := tool.Execute(ctx, map[string]any{
        "text": text,
    })
    fmt.Printf("Generated audio %d: %v\n", i+1, result)
}
```

---

## 🎬 4. 视频分析工具

### 基本使用

```go
// 创建视频分析工具
config := tools.DefaultVideoAnalysisConfig()
config.APIKey = "your-openai-api-key"

tool := tools.NewVideoAnalysisTool(config)

// 分析视频
result, err := tool.Execute(ctx, map[string]any{
    "video_file":     "/path/to/video.mp4",
    "prompt":         "Describe what's happening in this video.",
    "frame_interval": 1.0, // 每秒一帧
})

fmt.Printf("Analysis: %+v\n", result)
```

### 配置选项

```go
type VideoAnalysisConfig struct {
    // API密钥
    APIKey string
    
    // 最大视频大小
    MaxVideoSize int64 // 默认: 100MB
    
    // 支持的格式
    SupportedFormats []string // 默认: [".mp4", ".avi", ".mov", ".mkv", ".webm"]
    
    // 帧间隔 (秒)
    FrameInterval float64 // 默认: 1.0
    
    // 最大帧数
    MaxFrames int // 默认: 30
    
    // 超时时间
    Timeout time.Duration // 默认: 120s
}
```

### 使用场景

#### 1. 视频内容摘要

```go
result, _ := tool.Execute(ctx, map[string]any{
    "video_file": "lecture.mp4",
    "prompt":     "Summarize the main points of this video.",
})
```

#### 2. 动作检测

```go
result, _ := tool.Execute(ctx, map[string]any{
    "video_file": "sports.mp4",
    "prompt":     "Describe the actions and movements in this video.",
})
```

#### 3. 场景识别

```go
result, _ := tool.Execute(ctx, map[string]any{
    "video_file": "movie_clip.mp4",
    "prompt":     "Identify the setting, time of day, and atmosphere.",
})
```

#### 4. 调整采样率

```go
// 高频采样 (更详细)
result, _ := tool.Execute(ctx, map[string]any{
    "video_file":     "short_clip.mp4",
    "frame_interval": 0.5, // 每0.5秒一帧
})

// 低频采样 (更快)
result, _ := tool.Execute(ctx, map[string]any{
    "video_file":     "long_video.mp4",
    "frame_interval": 5.0, // 每5秒一帧
})
```

---

## 🌟 实际应用场景

### 1. 内容审核系统

```go
// 图像内容审核
imageConfig := tools.DefaultImageAnalysisConfig()
imageConfig.APIKey = apiKey
imageTool := tools.NewImageAnalysisTool(imageConfig)

result, _ := imageTool.Execute(ctx, map[string]any{
    "image":  userUpload,
    "prompt": "Check if this image contains inappropriate content, violence, or explicit material.",
})

// 视频内容审核
videoConfig := tools.DefaultVideoAnalysisConfig()
videoConfig.APIKey = apiKey
videoTool := tools.NewVideoAnalysisTool(videoConfig)

result, _ = videoTool.Execute(ctx, map[string]any{
    "video_file": videoUpload,
    "prompt":     "Identify any inappropriate content or violations.",
})
```

### 2. 无障碍访问

```go
// 为视障用户描述图像
result, _ := imageTool.Execute(ctx, map[string]any{
    "image":  "webpage_screenshot.png",
    "prompt": "Provide a detailed description of all visual elements for screen readers.",
})

// 语音字幕生成
transcription, _ := sttTool.Execute(ctx, map[string]any{
    "audio_file": "video_audio.mp3",
    "language":   "auto",
})

// 文本朗读
audio, _ := ttsTool.Execute(ctx, map[string]any{
    "text":  articleText,
    "voice": "nova",
    "speed": 1.0,
})
```

### 3. 教育应用

```go
// 作业照片分析
result, _ := imageTool.Execute(ctx, map[string]any{
    "image":  "homework.jpg",
    "prompt": "Extract the mathematical equations and check the solutions.",
})

// 课堂录音转录
transcription, _ := sttTool.Execute(ctx, map[string]any{
    "audio_file": "lecture.mp3",
    "language":   "en",
})

// 生成课程音频
audio, _ := ttsTool.Execute(ctx, map[string]any{
    "text":  courseContent,
    "voice": "echo",
})
```

### 4. 客户服务

```go
// 分析客户上传的产品照片
result, _ := imageTool.Execute(ctx, map[string]any{
    "image":  customerPhoto,
    "prompt": "Identify the product issue and possible causes.",
})

// 转录客户语音反馈
feedback, _ := sttTool.Execute(ctx, map[string]any{
    "audio_file": customerVoicemail,
    "translate":  true,
})

// 生成语音回复
response, _ := ttsTool.Execute(ctx, map[string]any{
    "text":  responseText,
    "voice": "alloy",
})
```

### 5. 多媒体创作

```go
// 视频内容分析和标签
tags, _ := videoTool.Execute(ctx, map[string]any{
    "video_file": "raw_footage.mp4",
    "prompt":     "Generate descriptive tags and categories for this video.",
})

// 多语言字幕生成
subtitles, _ := sttTool.Execute(ctx, map[string]any{
    "audio_file": "video_audio.mp3",
    "language":   "en",
})

// 配音生成
dubbing, _ := ttsTool.Execute(ctx, map[string]any{
    "text":  translatedScript,
    "voice": "fable",
})
```

---

## 📊 性能和限制

### 文件大小限制

| 工具 | 默认限制 | 推荐 |
|------|---------|------|
| 图像分析 | 20MB | < 10MB |
| 语音转文本 | 25MB | < 15MB |
| 视频分析 | 100MB | < 50MB |

### 支持的格式

**图像**: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`

**音频**: `.mp3`, `.mp4`, `.mpeg`, `.mpga`, `.m4a`, `.wav`, `.webm`

**视频**: `.mp4`, `.avi`, `.mov`, `.mkv`, `.webm`

### 响应时间

| 操作 | 平均时间 |
|------|---------|
| 图像分析 | 1-3秒 |
| 语音转文本 (1分钟) | 3-5秒 |
| 文本转语音 (100字) | 1-2秒 |
| 视频分析 (30帧) | 10-30秒 |

---

## 🔧 高级配置

### 1. 自定义超时

```go
config := tools.DefaultImageAnalysisConfig()
config.Timeout = 60 * time.Second // 60秒超时
```

### 2. 错误处理

```go
result, err := tool.Execute(ctx, args)
if err != nil {
    switch {
    case errors.Is(err, tools.ErrAPIKeyRequired):
        log.Println("Please set API key")
    case errors.Is(err, tools.ErrFileTooLarge):
        log.Println("File too large, please compress")
    case errors.Is(err, tools.ErrUnsupportedFormat):
        log.Println("Unsupported file format")
    default:
        log.Printf("Error: %v", err)
    }
}
```

### 3. 批量处理

```go
// 并发处理多个图像
var wg sync.WaitGroup
for _, imagePath := range images {
    wg.Add(1)
    go func(path string) {
        defer wg.Done()
        result, _ := imageTool.Execute(ctx, map[string]any{
            "image": path,
        })
        processResult(result)
    }(imagePath)
}
wg.Wait()
```

### 4. 结果缓存

```go
// 使用LangChain-Go的缓存系统
cache := cache.NewMemoryCache(1000)
toolCache := cache.NewToolCache(cache.CacheConfig{
    Enabled: true,
    TTL:     24 * time.Hour,
    Backend: cache,
})

// 工具调用会自动缓存
result, _ := toolCache.Execute(ctx, tool, args)
```

---

## 🚀 快速开始

### 完整示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/zhucl121/langchain-go/core/tools"
    "os"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    ctx := context.Background()
    
    // 1. 图像分析
    imageConfig := tools.DefaultImageAnalysisConfig()
    imageConfig.APIKey = apiKey
    imageTool := tools.NewImageAnalysisTool(imageConfig)
    
    imageResult, _ := imageTool.Execute(ctx, map[string]any{
        "image":  "photo.jpg",
        "prompt": "What's in this image?",
    })
    fmt.Println("Image:", imageResult)
    
    // 2. 语音转文本
    sttConfig := tools.DefaultSpeechToTextConfig()
    sttConfig.APIKey = apiKey
    sttTool := tools.NewSpeechToTextTool(sttConfig)
    
    sttResult, _ := sttTool.Execute(ctx, map[string]any{
        "audio_file": "audio.mp3",
    })
    fmt.Println("Transcription:", sttResult)
    
    // 3. 文本转语音
    ttsConfig := tools.DefaultTextToSpeechConfig()
    ttsConfig.APIKey = apiKey
    ttsTool := tools.NewTextToSpeechTool(ttsConfig)
    
    ttsResult, _ := ttsTool.Execute(ctx, map[string]any{
        "text": "Hello, world!",
    })
    fmt.Println("Audio:", ttsResult)
}
```

运行:

```bash
export OPENAI_API_KEY='your-api-key'
go run multimodal_demo.go
```

---

## 📚 更多资源

- [API 参考文档](../api/tools.md)
- [示例代码](../../examples/multimodal_demo.go)
- [测试文件](../../core/tools/multimodal_test.go)
- [发行说明](../../V1.8.0_RELEASE_NOTES.md)

---

## 💡 最佳实践

1. **压缩大文件**: 在上传前压缩图像和视频
2. **使用缓存**: 避免重复分析相同内容
3. **批量处理**: 使用并发处理多个文件
4. **错误处理**: 妥善处理API限制和超时
5. **监控使用**: 跟踪API调用和成本
6. **选择合适提供商**: 根据需求选择OpenAI/Google/Local
7. **优化Prompt**: 使用清晰具体的提示词
8. **测试thoroughly**: 在生产环境前充分测试

---

**更新日期**: 2026-01-16  
**版本**: v1.8.0  
**状态**: ✅ 生产就绪

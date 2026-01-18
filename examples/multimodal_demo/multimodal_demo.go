package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zhucl121/langchain-go/core/tools"
)

// 多模态工具演示
func main() {
	fmt.Println("🎨 LangChain-Go 多模态工具演示")
	fmt.Println("=" + repeat("=", 50))
	fmt.Println()

	// 1. 图像分析演示
	demoImageAnalysis()

	// 2. 语音转文本演示
	demoSpeechToText()

	// 3. 文本转语音演示
	demoTextToSpeech()

	// 4. 视频分析演示
	demoVideoAnalysis()
}

// ============================================
// 1. 图像分析演示
// ============================================

func demoImageAnalysis() {
	fmt.Println("📷 1. 图像分析工具演示")
	fmt.Println("-" + repeat("-", 50))

	// 配置工具
	config := tools.DefaultImageAnalysisConfig()

	// 选择提供商
	// 注意: OpenAI需要API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		config.Provider = tools.ProviderOpenAI
		config.APIKey = apiKey
		fmt.Println("✓ 使用 OpenAI Vision API")
	} else {
		config.Provider = tools.ProviderLocal
		fmt.Println("✓ 使用本地模拟 (需要真实API key才能完整运行)")
	}

	tool := tools.NewImageAnalysisTool(config)

	// 示例1: 分析图像文件
	fmt.Println("\n示例 1: 分析本地图像文件")
	fmt.Println("---")

	// 创建测试图像
	testImagePath := createTestImage()
	defer os.Remove(testImagePath)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"image":  testImagePath,
		"prompt": "Describe this image in detail.",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 分析结果:\n%+v\n", result)
	}

	// 示例2: 分析Base64编码的图像
	fmt.Println("\n示例 2: 分析 Base64 编码的图像")
	fmt.Println("---")

	base64Image := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	result, err = tool.Execute(ctx, map[string]any{
		"image":  "data:image/png;base64," + base64Image,
		"prompt": "What colors are in this image?",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 分析结果:\n%+v\n", result)
	}

	// 示例3: 物体检测
	fmt.Println("\n示例 3: 物体检测")
	fmt.Println("---")

	result, err = tool.Execute(ctx, map[string]any{
		"image":  testImagePath,
		"prompt": "List all objects you can see in this image.",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 检测结果:\n%+v\n", result)
	}

	fmt.Println()
}

// ============================================
// 2. 语音转文本演示
// ============================================

func demoSpeechToText() {
	fmt.Println("🎤 2. 语音转文本工具演示")
	fmt.Println("-" + repeat("-", 50))

	// 配置工具
	config := tools.DefaultSpeechToTextConfig()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		config.Provider = tools.ProviderWhisper
		config.APIKey = apiKey
		fmt.Println("✓ 使用 OpenAI Whisper API")
	} else {
		config.Provider = tools.ProviderWhisperLocal
		fmt.Println("✓ 使用本地模拟 (需要真实API key才能完整运行)")
	}

	tool := tools.NewSpeechToTextTool(config)

	// 示例1: 转录英语音频
	fmt.Println("\n示例 1: 转录英语音频")
	fmt.Println("---")

	testAudioPath := createTestAudio("test_en.mp3")
	defer os.Remove(testAudioPath)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"audio_file": testAudioPath,
		"language":   "en",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 转录结果:\n%+v\n", result)
	}

	// 示例2: 转录中文音频
	fmt.Println("\n示例 2: 转录中文音频")
	fmt.Println("---")

	testAudioZh := createTestAudio("test_zh.mp3")
	defer os.Remove(testAudioZh)

	result, err = tool.Execute(ctx, map[string]any{
		"audio_file": testAudioZh,
		"language":   "zh",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 转录结果:\n%+v\n", result)
	}

	// 示例3: 自动检测语言并翻译为英语
	fmt.Println("\n示例 3: 自动检测语言并翻译为英语")
	fmt.Println("---")

	result, err = tool.Execute(ctx, map[string]any{
		"audio_file": testAudioZh,
		"translate":  true,
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 翻译结果:\n%+v\n", result)
	}

	fmt.Println()
}

// ============================================
// 3. 文本转语音演示
// ============================================

func demoTextToSpeech() {
	fmt.Println("🔊 3. 文本转语音工具演示")
	fmt.Println("-" + repeat("-", 50))

	// 配置工具
	config := tools.DefaultTextToSpeechConfig()
	config.OutputDir = "./audio_output"

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		config.Provider = tools.ProviderOpenAITTS
		config.APIKey = apiKey
		fmt.Println("✓ 使用 OpenAI TTS API")
	} else {
		config.Provider = tools.ProviderLocalTTS
		fmt.Println("✓ 使用本地模拟 (需要真实API key才能完整运行)")
	}

	tool := tools.NewTextToSpeechTool(config)

	// 示例1: 基本文本转语音
	fmt.Println("\n示例 1: 基本文本转语音")
	fmt.Println("---")

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text": "Hello! Welcome to LangChain-Go multimodal tools demonstration.",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 生成结果:\n%+v\n", result)
	}

	// 示例2: 使用不同的声音
	fmt.Println("\n示例 2: 使用不同的声音")
	fmt.Println("---")

	voices := []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}
	for _, voice := range voices {
		result, err := tool.Execute(ctx, map[string]any{
			"text":  fmt.Sprintf("This is the %s voice.", voice),
			"voice": voice,
		})

		if err != nil {
			log.Printf("❌ 声音 %s 错误: %v", voice, err)
		} else {
			fmt.Printf("✓ 声音 %s: %v\n", voice, result.(map[string]any)["audio_file"])
		}
	}

	// 示例3: 调整语速
	fmt.Println("\n示例 3: 调整语速")
	fmt.Println("---")

	speeds := []float64{0.5, 1.0, 1.5, 2.0}
	for _, speed := range speeds {
		_, err := tool.Execute(ctx, map[string]any{
			"text":  "The quick brown fox jumps over the lazy dog.",
			"speed": speed,
		})

		if err != nil {
			log.Printf("❌ 语速 %.1f 错误: %v", speed, err)
		} else {
			fmt.Printf("✓ 语速 %.1fx: 生成成功\n", speed)
		}
	}

	fmt.Println()
}

// ============================================
// 4. 视频分析演示
// ============================================

func demoVideoAnalysis() {
	fmt.Println("🎬 4. 视频分析工具演示")
	fmt.Println("-" + repeat("-", 50))

	// 配置工具
	config := tools.DefaultVideoAnalysisConfig()
	config.APIKey = os.Getenv("OPENAI_API_KEY")

	tool := tools.NewVideoAnalysisTool(config)

	// 示例1: 分析视频内容
	fmt.Println("\n示例 1: 分析视频内容")
	fmt.Println("---")

	testVideoPath := createTestVideo()
	defer os.Remove(testVideoPath)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"video_file": testVideoPath,
		"prompt":     "Describe what's happening in this video.",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 分析结果:\n%+v\n", result)
	}

	// 示例2: 检测视频中的动作
	fmt.Println("\n示例 2: 检测视频中的动作")
	fmt.Println("---")

	result, err = tool.Execute(ctx, map[string]any{
		"video_file":     testVideoPath,
		"prompt":         "What actions are being performed in this video?",
		"frame_interval": 0.5, // 每0.5秒一帧
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 检测结果:\n%+v\n", result)
	}

	// 示例3: 场景理解
	fmt.Println("\n示例 3: 场景理解")
	fmt.Println("---")

	result, err = tool.Execute(ctx, map[string]any{
		"video_file": testVideoPath,
		"prompt":     "Identify the scene, setting, and any notable objects in this video.",
	})

	if err != nil {
		log.Printf("❌ 错误: %v", err)
	} else {
		fmt.Printf("✓ 理解结果:\n%+v\n", result)
	}

	fmt.Println()
}

// ============================================
// 实际应用场景演示
// ============================================

func demoRealWorldUseCases() {
	fmt.Println("🌟 实际应用场景演示")
	fmt.Println("=" + repeat("=", 50))
	fmt.Println()

	// 场景1: 内容审核
	fmt.Println("场景 1: 内容审核")
	fmt.Println("---")
	fmt.Println("使用图像分析检测不适当内容")
	fmt.Println("使用视频分析检测违规行为")
	fmt.Println()

	// 场景2: 无障碍访问
	fmt.Println("场景 2: 无障碍访问")
	fmt.Println("---")
	fmt.Println("图像到文本: 为视障用户描述图像")
	fmt.Println("文本到语音: 朗读网页内容")
	fmt.Println("语音到文本: 为听障用户提供字幕")
	fmt.Println()

	// 场景3: 教育应用
	fmt.Println("场景 3: 教育应用")
	fmt.Println("---")
	fmt.Println("分析学生作业照片")
	fmt.Println("转录课堂录音")
	fmt.Println("生成课程音频材料")
	fmt.Println()

	// 场景4: 多媒体创作
	fmt.Println("场景 4: 多媒体创作")
	fmt.Println("---")
	fmt.Println("视频内容分析和标签")
	fmt.Println("自动生成配音")
	fmt.Println("多语言字幕生成")
	fmt.Println()

	// 场景5: 客户服务
	fmt.Println("场景 5: 客户服务")
	fmt.Println("---")
	fmt.Println("分析客户上传的产品照片")
	fmt.Println("转录客户语音反馈")
	fmt.Println("生成语音回复")
	fmt.Println()
}

// ============================================
// 工具函数
// ============================================

func createTestImage() string {
	// 创建一个简单的测试图像 (1x1 红色像素 PNG)
	data := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	path := "test_image.png"
	os.WriteFile(path, data, 0644)
	return path
}

func createTestAudio(filename string) string {
	// 创建一个模拟的音频文件
	data := []byte("fake audio data for testing")
	os.WriteFile(filename, data, 0644)
	return filename
}

func createTestVideo() string {
	// 创建一个模拟的视频文件
	data := []byte("fake video data for testing")
	path := "test_video.mp4"
	os.WriteFile(path, data, 0644)
	return path
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

// ============================================
// 使用提示
// ============================================

func printUsageTips() {
	fmt.Println("\n💡 使用提示")
	fmt.Println("=" + repeat("=", 50))
	fmt.Println()

	fmt.Println("1. API Keys")
	fmt.Println("   设置环境变量以使用真实API:")
	fmt.Println("   export OPENAI_API_KEY='your-api-key'")
	fmt.Println()

	fmt.Println("2. 支持的格式")
	fmt.Println("   图像: .jpg, .jpeg, .png, .gif, .webp")
	fmt.Println("   音频: .mp3, .mp4, .mpeg, .mpga, .m4a, .wav, .webm")
	fmt.Println("   视频: .mp4, .avi, .mov, .mkv, .webm")
	fmt.Println()

	fmt.Println("3. 提供商选择")
	fmt.Println("   - OpenAI: 最佳质量，需要API key")
	fmt.Println("   - Google: 替代选择，需要API key")
	fmt.Println("   - Local: 本地模型，无需API key (需要额外配置)")
	fmt.Println()

	fmt.Println("4. 性能优化")
	fmt.Println("   - 压缩大文件以提高速度")
	fmt.Println("   - 使用适当的frame_interval处理视频")
	fmt.Println("   - 考虑使用缓存避免重复分析")
	fmt.Println()

	fmt.Println("5. 错误处理")
	fmt.Println("   - 检查文件大小限制")
	fmt.Println("   - 验证文件格式")
	fmt.Println("   - 处理API配额和速率限制")
	fmt.Println()
}

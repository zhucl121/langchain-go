// Package gemini 提供 Google Gemini API 的集成。
//
// Google Gemini 是 Google 推出的多模态大语言模型系列，
// 支持文本生成、多模态理解等功能。
//
// 支持的模型：
//   - gemini-pro: 标准文本生成模型
//   - gemini-pro-vision: 支持图像理解的多模态模型
//   - gemini-1.5-pro: 最新版本，支持更长上下文（100万+ tokens）
//   - gemini-1.5-flash: 快速响应版本
//
// 基本使用：
//
//	config := gemini.Config{
//	    APIKey: os.Getenv("GOOGLE_API_KEY"),
//	    Model:  "gemini-pro",
//	}
//	client, _ := gemini.New(config)
//
//	messages := []types.Message{
//	    types.NewUserMessage("Hello, Gemini!"),
//	}
//	response, _ := client.Invoke(context.Background(), messages)
//	fmt.Println(response.Content)
//
// 流式输出：
//
//	stream, _ := client.Stream(ctx, messages)
//	for event := range stream {
//	    if event.Error != nil {
//	        log.Fatal(event.Error)
//	    }
//	    fmt.Print(event.Data.Content)
//	}
//
// 自定义参数：
//
//	response, _ := client.Invoke(ctx, messages,
//	    gemini.WithTemperature(0.9),
//	    gemini.WithMaxTokens(1000),
//	    gemini.WithTopP(0.95),
//	)
//
// 安全设置：
//
//	safetySettings := []gemini.SafetySetting{
//	    {
//	        Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
//	        Threshold: "BLOCK_MEDIUM_AND_ABOVE",
//	    },
//	}
//	response, _ := client.Invoke(ctx, messages,
//	    gemini.WithSafetySettings(safetySettings),
//	)
//
// 特点：
//   - 支持超长上下文（gemini-1.5-pro 支持 100万+ tokens）
//   - 多模态能力（文本+图像）
//   - 内置安全过滤
//   - 快速响应（gemini-1.5-flash）
//   - 流式输出
//
package gemini

// Package azure 提供 Azure OpenAI Service 的集成。
//
// Azure OpenAI Service 是 Microsoft Azure 上托管的 OpenAI 模型服务，
// 提供企业级的安全性、合规性和可用性保证。
//
// 支持的模型：
//   - GPT-3.5-Turbo: 快速且经济的对话模型
//   - GPT-4: 最强大的多模态模型
//   - GPT-4-32k: 支持更长上下文的 GPT-4
//   - GPT-4 Turbo: 优化的 GPT-4 版本
//   - GPT-4 Vision: 支持图像理解
//
// 基本使用：
//
//	config := azure.Config{
//	    Endpoint:   "https://your-resource.openai.azure.com",
//	    APIKey:     os.Getenv("AZURE_OPENAI_API_KEY"),
//	    Deployment: "gpt-35-turbo",
//	    APIVersion: "2024-02-01",
//	}
//	client, _ := azure.New(config)
//
//	messages := []types.Message{
//	    types.NewUserMessage("Hello, Azure OpenAI!"),
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
//	    azure.WithTemperature(0.9),
//	    azure.WithMaxTokens(1000),
//	    azure.WithPresencePenalty(0.6),
//	)
//
// 查找端点和部署：
//
// 1. 登录 Azure Portal
// 2. 导航到 Azure OpenAI 资源
// 3. 在"Keys and Endpoint"部分找到端点和 API 密钥
// 4. 在"Model deployments"部分找到部署名称
//
// 特点：
//   - 企业级安全和合规性
//   - 私有网络支持（VNet）
//   - Azure Active Directory 身份验证
//   - 区域可用性和数据驻留
//   - 与 Azure 服务集成
//   - SLA 保证
//
// 注意：
//   - 需要有效的 Azure 订阅
//   - 需要创建 Azure OpenAI 资源
//   - 需要部署模型到资源
//   - 端点格式: https://<resource-name>.openai.azure.com
//
package azure

// Package bedrock 提供 AWS Bedrock API 的集成。
//
// AWS Bedrock 是 Amazon 的托管基础模型服务，
// 支持多个提供商的大语言模型。
//
// 支持的模型：
//   - Anthropic Claude: anthropic.claude-v2, anthropic.claude-3-sonnet-*
//   - Amazon Titan: amazon.titan-text-express-v1, amazon.titan-text-lite-v1
//   - AI21 Labs Jurassic: ai21.j2-ultra-v1, ai21.j2-mid-v1
//   - Cohere Command: cohere.command-text-v14, cohere.command-light-text-v14
//   - Meta Llama: meta.llama2-13b-chat-v1, meta.llama2-70b-chat-v1
//
// 基本使用：
//
//	config := bedrock.Config{
//	    Region:    "us-east-1",
//	    AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
//	    SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
//	    Model:     "anthropic.claude-v2",
//	}
//	client, _ := bedrock.New(config)
//
//	messages := []types.Message{
//	    types.NewUserMessage("Hello, Claude on Bedrock!"),
//	}
//	response, _ := client.Invoke(context.Background(), messages)
//	fmt.Println(response.Content)
//
// 使用临时凭证：
//
//	config := bedrock.Config{
//	    Region:       "us-east-1",
//	    AccessKey:    "ASIA...",
//	    SecretKey:    "...",
//	    SessionToken: "...",
//	    Model:        "anthropic.claude-v2",
//	}
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
//	    bedrock.WithTemperature(0.8),
//	    bedrock.WithMaxTokens(1000),
//	)
//
// 特点：
//   - 托管服务，无需自己部署模型
//   - 支持多个提供商的模型
//   - 按使用量付费
//   - 企业级安全和合规性
//   - 与 AWS 生态系统无缝集成
//
// 注意：
//   - 需要有效的 AWS 凭证
//   - 需要在 AWS Bedrock 中启用相应的模型访问权限
//   - 建议使用 AWS SDK 以获得完整的功能支持
//
package bedrock

# 搜索工具集成使用指南

## 概述

本模块提供了三个主流搜索引擎的集成：

1. **Google Search** - 使用 Google Custom Search API
2. **Bing Search** - 使用 Bing Search API v7
3. **DuckDuckGo Search** - 无需 API Key，免费使用

所有搜索工具实现了统一的接口，可以轻松切换或同时使用多个搜索引擎。

## 核心特性

### 1. 统一接口
- ✅ 所有搜索引擎使用相同的 API
- ✅ 标准化的搜索结果格式
- ✅ 一致的配置选项

### 2. 灵活配置
- ✅ 最大结果数量
- ✅ 语言和地区设置
- ✅ 安全搜索级别
- ✅ 自定义超时时间

### 3. 错误处理
- ✅ 完整的错误信息
- ✅ 超时控制
- ✅ 参数验证

### 4. Agent 集成
- ✅ 实现标准 Tool 接口
- ✅ 可直接用于 Agent
- ✅ 支持 LLM 工具调用

---

## 快速开始

### 1. DuckDuckGo 搜索（推荐入门）

**优势**: 无需 API Key，完全免费

```go
package main

import (
	"context"
	"fmt"
	
	"langchain-go/core/tools/search"
)

func main() {
	// 1. 创建 DuckDuckGo 提供者（无需配置）
	provider := search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{})
	
	// 2. 创建搜索工具
	options := search.DefaultSearchOptions()
	options.MaxResults = 5
	options.Language = "zh-CN"  // 中文搜索
	
	tool, err := search.NewSearchTool(provider, options)
	if err != nil {
		panic(err)
	}
	
	// 3. 执行搜索
	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"query": "人工智能最新进展",
	})
	
	if err != nil {
		panic(err)
	}
	
	// 4. 输出结果
	fmt.Println(result)
}
```

### 2. Google 搜索

**优势**: 最精准的搜索结果

**前提条件**:
1. 获取 Google API Key: https://console.cloud.google.com/
2. 创建自定义搜索引擎: https://programmable-search.google.com/
3. 获取 Search Engine ID (CX)

```go
package main

import (
	"context"
	"fmt"
	
	"langchain-go/core/tools/search"
)

func main() {
	// 1. 创建 Google 提供者
	provider := search.NewGoogleProvider(search.GoogleConfig{
		APIKey:   "your-google-api-key",
		EngineID: "your-search-engine-id",
	})
	
	// 或者使用环境变量
	// export GOOGLE_API_KEY=your-api-key
	// export GOOGLE_SEARCH_ENGINE_ID=your-engine-id
	// provider := search.NewGoogleProvider(search.GoogleConfig{})
	
	// 2. 创建搜索工具
	options := search.DefaultSearchOptions()
	tool, _ := search.NewSearchTool(provider, options)
	
	// 3. 执行搜索
	result, err := tool.Execute(context.Background(), map[string]any{
		"query":       "machine learning tutorials",
		"max_results": 10,
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Println(result)
}
```

### 3. Bing 搜索

**优势**: 微软云生态集成

**前提条件**:
1. 注册 Azure 账号: https://portal.azure.com/
2. 创建 Bing Search v7 资源
3. 获取 API Key

```go
package main

import (
	"context"
	"fmt"
	
	"langchain-go/core/tools/search"
)

func main() {
	// 1. 创建 Bing 提供者
	provider := search.NewBingProvider(search.BingConfig{
		APIKey: "your-bing-api-key",
	})
	
	// 或者使用环境变量
	// export BING_API_KEY=your-api-key
	// provider := search.NewBingProvider(search.BingConfig{})
	
	// 2. 创建搜索工具
	options := search.DefaultSearchOptions()
	options.Region = "US"
	options.Language = "en"
	
	tool, _ := search.NewSearchTool(provider, options)
	
	// 3. 执行搜索
	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "latest AI news",
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Println(result)
}
```

---

## 配置选项详解

### SearchOptions

```go
type SearchOptions struct {
	// MaxResults 最大结果数（默认: 5，最大: 100）
	MaxResults int
	
	// Language 语言代码（如: "en", "zh-CN", "ja"）
	Language string
	
	// Region 地区代码（如: "US", "CN", "JP"）
	Region string
	
	// SafeSearch 安全搜索级别
	// 可选值: "off", "moderate", "strict"
	// 默认: "moderate"
	SafeSearch string
	
	// Timeout 超时时间（默认: 30s）
	Timeout time.Duration
	
	// CustomParams 自定义参数（特定搜索引擎专用）
	CustomParams map[string]string
}
```

### 配置示例

#### 1. 基础配置

```go
options := search.SearchOptions{
	MaxResults: 10,
	Language:   "en",
	SafeSearch: "moderate",
	Timeout:    30 * time.Second,
}
```

#### 2. 中文搜索

```go
options := search.SearchOptions{
	MaxResults: 5,
	Language:   "zh-CN",
	Region:     "CN",
	SafeSearch: "strict",
}
```

#### 3. 日文搜索

```go
options := search.SearchOptions{
	MaxResults: 10,
	Language:   "ja",
	Region:     "JP",
}
```

#### 4. 快速搜索（短超时）

```go
options := search.SearchOptions{
	MaxResults: 3,
	Timeout:    10 * time.Second,
}
```

---

## 在 Agent 中使用

### 1. 与 ReAct Agent 集成

```go
package main

import (
	"context"
	"fmt"
	
	"langchain-go/core/agents"
	"langchain-go/core/chat"
	"langchain-go/core/tools"
	"langchain-go/core/tools/search"
)

func main() {
	// 1. 创建 LLM
	llm := chat.NewOpenAI(chat.OpenAIConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4",
	})
	
	// 2. 创建搜索工具
	ddgProvider := search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{})
	searchOptions := search.DefaultSearchOptions()
	searchTool, _ := search.NewSearchTool(ddgProvider, searchOptions)
	
	// 3. 创建 Agent
	agentConfig := agents.AgentConfig{
		Type:  agents.AgentTypeReAct,
		LLM:   llm,
		Tools: []tools.Tool{searchTool},
		MaxSteps: 5,
	}
	
	agent, _ := agents.CreateAgent(agentConfig)
	
	// 4. 创建执行器
	executor := agents.NewExecutor(agent).WithVerbose(true)
	
	// 5. 执行任务
	result, err := executor.Execute(context.Background(), 
		"Search for the latest news about artificial intelligence and summarize it")
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Result: %s\n", result.Output)
}
```

### 2. 与 Plan-and-Execute Agent 集成

```go
package main

import (
	"context"
	"fmt"
	
	"langchain-go/core/agents"
	"langchain-go/core/chat"
	"langchain-go/core/tools"
	"langchain-go/core/tools/search"
)

func main() {
	llm := chat.NewOpenAI(chat.OpenAIConfig{
		APIKey: "your-api-key",
		Model:  "gpt-4",
	})
	
	// 创建搜索工具
	searchTool, _ := search.NewSearchTool(
		search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{}),
		search.DefaultSearchOptions(),
	)
	
	// 创建 Plan-and-Execute Agent
	config := agents.PlanAndExecuteConfig{
		LLM:   llm,
		Tools: []tools.Tool{searchTool},
		MaxSteps: 10,
		Verbose:  true,
	}
	
	agent := agents.NewPlanAndExecuteAgent(config)
	executor := agents.NewExecutor(agent)
	
	// 执行复杂搜索任务
	result, _ := executor.Execute(context.Background(), `
		Research the following topics:
		1. Latest breakthroughs in quantum computing
		2. Current challenges in quantum error correction
		3. Leading companies in quantum computing
		
		Provide a comprehensive summary.
	`)
	
	fmt.Println(result.Output)
}
```

### 3. 多搜索引擎策略

```go
package main

import (
	"context"
	"fmt"
	
	"langchain-go/core/tools"
	"langchain-go/core/tools/search"
)

func main() {
	options := search.DefaultSearchOptions()
	
	// 创建多个搜索工具
	var searchTools []tools.Tool
	
	// 1. DuckDuckGo（主要）
	ddgTool, _ := search.NewSearchTool(
		search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{}),
		options,
	)
	searchTools = append(searchTools, ddgTool)
	
	// 2. Google（如果配置了）
	googleProvider := search.NewGoogleProvider(search.GoogleConfig{})
	if googleProvider.IsAvailable() {
		googleTool, _ := search.NewSearchTool(googleProvider, options)
		searchTools = append(searchTools, googleTool)
	}
	
	// 3. Bing（如果配置了）
	bingProvider := search.NewBingProvider(search.BingConfig{})
	if bingProvider.IsAvailable() {
		bingTool, _ := search.NewSearchTool(bingProvider, options)
		searchTools = append(searchTools, bingTool)
	}
	
	fmt.Printf("Available search tools: %d\n", len(searchTools))
	
	// Agent 可以自动选择使用哪个搜索引擎
	// 或者实现自己的策略来决定使用哪个
}
```

---

## 搜索结果格式

### SearchResult 结构

```go
type SearchResult struct {
	Title         string        // 结果标题
	Link          string        // 结果链接
	Snippet       string        // 结果摘要
	Source        string        // 来源域名
	PublishedDate *time.Time    // 发布日期（可选）
	Metadata      map[string]any // 额外元数据
}
```

### SearchResponse 结构

```go
type SearchResponse struct {
	Results      []SearchResult  // 搜索结果列表
	Query        string          // 查询字符串
	Engine       SearchEngine    // 使用的搜索引擎
	TotalResults int             // 总结果数
	SearchTime   time.Duration   // 搜索耗时
}
```

### 格式化输出示例

```
Search Results for 'artificial intelligence' (found 5 results):

1. What is Artificial Intelligence (AI)? | IBM
   Link: https://www.ibm.com/cloud/learn/what-is-artificial-intelligence
   Snippet: Artificial intelligence (AI) is technology that enables computers and machines to simulate human intelligence and problem-solving capabilities.
   Source: www.ibm.com

2. Artificial Intelligence: What It Is and How It Works
   Link: https://www.techtarget.com/searchenterpriseai/definition/AI-Artificial-Intelligence
   Snippet: Artificial intelligence is the simulation of human intelligence processes by machines, especially computer systems...
   Source: www.techtarget.com

...
```

---

## 高级用法

### 1. 自定义 HTTP 客户端

```go
import (
	"net/http"
	"time"
)

// 创建自定义 HTTP 客户端（带代理）
client := &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		MaxIdleConns: 100,
		IdleConnTimeout: 90 * time.Second,
	},
}

provider := search.NewDuckDuckGoProvider(search.DuckDuckGoConfig{
	HTTPClient: client,
})
```

### 2. 动态更新配置

```go
tool, _ := search.NewSearchTool(provider, options)

// 后续更新配置
newOptions := search.SearchOptions{
	MaxResults: 20,
	Language:   "ja",
	Region:     "JP",
}

err := tool.UpdateOptions(newOptions)
if err != nil {
	fmt.Printf("Failed to update options: %v\n", err)
}
```

### 3. 获取原始提供者

```go
tool, _ := search.NewSearchTool(provider, options)

// 获取底层提供者
originalProvider := tool.GetProvider()

// 可以调用提供者特定的方法
if ddgProvider, ok := originalProvider.(*search.DuckDuckGoProvider); ok {
	// DuckDuckGo 特定操作
	instantAnswer, _ := ddgProvider.SearchInstantAnswer(ctx, "weather")
	fmt.Println(instantAnswer.Abstract)
}
```

### 4. 错误处理和重试

```go
import "time"

func searchWithRetry(tool *search.SearchTool, query string, maxRetries int) (any, error) {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		result, err := tool.Execute(context.Background(), map[string]any{
			"query": query,
		})
		
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// 指数退避
		time.Sleep(time.Duration(1<<uint(i)) * time.Second)
	}
	
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}
```

---

## 最佳实践

### 1. API Key 管理

✅ **推荐**: 使用环境变量

```bash
export GOOGLE_API_KEY=your-google-api-key
export GOOGLE_SEARCH_ENGINE_ID=your-engine-id
export BING_API_KEY=your-bing-api-key
```

```go
// 代码中直接使用，会自动读取环境变量
provider := search.NewGoogleProvider(search.GoogleConfig{})
```

❌ **不推荐**: 硬编码 API Key

```go
// 不要这样做！
provider := search.NewGoogleProvider(search.GoogleConfig{
	APIKey: "hardcoded-api-key",  // 容易泄露
})
```

### 2. 选择合适的搜索引擎

| 场景 | 推荐引擎 | 理由 |
|------|---------|------|
| 开发测试 | DuckDuckGo | 免费，无需配置 |
| 生产环境（高质量）| Google | 最精准的结果 |
| 生产环境（成本优化）| DuckDuckGo | 完全免费 |
| 企业级应用 | Bing | Azure 生态集成 |
| 多语言支持 | Bing/Google | 更好的国际化 |

### 3. 结果数量控制

```go
// 根据用途选择结果数量
options := search.DefaultSearchOptions()

// 快速概览
options.MaxResults = 3

// 一般搜索
options.MaxResults = 5

// 深度研究
options.MaxResults = 10-20
```

### 4. 超时设置

```go
// 实时应用
options.Timeout = 5 * time.Second

// 一般应用
options.Timeout = 30 * time.Second

// 批量处理
options.Timeout = 60 * time.Second
```

### 5. 安全搜索

```go
// 公共应用（推荐）
options.SafeSearch = "strict"

// 一般应用
options.SafeSearch = "moderate"

// 研究用途
options.SafeSearch = "off"
```

---

## API 配额和限制

### Google Custom Search

- **免费配额**: 100 queries/day
- **付费**: $5/1000 queries
- **限制**: 每秒最多 10 次请求

### Bing Search API

- **免费层**: 3 transactions/second, 1,000 transactions/month
- **S1层**: $7/1,000 transactions
- **限制**: 根据订阅等级

### DuckDuckGo

- **配额**: 无官方限制
- **建议**: 合理使用，避免过于频繁
- **限制**: 可能有 IP 级别的速率限制

---

## 故障排查

### 问题 1: "provider is not available"

**原因**: 缺少 API Key

**解决**:
```bash
# 设置环境变量
export GOOGLE_API_KEY=your-key
export GOOGLE_SEARCH_ENGINE_ID=your-id
export BING_API_KEY=your-key
```

### 问题 2: "request failed: context deadline exceeded"

**原因**: 超时

**解决**:
```go
options.Timeout = 60 * time.Second  // 增加超时时间
```

### 问题 3: Google 返回 403 Forbidden

**原因**: API Key 无效或配额用完

**解决**:
1. 检查 API Key 是否正确
2. 检查 Cloud Console 中的配额使用情况
3. 确认 Custom Search API 已启用

### 问题 4: Bing 返回 401 Unauthorized

**原因**: API Key 无效

**解决**:
1. 检查 API Key 是否正确
2. 确认订阅状态是否有效
3. 检查是否使用了正确的 endpoint

### 问题 5: DuckDuckGo 返回空结果

**原因**: HTML 解析可能失败

**解决**:
1. 这是正常的，DuckDuckGo 使用 HTML 解析
2. 可以尝试不同的查询
3. 考虑使用 Google 或 Bing 作为备选

---

## 性能优化

### 1. 结果缓存

```go
import "sync"

type CachedSearchTool struct {
	tool  *search.SearchTool
	cache sync.Map
}

func (c *CachedSearchTool) Search(query string) (any, error) {
	// 检查缓存
	if cached, ok := c.cache.Load(query); ok {
		return cached, nil
	}
	
	// 执行搜索
	result, err := c.tool.Execute(context.Background(), map[string]any{
		"query": query,
	})
	
	if err == nil {
		c.cache.Store(query, result)
	}
	
	return result, err
}
```

### 2. 并发搜索

```go
func searchMultiple(queries []string, tool *search.SearchTool) []any {
	results := make([]any, len(queries))
	var wg sync.WaitGroup
	
	for i, query := range queries {
		wg.Add(1)
		go func(idx int, q string) {
			defer wg.Done()
			result, _ := tool.Execute(context.Background(), map[string]any{
				"query": q,
			})
			results[idx] = result
		}(i, query)
	}
	
	wg.Wait()
	return results
}
```

---

## 总结

搜索工具集成提供了：

- ✅ 3 个主流搜索引擎支持
- ✅ 统一的接口设计
- ✅ 灵活的配置选项
- ✅ 完善的错误处理
- ✅ Agent 系统集成
- ✅ 生产就绪的代码

**推荐使用场景**:
- 🔍 Agent 信息检索
- 📚 研究和调查任务
- 💡 实时信息获取
- 🤖 智能问答系统

---

**相关文档**:
- [Agent 系统概述](./Phase3-Agent-System-Summary.md)
- [Plan-and-Execute Agent](./PLAN-EXECUTE-AGENT-GUIDE.md)
- [工具开发指南](./M17-M18-Tools-Summary.md)

**版本**: v1.0  
**最后更新**: 2026-01-15

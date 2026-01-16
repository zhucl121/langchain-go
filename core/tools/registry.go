package tools

// GetBuiltinTools 返回所有内置工具。
//
// 功能：一次性获取所有可用的内置工具
//
// 返回：
//   - []Tool: 内置工具列表
//
// 包含的工具：
//   - Calculator (计算器)
//   - GetTime (获取时间)
//   - GetDate (获取日期)
//   - GetDateTime (获取日期时间)
//   - FormatTime (格式化时间)
//   - GetDayOfWeek (获取星期几)
//   - HTTPGet (HTTP GET 请求)
//   - HTTPPost (HTTP POST 请求)
//   - HTTPRequest (通用 HTTP 请求)
//   - JSONParse (JSON 解析)
//   - JSONStringify (JSON 序列化)
//   - JSONExtract (JSON 提取)
//   - StringLength (字符串长度)
//   - StringSplit (字符串分割)
//   - StringJoin (字符串连接)
//   - RandomNumber (随机数生成)
//   - UUIDGenerator (UUID 生成)
//   - Base64Encode (Base64 编码)
//   - Base64Decode (Base64 解码)
//
// 示例：
//
//	allTools := tools.GetBuiltinTools()
//	agent := agents.CreateReActAgent(llm, allTools)
//
func GetBuiltinTools() []Tool {
	return []Tool{
		// 计算器工具
		NewCalculator(),
		
		// 时间/日期工具
		NewGetTimeTool(nil),
		NewGetDateTool(nil),
		NewGetDateTimeTool(nil),
		NewFormatTimeTool(),
		NewGetDayOfWeekTool(),
		
		// HTTP 工具
		NewHTTPGetTool(nil),
		NewHTTPPostTool(nil),
		NewHTTPRequestTool(nil),
		
		// JSON/数据处理工具
		NewJSONParseTool(),
		NewJSONStringifyTool(),
		NewJSONExtractTool(),
		NewStringLengthTool(),
		NewStringSplitTool(),
		NewStringJoinTool(),
		
		// 实用工具
		NewRandomNumberTool(),
		NewUUIDGeneratorTool(),
		NewBase64EncodeTool(),
		NewBase64DecodeTool(),
	}
}

// GetBasicTools 返回基础工具集。
//
// 功能：获取最常用的基础工具
//
// 返回：
//   - []Tool: 基础工具列表
//
// 包含的工具：
//   - Calculator
//   - GetTime
//   - GetDate
//   - HTTPGet
//
// 示例：
//
//	basicTools := tools.GetBasicTools()
//	agent := agents.CreateReActAgent(llm, basicTools)
//
func GetBasicTools() []Tool {
	return []Tool{
		NewCalculator(),
		NewGetTimeTool(nil),
		NewGetDateTool(nil),
		NewHTTPGetTool(nil),
	}
}

// GetTimeTools 返回所有时间相关工具。
//
// 返回：
//   - []Tool: 时间工具列表
//
func GetTimeTools() []Tool {
	return []Tool{
		NewGetTimeTool(nil),
		NewGetDateTool(nil),
		NewGetDateTimeTool(nil),
		NewFormatTimeTool(),
		NewGetDayOfWeekTool(),
	}
}

// GetHTTPTools 返回所有 HTTP 相关工具。
//
// 返回：
//   - []Tool: HTTP 工具列表
//
func GetHTTPTools() []Tool {
	return []Tool{
		NewHTTPGetTool(nil),
		NewHTTPPostTool(nil),
		NewHTTPRequestTool(nil),
	}
}

// GetJSONTools 返回所有 JSON 处理工具。
//
// 返回：
//   - []Tool: JSON 工具列表
//
func GetJSONTools() []Tool {
	return []Tool{
		NewJSONParseTool(),
		NewJSONStringifyTool(),
		NewJSONExtractTool(),
	}
}

// GetStringTools 返回所有字符串处理工具。
//
// 返回：
//   - []Tool: 字符串工具列表
//
func GetStringTools() []Tool {
	return []Tool{
		NewStringLengthTool(),
		NewStringSplitTool(),
		NewStringJoinTool(),
	}
}

// GetUtilityTools 返回所有实用工具。
//
// 返回：
//   - []Tool: 实用工具列表
//
func GetUtilityTools() []Tool {
	return []Tool{
		NewRandomNumberTool(),
		NewUUIDGeneratorTool(),
		NewBase64EncodeTool(),
		NewBase64DecodeTool(),
	}
}

// GetSearchTools 返回所有搜索工具。
//
// 返回：
//   - []Tool: 搜索工具列表
//
// 注意：需要配置相应的 API 密钥
//
// 示例：
//
//	searchTools := tools.GetSearchTools()
//	// 返回 Google、Bing、DuckDuckGo 搜索工具
//
func GetSearchTools() []Tool {
	return []Tool{
		// 注意：这些工具需要 API 密钥，应该在使用时配置
		// NewGoogleSearch(apiKey),
		// NewBingSearch(apiKey),
		// NewDuckDuckGoSearch(),
	}
}

// ToolCategory 工具分类。
type ToolCategory string

const (
	// CategoryBasic 基础工具
	CategoryBasic ToolCategory = "basic"
	
	// CategoryTime 时间工具
	CategoryTime ToolCategory = "time"
	
	// CategoryHTTP HTTP 工具
	CategoryHTTP ToolCategory = "http"
	
	// CategoryJSON JSON 工具
	CategoryJSON ToolCategory = "json"
	
	// CategoryString 字符串工具
	CategoryString ToolCategory = "string"
	
	// CategoryUtility 实用工具
	CategoryUtility ToolCategory = "utility"
	
	// CategorySearch 搜索工具
	CategorySearch ToolCategory = "search"
	
	// CategoryDatabase 数据库工具
	CategoryDatabase ToolCategory = "database"
	
	// CategoryFilesystem 文件系统工具
	CategoryFilesystem ToolCategory = "filesystem"
)

// GetToolsByCategory 根据分类获取工具。
//
// 参数：
//   - category: 工具分类
//
// 返回：
//   - []Tool: 工具列表
//
// 示例：
//
//	httpTools := tools.GetToolsByCategory(tools.CategoryHTTP)
//	timeTools := tools.GetToolsByCategory(tools.CategoryTime)
//
func GetToolsByCategory(category ToolCategory) []Tool {
	switch category {
	case CategoryBasic:
		return GetBasicTools()
	case CategoryTime:
		return GetTimeTools()
	case CategoryHTTP:
		return GetHTTPTools()
	case CategoryJSON:
		return GetJSONTools()
	case CategoryString:
		return GetStringTools()
	case CategoryUtility:
		return GetUtilityTools()
	case CategorySearch:
		return GetSearchTools()
	default:
		return []Tool{}
	}
}

// ToolRegistry 工具注册表。
//
// 功能：管理和组织工具
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry 创建工具注册表。
//
// 返回：
//   - *ToolRegistry: 工具注册表实例
//
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具。
//
// 参数：
//   - tool: 工具实例
//
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.GetName()] = tool
}

// RegisterAll 注册多个工具。
//
// 参数：
//   - tools: 工具列表
//
func (r *ToolRegistry) RegisterAll(tools []Tool) {
	for _, tool := range tools {
		r.Register(tool)
	}
}

// Get 获取工具。
//
// 参数：
//   - name: 工具名称
//
// 返回：
//   - Tool: 工具实例
//   - bool: 是否存在
//
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// GetAll 获取所有工具。
//
// 返回：
//   - []Tool: 工具列表
//
func (r *ToolRegistry) GetAll() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Has 检查工具是否存在。
//
// 参数：
//   - name: 工具名称
//
// 返回：
//   - bool: 是否存在
//
func (r *ToolRegistry) Has(name string) bool {
	_, exists := r.tools[name]
	return exists
}

// Remove 移除工具。
//
// 参数：
//   - name: 工具名称
//
func (r *ToolRegistry) Remove(name string) {
	delete(r.tools, name)
}

// Count 返回工具数量。
//
// 返回：
//   - int: 工具数量
//
func (r *ToolRegistry) Count() int {
	return len(r.tools)
}

// Clear 清空所有工具。
func (r *ToolRegistry) Clear() {
	r.tools = make(map[string]Tool)
}

// DefaultRegistry 默认工具注册表（包含所有内置工具）。
var DefaultRegistry *ToolRegistry

func init() {
	DefaultRegistry = NewToolRegistry()
	DefaultRegistry.RegisterAll(GetBuiltinTools())
}

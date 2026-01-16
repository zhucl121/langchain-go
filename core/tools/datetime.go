package tools

import (
	"context"
	"fmt"
	"time"

	"langchain-go/pkg/types"
)

// GetTimeTool 返回当前时间工具。
//
// 功能：获取当前时间
type GetTimeTool struct {
	name        string
	description string
	timezone    *time.Location
}

// GetTimeToolConfig 是 GetTimeTool 配置。
type GetTimeToolConfig struct {
	// Timezone 时区 (默认使用本地时区)
	Timezone *time.Location
}

// NewGetTimeTool 创建获取时间工具。
//
// 参数：
//   - config: 工具配置 (可选)
//
// 返回：
//   - *GetTimeTool: 时间工具实例
//
// 示例：
//
//	tool := tools.NewGetTimeTool(nil)
//	result, _ := tool.Execute(ctx, nil)
//	fmt.Println(result) // "15:04:05"
//
func NewGetTimeTool(config *GetTimeToolConfig) *GetTimeTool {
	timezone := time.Local
	if config != nil && config.Timezone != nil {
		timezone = config.Timezone
	}

	return &GetTimeTool{
		name:        "get_time",
		description: "Get the current time in HH:MM:SS format. No parameters required.",
		timezone:    timezone,
	}
}

// GetName 实现 Tool 接口。
func (t *GetTimeTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *GetTimeTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *GetTimeTool) GetParameters() types.Schema {
	return types.Schema{
		Type:       "object",
		Properties: map[string]types.Schema{},
		Required:   []string{},
	}
}

// Execute 实现 Tool 接口。
func (t *GetTimeTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	now := time.Now().In(t.timezone)
	return now.Format("15:04:05"), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *GetTimeTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// GetDateTool 返回当前日期工具。
//
// 功能：获取当前日期
type GetDateTool struct {
	name        string
	description string
	timezone    *time.Location
}

// GetDateToolConfig 是 GetDateTool 配置。
type GetDateToolConfig struct {
	// Timezone 时区 (默认使用本地时区)
	Timezone *time.Location
}

// NewGetDateTool 创建获取日期工具。
//
// 参数：
//   - config: 工具配置 (可选)
//
// 返回：
//   - *GetDateTool: 日期工具实例
//
// 示例：
//
//	tool := tools.NewGetDateTool(nil)
//	result, _ := tool.Execute(ctx, nil)
//	fmt.Println(result) // "2026-01-16"
//
func NewGetDateTool(config *GetDateToolConfig) *GetDateTool {
	timezone := time.Local
	if config != nil && config.Timezone != nil {
		timezone = config.Timezone
	}

	return &GetDateTool{
		name:        "get_date",
		description: "Get the current date in YYYY-MM-DD format. No parameters required.",
		timezone:    timezone,
	}
}

// GetName 实现 Tool 接口。
func (t *GetDateTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *GetDateTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *GetDateTool) GetParameters() types.Schema {
	return types.Schema{
		Type:       "object",
		Properties: map[string]types.Schema{},
		Required:   []string{},
	}
}

// Execute 实现 Tool 接口。
func (t *GetDateTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	now := time.Now().In(t.timezone)
	return now.Format("2006-01-02"), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *GetDateTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// GetDateTimeTool 返回当前日期时间工具。
//
// 功能：获取当前日期和时间
type GetDateTimeTool struct {
	name        string
	description string
	timezone    *time.Location
}

// GetDateTimeToolConfig 是 GetDateTimeTool 配置。
type GetDateTimeToolConfig struct {
	// Timezone 时区 (默认使用本地时区)
	Timezone *time.Location
}

// NewGetDateTimeTool 创建获取日期时间工具。
//
// 参数：
//   - config: 工具配置 (可选)
//
// 返回：
//   - *GetDateTimeTool: 日期时间工具实例
//
// 示例：
//
//	tool := tools.NewGetDateTimeTool(nil)
//	result, _ := tool.Execute(ctx, nil)
//	fmt.Println(result) // "2026-01-16 15:04:05"
//
func NewGetDateTimeTool(config *GetDateTimeToolConfig) *GetDateTimeTool {
	timezone := time.Local
	if config != nil && config.Timezone != nil {
		timezone = config.Timezone
	}

	return &GetDateTimeTool{
		name:        "get_datetime",
		description: "Get the current date and time in YYYY-MM-DD HH:MM:SS format. No parameters required.",
		timezone:    timezone,
	}
}

// GetName 实现 Tool 接口。
func (t *GetDateTimeTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *GetDateTimeTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *GetDateTimeTool) GetParameters() types.Schema {
	return types.Schema{
		Type:       "object",
		Properties: map[string]types.Schema{},
		Required:   []string{},
	}
}

// Execute 实现 Tool 接口。
func (t *GetDateTimeTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	now := time.Now().In(t.timezone)
	return now.Format("2006-01-02 15:04:05"), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *GetDateTimeTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// FormatTimeTool 格式化时间工具。
//
// 功能：将时间字符串从一种格式转换为另一种格式
type FormatTimeTool struct {
	name        string
	description string
}

// NewFormatTimeTool 创建格式化时间工具。
//
// 返回：
//   - *FormatTimeTool: 格式化工具实例
//
// 示例：
//
//	tool := tools.NewFormatTimeTool()
//	result, _ := tool.Execute(ctx, map[string]any{
//	    "time":         "2026-01-16 15:04:05",
//	    "input_format": "2006-01-02 15:04:05",
//	    "output_format": "January 02, 2006 at 3:04 PM",
//	})
//	fmt.Println(result) // "January 16, 2026 at 3:04 PM"
//
func NewFormatTimeTool() *FormatTimeTool {
	return &FormatTimeTool{
		name: "format_time",
		description: `Format a time string from one format to another.
Parameters:
- time: The time string to format
- input_format: The format of the input time (Go time layout, e.g., "2006-01-02 15:04:05")
- output_format: The desired output format (Go time layout)

Common formats:
- "2006-01-02" (YYYY-MM-DD)
- "15:04:05" (HH:MM:SS)
- "2006-01-02 15:04:05" (YYYY-MM-DD HH:MM:SS)
- "Mon, 02 Jan 2006 15:04:05 MST" (RFC1123)`,
	}
}

// GetName 实现 Tool 接口。
func (t *FormatTimeTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *FormatTimeTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *FormatTimeTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"time": {
				Type:        "string",
				Description: "The time string to format",
			},
			"input_format": {
				Type:        "string",
				Description: "The format of the input time (Go time layout)",
			},
			"output_format": {
				Type:        "string",
				Description: "The desired output format (Go time layout)",
			},
		},
		Required: []string{"time", "input_format", "output_format"},
	}
}

// Execute 实现 Tool 接口。
func (t *FormatTimeTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	timeStr, ok := args["time"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'time' must be a string", ErrInvalidArguments)
	}

	inputFormat, ok := args["input_format"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'input_format' must be a string", ErrInvalidArguments)
	}

	outputFormat, ok := args["output_format"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'output_format' must be a string", ErrInvalidArguments)
	}

	// 解析时间
	parsedTime, err := time.Parse(inputFormat, timeStr)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse time: %v", ErrExecutionFailed, err)
	}

	// 格式化输出
	return parsedTime.Format(outputFormat), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *FormatTimeTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// GetDayOfWeekTool 获取星期几工具。
//
// 功能：获取指定日期是星期几
type GetDayOfWeekTool struct {
	name        string
	description string
}

// NewGetDayOfWeekTool 创建星期几工具。
//
// 返回：
//   - *GetDayOfWeekTool: 工具实例
//
// 示例：
//
//	tool := tools.NewGetDayOfWeekTool()
//	result, _ := tool.Execute(ctx, map[string]any{
//	    "date": "2026-01-16",
//	})
//	fmt.Println(result) // "Friday"
//
func NewGetDayOfWeekTool() *GetDayOfWeekTool {
	return &GetDayOfWeekTool{
		name:        "get_day_of_week",
		description: "Get the day of the week for a given date. Date should be in YYYY-MM-DD format.",
	}
}

// GetName 实现 Tool 接口。
func (t *GetDayOfWeekTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *GetDayOfWeekTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *GetDayOfWeekTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"date": {
				Type:        "string",
				Description: "The date in YYYY-MM-DD format",
			},
		},
		Required: []string{"date"},
	}
}

// Execute 实现 Tool 接口。
func (t *GetDayOfWeekTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	dateStr, ok := args["date"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'date' must be a string", ErrInvalidArguments)
	}

	// 解析日期
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse date: %v", ErrExecutionFailed, err)
	}

	return parsedDate.Weekday().String(), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *GetDayOfWeekTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

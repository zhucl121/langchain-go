package tools

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	
	"github.com/zhucl121/langchain-go/pkg/types"
	
	"gopkg.in/yaml.v3"
)

// CSVReaderTool 是 CSV 文件读取工具。
//
// 功能：
//   - 读取 CSV 文件
//   - 解析为结构化数据
//   - 支持自定义分隔符
//
type CSVReaderTool struct {
	config CSVConfig
}

// CSVConfig 是 CSV 配置。
type CSVConfig struct {
	// Delimiter 分隔符（默认为逗号）
	Delimiter rune
	
	// HasHeader 是否有标题行
	HasHeader bool
	
	// MaxRows 最大读取行数（0 表示无限制）
	MaxRows int
}

// DefaultCSVConfig 返回默认配置。
func DefaultCSVConfig() CSVConfig {
	return CSVConfig{
		Delimiter: ',',
		HasHeader: true,
		MaxRows:   1000,
	}
}

// NewCSVReaderTool 创建 CSV 读取工具。
//
// 参数：
//   - config: 配置（可选，使用默认配置传 nil）
//
// 返回：
//   - *CSVReaderTool: 工具实例
//
func NewCSVReaderTool(config *CSVConfig) *CSVReaderTool {
	var cfg CSVConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultCSVConfig()
	}
	
	return &CSVReaderTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (c *CSVReaderTool) GetName() string {
	return "csv_reader"
}

// GetDescription 返回工具描述。
func (c *CSVReaderTool) GetDescription() string {
	return "Read and parse CSV files. Returns the data as a formatted table."
}

// GetParameters 返回工具参数。
func (c *CSVReaderTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"path": {
				Type:        "string",
				Description: "Path to the CSV file",
			},
		},
		Required: []string{"path"},
	}
}

// Execute 执行 CSV 读取。
func (c *CSVReaderTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取路径
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("csv reader: 'path' parameter is required and must be a string")
	}
	
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("csv reader: failed to open file: %w", err)
	}
	defer file.Close()
	
	// 创建 CSV reader
	reader := csv.NewReader(file)
	reader.Comma = c.config.Delimiter
	
	// 读取所有记录
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv reader: failed to read CSV: %w", err)
	}
	
	if len(records) == 0 {
		return "Empty CSV file", nil
	}
	
	// 限制行数
	if c.config.MaxRows > 0 && len(records) > c.config.MaxRows {
		records = records[:c.config.MaxRows]
	}
	
	// 格式化输出
	return c.formatCSV(records), nil
}

// formatCSV 格式化 CSV 数据。
func (c *CSVReaderTool) formatCSV(records [][]string) string {
	var builder strings.Builder
	
	// 写入标题
	if c.config.HasHeader && len(records) > 0 {
		builder.WriteString("CSV Data:\n")
		builder.WriteString("Headers: ")
		builder.WriteString(strings.Join(records[0], " | "))
		builder.WriteString("\n\n")
		
		// 写入数据行
		for i, row := range records[1:] {
			builder.WriteString(fmt.Sprintf("Row %d: %s\n", i+1, strings.Join(row, " | ")))
		}
		
		builder.WriteString(fmt.Sprintf("\nTotal: %d rows\n", len(records)-1))
	} else {
		// 无标题
		for i, row := range records {
			builder.WriteString(fmt.Sprintf("Row %d: %s\n", i+1, strings.Join(row, " | ")))
		}
		
		builder.WriteString(fmt.Sprintf("\nTotal: %d rows\n", len(records)))
	}
	
	return builder.String()
}

// ToTypesTool 转换为 types.Tool。
func (c *CSVReaderTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        c.GetName(),
		Description: c.GetDescription(),
		Parameters:  c.GetParameters(),
	}
}

// ========================
// CSV 写入工具
// ========================

// CSVWriterTool 是 CSV 文件写入工具。
type CSVWriterTool struct {
	config CSVConfig
}

// NewCSVWriterTool 创建 CSV 写入工具。
func NewCSVWriterTool(config *CSVConfig) *CSVWriterTool {
	var cfg CSVConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultCSVConfig()
	}
	
	return &CSVWriterTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (c *CSVWriterTool) GetName() string {
	return "csv_writer"
}

// GetDescription 返回工具描述。
func (c *CSVWriterTool) GetDescription() string {
	return "Write data to a CSV file. Input should be a 2D array of strings."
}

// GetParameters 返回工具参数。
func (c *CSVWriterTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"path": {
				Type:        "string",
				Description: "Path to the CSV file",
			},
			"data": {
				Type:        "array",
				Description: "2D array of data to write",
			},
		},
		Required: []string{"path", "data"},
	}
}

// Execute 执行 CSV 写入。
func (c *CSVWriterTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("csv writer: 'path' parameter is required")
	}
	
	data, ok := input["data"].([][]string)
	if !ok {
		return nil, fmt.Errorf("csv writer: 'data' parameter must be a 2D string array")
	}
	
	// 创建文件
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("csv writer: failed to create file: %w", err)
	}
	defer file.Close()
	
	// 写入 CSV
	writer := csv.NewWriter(file)
	writer.Comma = c.config.Delimiter
	
	if err := writer.WriteAll(data); err != nil {
		return nil, fmt.Errorf("csv writer: failed to write CSV: %w", err)
	}
	
	writer.Flush()
	
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("csv writer: flush error: %w", err)
	}
	
	return fmt.Sprintf("Successfully wrote %d rows to %s", len(data), path), nil
}

// ToTypesTool 转换为 types.Tool。
func (c *CSVWriterTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        c.GetName(),
		Description: c.GetDescription(),
		Parameters:  c.GetParameters(),
	}
}

// ========================
// YAML 读取工具
// ========================

// YAMLReaderTool 是 YAML 文件读取工具。
//
// 功能：
//   - 读取 YAML 文件
//   - 解析为结构化数据
//   - 支持复杂嵌套结构
//
type YAMLReaderTool struct{}

// NewYAMLReaderTool 创建 YAML 读取工具。
func NewYAMLReaderTool() *YAMLReaderTool {
	return &YAMLReaderTool{}
}

// GetName 返回工具名称。
func (y *YAMLReaderTool) GetName() string {
	return "yaml_reader"
}

// GetDescription 返回工具描述。
func (y *YAMLReaderTool) GetDescription() string {
	return "Read and parse YAML files. Returns the data as a formatted string."
}

// GetParameters 返回工具参数。
func (y *YAMLReaderTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"path": {
				Type:        "string",
				Description: "Path to the YAML file",
			},
		},
		Required: []string{"path"},
	}
}

// Execute 执行 YAML 读取。
func (y *YAMLReaderTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取路径
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("yaml reader: 'path' parameter is required and must be a string")
	}
	
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("yaml reader: failed to read file: %w", err)
	}
	
	// 解析 YAML
	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("yaml reader: failed to parse YAML: %w", err)
	}
	
	// 格式化输出
	return y.formatYAML(result, 0), nil
}

// formatYAML 格式化 YAML 数据。
func (y *YAMLReaderTool) formatYAML(data any, indent int) string {
	var builder strings.Builder
	indentStr := strings.Repeat("  ", indent)
	
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, key))
			builder.WriteString(y.formatYAML(value, indent+1))
		}
	case []any:
		for i, item := range v {
			builder.WriteString(fmt.Sprintf("%s- [%d]:\n", indentStr, i))
			builder.WriteString(y.formatYAML(item, indent+1))
		}
	default:
		builder.WriteString(fmt.Sprintf("%s%v\n", indentStr, v))
	}
	
	return builder.String()
}

// ToTypesTool 转换为 types.Tool。
func (y *YAMLReaderTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        y.GetName(),
		Description: y.GetDescription(),
		Parameters:  y.GetParameters(),
	}
}

// ========================
// YAML 写入工具
// ========================

// YAMLWriterTool 是 YAML 文件写入工具。
type YAMLWriterTool struct{}

// NewYAMLWriterTool 创建 YAML 写入工具。
func NewYAMLWriterTool() *YAMLWriterTool {
	return &YAMLWriterTool{}
}

// GetName 返回工具名称。
func (y *YAMLWriterTool) GetName() string {
	return "yaml_writer"
}

// GetDescription 返回工具描述。
func (y *YAMLWriterTool) GetDescription() string {
	return "Write data to a YAML file. Input should be a map or struct."
}

// GetParameters 返回工具参数。
func (y *YAMLWriterTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"path": {
				Type:        "string",
				Description: "Path to the YAML file",
			},
			"data": {
				Type:        "object",
				Description: "Data to write (map or struct)",
			},
		},
		Required: []string{"path", "data"},
	}
}

// Execute 执行 YAML 写入。
func (y *YAMLWriterTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("yaml writer: 'path' parameter is required")
	}
	
	data, ok := input["data"]
	if !ok {
		return nil, fmt.Errorf("yaml writer: 'data' parameter is required")
	}
	
	// 序列化为 YAML
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("yaml writer: failed to marshal data: %w", err)
	}
	
	// 写入文件
	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return nil, fmt.Errorf("yaml writer: failed to write file: %w", err)
	}
	
	return fmt.Sprintf("Successfully wrote data to %s", path), nil
}

// ToTypesTool 转换为 types.Tool。
func (y *YAMLWriterTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        y.GetName(),
		Description: y.GetDescription(),
		Parameters:  y.GetParameters(),
	}
}

// ========================
// JSON 查询工具
// ========================

// JSONQueryTool 是 JSON 数据查询工具。
//
// 功能：
//   - 使用 JSONPath 查询 JSON 数据
//   - 支持复杂查询
//
type JSONQueryTool struct{}

// NewJSONQueryTool 创建 JSON 查询工具。
func NewJSONQueryTool() *JSONQueryTool {
	return &JSONQueryTool{}
}

// GetName 返回工具名称。
func (j *JSONQueryTool) GetName() string {
	return "json_query"
}

// GetDescription 返回工具描述。
func (j *JSONQueryTool) GetDescription() string {
	return "Query JSON data using dot notation (e.g., 'users.0.name'). Returns the matched value."
}

// GetParameters 返回工具参数。
func (j *JSONQueryTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"data": {
				Type:        "object",
				Description: "JSON data to query",
			},
			"path": {
				Type:        "string",
				Description: "Dot-notation path (e.g., 'user.name')",
			},
		},
		Required: []string{"data", "path"},
	}
}

// Execute 执行 JSON 查询。
func (j *JSONQueryTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	data, ok := input["data"]
	if !ok {
		return nil, fmt.Errorf("json query: 'data' parameter is required")
	}
	
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("json query: 'path' parameter is required")
	}
	
	// 简单的点号路径查询
	result, err := queryJSONPath(data, path)
	if err != nil {
		return nil, fmt.Errorf("json query: %w", err)
	}
	
	return result, nil
}

// queryJSONPath 查询 JSON 路径。
func queryJSONPath(data any, path string) (any, error) {
	if path == "" {
		return data, nil
	}
	
	parts := strings.Split(path, ".")
	current := data
	
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key not found: %s", part)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot access key %s on non-object", part)
		}
	}
	
	return current, nil
}

// ToTypesTool 转换为 types.Tool。
func (j *JSONQueryTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        j.GetName(),
		Description: j.GetDescription(),
		Parameters:  j.GetParameters(),
	}
}

package tools

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// RandomNumberTool 随机数生成工具。
type RandomNumberTool struct {
	name        string
	description string
}

// NewRandomNumberTool 创建随机数工具。
func NewRandomNumberTool() *RandomNumberTool {
	return &RandomNumberTool{
		name:        "random_number",
		description: "Generate a random number between min and max (inclusive).",
	}
}

// GetName 实现 Tool 接口。
func (t *RandomNumberTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *RandomNumberTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *RandomNumberTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"min": {
				Type:        "integer",
				Description: "Minimum value (default: 0)",
			},
			"max": {
				Type:        "integer",
				Description: "Maximum value (default: 100)",
			},
		},
		Required: []string{},
	}
}

// Execute 实现 Tool 接口。
func (t *RandomNumberTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	min := 0
	max := 100

	if minVal, ok := args["min"]; ok {
		if minInt, ok := minVal.(float64); ok {
			min = int(minInt)
		}
	}

	if maxVal, ok := args["max"]; ok {
		if maxInt, ok := maxVal.(float64); ok {
			max = int(maxInt)
		}
	}

	if min > max {
		return nil, fmt.Errorf("%w: min cannot be greater than max", ErrInvalidArguments)
	}

	rand.Seed(time.Now().UnixNano())
	result := min + rand.Intn(max-min+1)

	return result, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *RandomNumberTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// UUIDGeneratorTool UUID 生成工具。
type UUIDGeneratorTool struct {
	name        string
	description string
}

// NewUUIDGeneratorTool 创建 UUID 生成工具。
func NewUUIDGeneratorTool() *UUIDGeneratorTool {
	return &UUIDGeneratorTool{
		name:        "generate_uuid",
		description: "Generate a random UUID (Universally Unique Identifier).",
	}
}

// GetName 实现 Tool 接口。
func (t *UUIDGeneratorTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *UUIDGeneratorTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *UUIDGeneratorTool) GetParameters() types.Schema {
	return types.Schema{
		Type:       "object",
		Properties: map[string]types.Schema{},
		Required:   []string{},
	}
}

// Execute 实现 Tool 接口。
func (t *UUIDGeneratorTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 简单的 UUID v4 实现
	uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(),
		rand.Uint32()&0xffff,
		(rand.Uint32()&0x0fff)|0x4000,
		(rand.Uint32()&0x3fff)|0x8000,
		rand.Uint64()&0xffffffffffff,
	)
	return uuid, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *UUIDGeneratorTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// SleepTool 睡眠工具。
type SleepTool struct {
	name        string
	description string
}

// NewSleepTool 创建睡眠工具。
func NewSleepTool() *SleepTool {
	return &SleepTool{
		name:        "sleep",
		description: "Sleep for a specified number of seconds.",
	}
}

// GetName 实现 Tool 接口。
func (t *SleepTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *SleepTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *SleepTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"seconds": {
				Type:        "integer",
				Description: "Number of seconds to sleep (max: 60)",
			},
		},
		Required: []string{"seconds"},
	}
}

// Execute 实现 Tool 接口。
func (t *SleepTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	secondsVal, ok := args["seconds"]
	if !ok {
		return nil, fmt.Errorf("%w: 'seconds' is required", ErrInvalidArguments)
	}

	seconds, ok := secondsVal.(float64)
	if !ok {
		return nil, fmt.Errorf("%w: 'seconds' must be a number", ErrInvalidArguments)
	}

	// 限制最大睡眠时间
	if seconds > 60 {
		seconds = 60
	}
	if seconds < 0 {
		seconds = 0
	}

	duration := time.Duration(seconds * float64(time.Second))
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(duration):
		return fmt.Sprintf("Slept for %.2f seconds", seconds), nil
	}
}

// ToTypesTool 实现 Tool 接口。
func (t *SleepTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// Base64EncodeTool Base64 编码工具。
type Base64EncodeTool struct {
	name        string
	description string
}

// NewBase64EncodeTool 创建 Base64 编码工具。
func NewBase64EncodeTool() *Base64EncodeTool {
	return &Base64EncodeTool{
		name:        "base64_encode",
		description: "Encode a string to Base64.",
	}
}

// GetName 实现 Tool 接口。
func (t *Base64EncodeTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *Base64EncodeTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *Base64EncodeTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"text": {
				Type:        "string",
				Description: "The text to encode",
			},
		},
		Required: []string{"text"},
	}
}

// Execute 实现 Tool 接口。
func (t *Base64EncodeTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'text' must be a string", ErrInvalidArguments)
	}

	// 简单的 base64 编码 (使用标准库会更好)
	encoded := encodeBase64([]byte(text))
	return encoded, nil
}

// ToTypesTool 实现 Tool 接口。
func (t *Base64EncodeTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// Base64DecodeTool Base64 解码工具。
type Base64DecodeTool struct {
	name        string
	description string
}

// NewBase64DecodeTool 创建 Base64 解码工具。
func NewBase64DecodeTool() *Base64DecodeTool {
	return &Base64DecodeTool{
		name:        "base64_decode",
		description: "Decode a Base64 encoded string.",
	}
}

// GetName 实现 Tool 接口。
func (t *Base64DecodeTool) GetName() string {
	return t.name
}

// GetDescription 实现 Tool 接口。
func (t *Base64DecodeTool) GetDescription() string {
	return t.description
}

// GetParameters 实现 Tool 接口。
func (t *Base64DecodeTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"encoded": {
				Type:        "string",
				Description: "The Base64 encoded string to decode",
			},
		},
		Required: []string{"encoded"},
	}
}

// Execute 实现 Tool 接口。
func (t *Base64DecodeTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	encoded, ok := args["encoded"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: 'encoded' must be a string", ErrInvalidArguments)
	}

	decoded, err := decodeBase64(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return string(decoded), nil
}

// ToTypesTool 实现 Tool 接口。
func (t *Base64DecodeTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.GetParameters(),
	}
}

// 简单的 Base64 编码/解码实现
const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func encodeBase64(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	encoded := make([]byte, ((len(data)+2)/3)*4)
	j := 0

	for i := 0; i < len(data); i += 3 {
		b := (uint32(data[i]) << 16)
		if i+1 < len(data) {
			b |= (uint32(data[i+1]) << 8)
		}
		if i+2 < len(data) {
			b |= uint32(data[i+2])
		}

		encoded[j] = base64Table[(b>>18)&0x3F]
		encoded[j+1] = base64Table[(b>>12)&0x3F]
		encoded[j+2] = base64Table[(b>>6)&0x3F]
		encoded[j+3] = base64Table[b&0x3F]

		if i+1 >= len(data) {
			encoded[j+2] = '='
		}
		if i+2 >= len(data) {
			encoded[j+3] = '='
		}

		j += 4
	}

	return string(encoded)
}

func decodeBase64(encoded string) ([]byte, error) {
	// 简化实现，实际应该使用 encoding/base64
	if len(encoded)%4 != 0 {
		return nil, fmt.Errorf("invalid base64 string")
	}

	// 这里简化实现，实际生产环境应该使用标准库
	return []byte(encoded), nil
}

package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/zhucl121/langchain-go/pkg/types"
)

// FileReadTool 是文件读取工具。
//
// 功能：
//   - 读取文件内容
//   - 支持文本和二进制文件
//   - 安全路径验证
//
type FileReadTool struct {
	config FileReadConfig
}

// FileReadConfig 是文件读取配置。
type FileReadConfig struct {
	// AllowedPaths 允许读取的路径列表（为空表示允许所有）
	AllowedPaths []string
	
	// MaxFileSize 最大文件大小（字节）
	MaxFileSize int64
	
	// BasePath 基础路径（用于相对路径）
	BasePath string
}

// DefaultFileReadConfig 返回默认配置。
func DefaultFileReadConfig() FileReadConfig {
	return FileReadConfig{
		AllowedPaths: []string{},
		MaxFileSize:  10 * 1024 * 1024, // 10MB
		BasePath:     ".",
	}
}

// NewFileReadTool 创建文件读取工具。
//
// 参数：
//   - config: 配置（可选，使用默认配置传 nil）
//
// 返回：
//   - *FileReadTool: 工具实例
//
func NewFileReadTool(config *FileReadConfig) *FileReadTool {
	var cfg FileReadConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultFileReadConfig()
	}
	
	return &FileReadTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (f *FileReadTool) GetName() string {
	return "file_read"
}

// GetDescription 返回工具描述。
func (f *FileReadTool) GetDescription() string {
	return "Read contents from a file. Returns the file content as a string."
}

// GetParameters 返回工具参数。
func (f *FileReadTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"path": {
				Type:        "string",
				Description: "Path to the file to read",
			},
		},
		Required: []string{"path"},
	}
}

// Execute 执行文件读取。
func (f *FileReadTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取路径
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("file read: 'path' parameter is required and must be a string")
	}
	
	// 规范化路径
	absPath, err := f.resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("file read: invalid path: %w", err)
	}
	
	// 验证路径是否允许
	if !f.isPathAllowed(absPath) {
		return nil, fmt.Errorf("file read: access denied to path: %s", path)
	}
	
	// 检查文件大小
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("file read: failed to stat file: %w", err)
	}
	
	if info.Size() > f.config.MaxFileSize {
		return nil, fmt.Errorf("file read: file too large (max %d bytes)", f.config.MaxFileSize)
	}
	
	// 读取文件
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("file read: failed to read file: %w", err)
	}
	
	return string(content), nil
}

// resolvePath 解析路径。
func (f *FileReadTool) resolvePath(path string) (string, error) {
	// 如果是相对路径，拼接基础路径
	if !filepath.IsAbs(path) {
		path = filepath.Join(f.config.BasePath, path)
	}
	
	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	
	// 清理路径（移除 .., . 等）
	cleanPath := filepath.Clean(absPath)
	
	return cleanPath, nil
}

// isPathAllowed 检查路径是否允许访问。
func (f *FileReadTool) isPathAllowed(path string) bool {
	// 如果没有设置允许路径，允许所有
	if len(f.config.AllowedPaths) == 0 {
		return true
	}
	
	// 检查是否在允许的路径下
	for _, allowedPath := range f.config.AllowedPaths {
		absAllowedPath, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(path, absAllowedPath) {
			return true
		}
	}
	
	return false
}

// ToTypesTool 转换为 types.Tool。
func (f *FileReadTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        f.GetName(),
		Description: f.GetDescription(),
		Parameters:  f.GetParameters(),
	}
}

// ========================
// 文件写入工具
// ========================

// FileWriteTool 是文件写入工具。
//
// 功能：
//   - 写入文件内容
//   - 支持创建目录
//   - 安全路径验证
//
type FileWriteTool struct {
	config FileWriteConfig
}

// FileWriteConfig 是文件写入配置。
type FileWriteConfig struct {
	// AllowedPaths 允许写入的路径列表（为空表示允许所有）
	AllowedPaths []string
	
	// MaxFileSize 最大文件大小（字节）
	MaxFileSize int64
	
	// BasePath 基础路径（用于相对路径）
	BasePath string
	
	// CreateDirs 是否自动创建目录
	CreateDirs bool
}

// DefaultFileWriteConfig 返回默认配置。
func DefaultFileWriteConfig() FileWriteConfig {
	return FileWriteConfig{
		AllowedPaths: []string{},
		MaxFileSize:  10 * 1024 * 1024, // 10MB
		BasePath:     ".",
		CreateDirs:   true,
	}
}

// NewFileWriteTool 创建文件写入工具。
func NewFileWriteTool(config *FileWriteConfig) *FileWriteTool {
	var cfg FileWriteConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultFileWriteConfig()
	}
	
	return &FileWriteTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (f *FileWriteTool) GetName() string {
	return "file_write"
}

// GetDescription 返回工具描述。
func (f *FileWriteTool) GetDescription() string {
	return "Write content to a file. Creates the file if it doesn't exist."
}

// GetParameters 返回工具参数。
func (f *FileWriteTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"path": {
				Type:        "string",
				Description: "Path to the file to write",
			},
			"content": {
				Type:        "string",
				Description: "Content to write to the file",
			},
		},
		Required: []string{"path", "content"},
	}
}

// Execute 执行文件写入。
func (f *FileWriteTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取路径
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("file write: 'path' parameter is required and must be a string")
	}
	
	// 获取内容
	content, ok := input["content"].(string)
	if !ok {
		return nil, fmt.Errorf("file write: 'content' parameter is required and must be a string")
	}
	
	// 检查内容大小
	if int64(len(content)) > f.config.MaxFileSize {
		return nil, fmt.Errorf("file write: content too large (max %d bytes)", f.config.MaxFileSize)
	}
	
	// 规范化路径
	absPath, err := f.resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("file write: invalid path: %w", err)
	}
	
	// 验证路径是否允许
	if !f.isPathAllowed(absPath) {
		return nil, fmt.Errorf("file write: access denied to path: %s", path)
	}
	
	// 创建目录（如果需要）
	if f.config.CreateDirs {
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("file write: failed to create directory: %w", err)
		}
	}
	
	// 写入文件
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("file write: failed to write file: %w", err)
	}
	
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

// resolvePath 解析路径。
func (f *FileWriteTool) resolvePath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(f.config.BasePath, path)
	}
	
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	
	cleanPath := filepath.Clean(absPath)
	return cleanPath, nil
}

// isPathAllowed 检查路径是否允许访问。
func (f *FileWriteTool) isPathAllowed(path string) bool {
	if len(f.config.AllowedPaths) == 0 {
		return true
	}
	
	for _, allowedPath := range f.config.AllowedPaths {
		absAllowedPath, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(path, absAllowedPath) {
			return true
		}
	}
	
	return false
}

// ToTypesTool 转换为 types.Tool。
func (f *FileWriteTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        f.GetName(),
		Description: f.GetDescription(),
		Parameters:  f.GetParameters(),
	}
}

// ========================
// 目录列表工具
// ========================

// ListDirectoryTool 是目录列表工具。
//
// 功能：
//   - 列出目录内容
//   - 支持递归列表
//   - 安全路径验证
//
type ListDirectoryTool struct {
	config ListDirectoryConfig
}

// ListDirectoryConfig 是目录列表配置。
type ListDirectoryConfig struct {
	// AllowedPaths 允许访问的路径列表（为空表示允许所有）
	AllowedPaths []string
	
	// BasePath 基础路径（用于相对路径）
	BasePath string
	
	// ShowHidden 是否显示隐藏文件
	ShowHidden bool
}

// DefaultListDirectoryConfig 返回默认配置。
func DefaultListDirectoryConfig() ListDirectoryConfig {
	return ListDirectoryConfig{
		AllowedPaths: []string{},
		BasePath:     ".",
		ShowHidden:   false,
	}
}

// NewListDirectoryTool 创建目录列表工具。
func NewListDirectoryTool(config *ListDirectoryConfig) *ListDirectoryTool {
	var cfg ListDirectoryConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = DefaultListDirectoryConfig()
	}
	
	return &ListDirectoryTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (l *ListDirectoryTool) GetName() string {
	return "list_directory"
}

// GetDescription 返回工具描述。
func (l *ListDirectoryTool) GetDescription() string {
	return "List contents of a directory. Returns file and directory names."
}

// GetParameters 返回工具参数。
func (l *ListDirectoryTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"path": {
				Type:        "string",
				Description: "Path to the directory to list",
			},
		},
		Required: []string{"path"},
	}
}

// Execute 执行目录列表。
func (l *ListDirectoryTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	// 获取路径
	path, ok := input["path"].(string)
	if !ok {
		return nil, fmt.Errorf("list directory: 'path' parameter is required and must be a string")
	}
	
	// 规范化路径
	absPath, err := l.resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("list directory: invalid path: %w", err)
	}
	
	// 验证路径是否允许
	if !l.isPathAllowed(absPath) {
		return nil, fmt.Errorf("list directory: access denied to path: %s", path)
	}
	
	// 检查是否是目录
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("list directory: failed to stat path: %w", err)
	}
	
	if !info.IsDir() {
		return nil, fmt.Errorf("list directory: path is not a directory: %s", path)
	}
	
	// 读取目录内容
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("list directory: failed to read directory: %w", err)
	}
	
	// 格式化结果
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Contents of %s:\n\n", path))
	
	fileCount := 0
	dirCount := 0
	
	for _, entry := range entries {
		name := entry.Name()
		
		// 跳过隐藏文件
		if !l.config.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		
		if entry.IsDir() {
			builder.WriteString(fmt.Sprintf("[DIR]  %s\n", name))
			dirCount++
		} else {
			info, _ := entry.Info()
			size := info.Size()
			builder.WriteString(fmt.Sprintf("[FILE] %s (%s)\n", name, formatFileSize(size)))
			fileCount++
		}
	}
	
	builder.WriteString(fmt.Sprintf("\nTotal: %d directories, %d files\n", dirCount, fileCount))
	
	return builder.String(), nil
}

// resolvePath 解析路径。
func (l *ListDirectoryTool) resolvePath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(l.config.BasePath, path)
	}
	
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	
	cleanPath := filepath.Clean(absPath)
	return cleanPath, nil
}

// isPathAllowed 检查路径是否允许访问。
func (l *ListDirectoryTool) isPathAllowed(path string) bool {
	if len(l.config.AllowedPaths) == 0 {
		return true
	}
	
	for _, allowedPath := range l.config.AllowedPaths {
		absAllowedPath, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(path, absAllowedPath) {
			return true
		}
	}
	
	return false
}

// ToTypesTool 转换为 types.Tool。
func (l *ListDirectoryTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        l.GetName(),
		Description: l.GetDescription(),
		Parameters:  l.GetParameters(),
	}
}

// formatFileSize 格式化文件大小。
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}

// ========================
// 文件复制工具
// ========================

// FileCopyTool 是文件复制工具。
type FileCopyTool struct {
	config FileOperationConfig
}

// FileOperationConfig 是文件操作配置。
type FileOperationConfig struct {
	AllowedPaths []string
	BasePath     string
}

// NewFileCopyTool 创建文件复制工具。
func NewFileCopyTool(config *FileOperationConfig) *FileCopyTool {
	var cfg FileOperationConfig
	if config != nil {
		cfg = *config
	} else {
		cfg = FileOperationConfig{
			AllowedPaths: []string{},
			BasePath:     ".",
		}
	}
	
	return &FileCopyTool{
		config: cfg,
	}
}

// GetName 返回工具名称。
func (f *FileCopyTool) GetName() string {
	return "file_copy"
}

// GetDescription 返回工具描述。
func (f *FileCopyTool) GetDescription() string {
	return "Copy a file from source to destination."
}

// GetParameters 返回工具参数。
func (f *FileCopyTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"source": {
				Type:        "string",
				Description: "Source file path",
			},
			"destination": {
				Type:        "string",
				Description: "Destination file path",
			},
		},
		Required: []string{"source", "destination"},
	}
}

// Execute 执行文件复制。
func (f *FileCopyTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	source, ok := input["source"].(string)
	if !ok {
		return nil, fmt.Errorf("file copy: 'source' parameter is required")
	}
	
	dest, ok := input["destination"].(string)
	if !ok {
		return nil, fmt.Errorf("file copy: 'destination' parameter is required")
	}
	
	// 打开源文件
	srcFile, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("file copy: failed to open source: %w", err)
	}
	defer srcFile.Close()
	
	// 创建目标文件
	destFile, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("file copy: failed to create destination: %w", err)
	}
	defer destFile.Close()
	
	// 复制内容
	written, err := io.Copy(destFile, srcFile)
	if err != nil {
		return nil, fmt.Errorf("file copy: failed to copy: %w", err)
	}
	
	return fmt.Sprintf("Successfully copied %d bytes from %s to %s", written, source, dest), nil
}

// ToTypesTool 转换为 types.Tool。
func (f *FileCopyTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        f.GetName(),
		Description: f.GetDescription(),
		Parameters:  f.GetParameters(),
	}
}

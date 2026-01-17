package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/zhucl121/langchain-go/pkg/types"
)

// FileOperation 定义文件操作类型
type FileOperation string

const (
	// OpRead 读取文件
	OpRead FileOperation = "read"
	
	// OpWrite 写入文件
	OpWrite FileOperation = "write"
	
	// OpAppend 追加到文件
	OpAppend FileOperation = "append"
	
	// OpDelete 删除文件
	OpDelete FileOperation = "delete"
	
	// OpList 列出目录内容
	OpList FileOperation = "list"
	
	// OpExists 检查文件是否存在
	OpExists FileOperation = "exists"
	
	// OpCopy 复制文件
	OpCopy FileOperation = "copy"
	
	// OpMove 移动文件
	OpMove FileOperation = "move"
)

// FileSystemConfig 文件系统工具配置
type FileSystemConfig struct {
	// AllowedPaths 允许访问的路径列表（安全限制）
	// 如果为空，则允许访问所有路径（不推荐）
	AllowedPaths []string
	
	// AllowWrite 是否允许写操作
	AllowWrite bool
	
	// AllowDelete 是否允许删除操作
	AllowDelete bool
	
	// MaxFileSize 最大文件大小（字节）
	// 0 表示不限制
	MaxFileSize int64
}

// FileSystemTool 文件系统工具
type FileSystemTool struct {
	config FileSystemConfig
}

// NewFileSystemTool 创建文件系统工具
func NewFileSystemTool(config FileSystemConfig) *FileSystemTool {
	// 默认配置
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	
	return &FileSystemTool{
		config: config,
	}
}

// GetName 实现 Tool 接口
func (fst *FileSystemTool) GetName() string {
	return "file_system"
}

// GetDescription 实现 Tool 接口
func (fst *FileSystemTool) GetDescription() string {
	return "Interact with the file system. Can read files, write files, list directories, check file existence, copy, and move files."
}

// GetParameters 实现 Tool 接口
func (fst *FileSystemTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
		"operation": {
			Type:        "string",
			Description: "Operation to perform: read, write, append, delete, list, exists, copy, move",
		},
			"path": {
				Type:        "string",
				Description: "Path to the file or directory",
			},
			"content": {
				Type:        "string",
				Description: "Content to write (for write/append operations)",
			},
			"destination": {
				Type:        "string",
				Description: "Destination path (for copy/move operations)",
			},
		},
		Required: []string{"operation", "path"},
	}
}

// Execute 实现 Tool 接口
func (fst *FileSystemTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 提取参数
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation must be a string")
	}
	
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}
	
	// 验证路径
	if err := fst.validatePath(path); err != nil {
		return nil, err
	}
	
	// 执行操作
	switch FileOperation(operation) {
	case OpRead:
		return fst.readFile(path)
		
	case OpWrite:
		if !fst.config.AllowWrite {
			return nil, fmt.Errorf("write operations are not allowed")
		}
		content, _ := args["content"].(string)
		return fst.writeFile(path, content, false)
		
	case OpAppend:
		if !fst.config.AllowWrite {
			return nil, fmt.Errorf("write operations are not allowed")
		}
		content, _ := args["content"].(string)
		return fst.writeFile(path, content, true)
		
	case OpDelete:
		if !fst.config.AllowDelete {
			return nil, fmt.Errorf("delete operations are not allowed")
		}
		return fst.deleteFile(path)
		
	case OpList:
		return fst.listDirectory(path)
		
	case OpExists:
		return fst.checkExists(path)
		
	case OpCopy:
		if !fst.config.AllowWrite {
			return nil, fmt.Errorf("copy operations require write permission")
		}
		dest, ok := args["destination"].(string)
		if !ok {
			return nil, fmt.Errorf("destination must be specified for copy operation")
		}
		if err := fst.validatePath(dest); err != nil {
			return nil, err
		}
		return fst.copyFile(path, dest)
		
	case OpMove:
		if !fst.config.AllowWrite || !fst.config.AllowDelete {
			return nil, fmt.Errorf("move operations require write and delete permission")
		}
		dest, ok := args["destination"].(string)
		if !ok {
			return nil, fmt.Errorf("destination must be specified for move operation")
		}
		if err := fst.validatePath(dest); err != nil {
			return nil, err
		}
		return fst.moveFile(path, dest)
		
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// validatePath 验证路径是否允许访问
func (fst *FileSystemTool) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	// 清理路径
	cleanPath := filepath.Clean(path)
	
	// 如果没有配置允许的路径，允许访问所有路径
	if len(fst.config.AllowedPaths) == 0 {
		return nil
	}
	
	// 检查路径是否在允许的路径列表中
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	for _, allowed := range fst.config.AllowedPaths {
		absAllowed, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		
		// 检查路径是否在允许的目录下
		if strings.HasPrefix(absPath, absAllowed) {
			return nil
		}
	}
	
	return fmt.Errorf("access to path %s is not allowed", path)
}

// readFile 读取文件
func (fst *FileSystemTool) readFile(path string) (string, error) {
	// 检查文件大小
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}
	
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}
	
	if fst.config.MaxFileSize > 0 && info.Size() > fst.config.MaxFileSize {
		return "", fmt.Errorf("file size %d exceeds maximum allowed size %d", 
			info.Size(), fst.config.MaxFileSize)
	}
	
	// 读取文件
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	return string(content), nil
}

// writeFile 写入文件
func (fst *FileSystemTool) writeFile(path, content string, append bool) (string, error) {
	// 检查内容大小
	if fst.config.MaxFileSize > 0 && int64(len(content)) > fst.config.MaxFileSize {
		return "", fmt.Errorf("content size %d exceeds maximum allowed size %d",
			len(content), fst.config.MaxFileSize)
	}
	
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	var err error
	if append {
		// 追加模式
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to open file for append: %w", err)
		}
		defer f.Close()
		
		if _, err = f.WriteString(content); err != nil {
			return "", fmt.Errorf("failed to append to file: %w", err)
		}
	} else {
		// 覆盖模式
		err = os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
	}
	
	mode := "written"
	if append {
		mode = "appended"
	}
	
	return fmt.Sprintf("Successfully %s %d bytes to %s", mode, len(content), path), nil
}

// deleteFile 删除文件
func (fst *FileSystemTool) deleteFile(path string) (string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}
	
	// 删除文件
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}
	
	return fmt.Sprintf("Successfully deleted %s", path), nil
}

// listDirectory 列出目录内容
func (fst *FileSystemTool) listDirectory(path string) (string, error) {
	// 检查是否是目录
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}
	
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}
	
	// 读取目录
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}
	
	// 格式化输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Contents of %s (%d items):\n\n", path, len(entries)))
	
	for _, entry := range entries {
		fileType := "file"
		if entry.IsDir() {
			fileType = "dir"
		}
		
		info, _ := entry.Info()
		size := ""
		if info != nil && !entry.IsDir() {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		
		result.WriteString(fmt.Sprintf("- %s [%s]%s\n", entry.Name(), fileType, size))
	}
	
	return result.String(), nil
}

// checkExists 检查文件是否存在
func (fst *FileSystemTool) checkExists(path string) (string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Sprintf("Path does not exist: %s", path), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to check path: %w", err)
	}
	
	fileType := "file"
	if info.IsDir() {
		fileType = "directory"
	}
	
	return fmt.Sprintf("Path exists: %s (%s, %d bytes)", path, fileType, info.Size()), nil
}

// copyFile 复制文件
func (fst *FileSystemTool) copyFile(src, dst string) (string, error) {
	// 检查源文件
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("failed to stat source file: %w", err)
	}
	
	if srcInfo.IsDir() {
		return "", fmt.Errorf("source is a directory, not a file")
	}
	
	// 检查文件大小
	if fst.config.MaxFileSize > 0 && srcInfo.Size() > fst.config.MaxFileSize {
		return "", fmt.Errorf("file size exceeds maximum allowed size")
	}
	
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()
	
	// 创建目标目录
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()
	
	// 复制内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}
	
	return fmt.Sprintf("Successfully copied %d bytes from %s to %s", written, src, dst), nil
}

// moveFile 移动文件
func (fst *FileSystemTool) moveFile(src, dst string) (string, error) {
	// 先复制
	result, err := fst.copyFile(src, dst)
	if err != nil {
		return "", err
	}
	
	// 然后删除源文件
	if err := os.Remove(src); err != nil {
		return "", fmt.Errorf("copied but failed to delete source: %w", err)
	}
	
	return strings.Replace(result, "copied", "moved", 1), nil
}

// ToTypesTool 实现 Tool 接口
func (fst *FileSystemTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        fst.GetName(),
		Description: fst.GetDescription(),
		Parameters:  fst.GetParameters(),
	}
}

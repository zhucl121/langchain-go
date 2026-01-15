package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestFileSystemTool 测试文件系统工具
func TestFileSystemTool(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()
	
	// 创建工具（允许读写）
	tool := NewFileSystemTool(FileSystemConfig{
		AllowedPaths: []string{tempDir},
		AllowWrite:   true,
		AllowDelete:  true,
		MaxFileSize:  1024 * 1024, // 1MB
	})
	
	ctx := context.Background()
	
	t.Run("write file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "write",
			"path":      testFile,
			"content":   "Hello, World!",
		})
		
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		
		resultStr, ok := result.(string)
		if !ok {
			t.Fatal("Result is not a string")
		}
		
		if resultStr == "" {
			t.Error("Result string is empty")
		}
	})
	
	t.Run("read file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "read",
			"path":      testFile,
		})
		
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		
		content, ok := result.(string)
		if !ok {
			t.Fatal("Result is not a string")
		}
		
		if content != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got '%s'", content)
		}
	})
	
	t.Run("append to file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "append",
			"path":      testFile,
			"content":   "\nAppended line",
		})
		
		if err != nil {
			t.Fatalf("Append failed: %v", err)
		}
		
		// 读取验证
		result, _ := tool.Execute(ctx, map[string]any{
			"operation": "read",
			"path":      testFile,
		})
		
		content := result.(string)
		expected := "Hello, World!\nAppended line"
		if content != expected {
			t.Errorf("Expected '%s', got '%s'", expected, content)
		}
	})
	
	t.Run("list directory", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "list",
			"path":      tempDir,
		})
		
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		
		content, ok := result.(string)
		if !ok {
			t.Fatal("Result is not a string")
		}
		
		if content == "" {
			t.Error("List result is empty")
		}
	})
	
	t.Run("check exists", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "exists",
			"path":      testFile,
		})
		
		if err != nil {
			t.Fatalf("Exists check failed: %v", err)
		}
		
		resultStr, _ := result.(string)
		if resultStr == "" {
			t.Error("Exists result is empty")
		}
	})
	
	t.Run("copy file", func(t *testing.T) {
		srcFile := filepath.Join(tempDir, "test.txt")
		dstFile := filepath.Join(tempDir, "test_copy.txt")
		
		result, err := tool.Execute(ctx, map[string]any{
			"operation":   "copy",
			"path":        srcFile,
			"destination": dstFile,
		})
		
		if err != nil {
			t.Fatalf("Copy failed: %v", err)
		}
		
		resultStr, _ := result.(string)
		if resultStr == "" {
			t.Error("Copy result is empty")
		}
		
		// 验证目标文件存在
		if _, err := os.Stat(dstFile); os.IsNotExist(err) {
			t.Error("Destination file was not created")
		}
	})
	
	t.Run("move file", func(t *testing.T) {
		srcFile := filepath.Join(tempDir, "test_copy.txt")
		dstFile := filepath.Join(tempDir, "test_moved.txt")
		
		_, err := tool.Execute(ctx, map[string]any{
			"operation":   "move",
			"path":        srcFile,
			"destination": dstFile,
		})
		
		if err != nil {
			t.Fatalf("Move failed: %v", err)
		}
		
		// 验证源文件不存在
		if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
			t.Error("Source file still exists after move")
		}
		
		// 验证目标文件存在
		if _, err := os.Stat(dstFile); os.IsNotExist(err) {
			t.Error("Destination file was not created")
		}
	})
	
	t.Run("delete file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_moved.txt")
		
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "delete",
			"path":      testFile,
		})
		
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		
		resultStr, _ := result.(string)
		if resultStr == "" {
			t.Error("Delete result is empty")
		}
		
		// 验证文件被删除
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("File still exists after delete")
		}
	})
}

// TestFileSystemSecurity 测试安全限制
func TestFileSystemSecurity(t *testing.T) {
	tempDir := t.TempDir()
	
	t.Run("path validation", func(t *testing.T) {
		tool := NewFileSystemTool(FileSystemConfig{
			AllowedPaths: []string{tempDir},
			AllowWrite:   true,
		})
		
		ctx := context.Background()
		
		// 尝试访问不允许的路径
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "read",
			"path":      "/etc/passwd",
		})
		
		if err == nil {
			t.Error("Expected error for unauthorized path access")
		}
	})
	
	t.Run("write permission", func(t *testing.T) {
		tool := NewFileSystemTool(FileSystemConfig{
			AllowedPaths: []string{tempDir},
			AllowWrite:   false, // 禁止写入
		})
		
		ctx := context.Background()
		testFile := filepath.Join(tempDir, "readonly.txt")
		
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "write",
			"path":      testFile,
			"content":   "test",
		})
		
		if err == nil {
			t.Error("Expected error for write operation when AllowWrite is false")
		}
	})
	
	t.Run("delete permission", func(t *testing.T) {
		// 先创建文件
		testFile := filepath.Join(tempDir, "todelete.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		
		tool := NewFileSystemTool(FileSystemConfig{
			AllowedPaths: []string{tempDir},
			AllowDelete:  false, // 禁止删除
		})
		
		ctx := context.Background()
		
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "delete",
			"path":      testFile,
		})
		
		if err == nil {
			t.Error("Expected error for delete operation when AllowDelete is false")
		}
	})
	
	t.Run("file size limit", func(t *testing.T) {
		tool := NewFileSystemTool(FileSystemConfig{
			AllowedPaths: []string{tempDir},
			AllowWrite:   true,
			MaxFileSize:  10, // 只允许 10 字节
		})
		
		ctx := context.Background()
		testFile := filepath.Join(tempDir, "large.txt")
		
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "write",
			"path":      testFile,
			"content":   "This content is too large",
		})
		
		if err == nil {
			t.Error("Expected error for content exceeding max file size")
		}
	})
}

// TestFileSystemErrors 测试错误处理
func TestFileSystemErrors(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewFileSystemTool(FileSystemConfig{
		AllowedPaths: []string{tempDir},
		AllowWrite:   true,
		AllowDelete:  true,
	})
	
	ctx := context.Background()
	
	t.Run("invalid operation", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "invalid_op",
			"path":      tempDir,
		})
		
		if err == nil {
			t.Error("Expected error for invalid operation")
		}
	})
	
	t.Run("missing path", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "read",
		})
		
		if err == nil {
			t.Error("Expected error for missing path")
		}
	})
	
	t.Run("read nonexistent file", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "read",
			"path":      filepath.Join(tempDir, "nonexistent.txt"),
		})
		
		if err == nil {
			t.Error("Expected error for reading nonexistent file")
		}
	})
	
	t.Run("list nonexistent directory", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "list",
			"path":      filepath.Join(tempDir, "nonexistent_dir"),
		})
		
		if err == nil {
			t.Error("Expected error for listing nonexistent directory")
		}
	})
	
	t.Run("copy without destination", func(t *testing.T) {
		// 先创建源文件
		srcFile := filepath.Join(tempDir, "source.txt")
		os.WriteFile(srcFile, []byte("test"), 0644)
		
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "copy",
			"path":      srcFile,
		})
		
		if err == nil {
			t.Error("Expected error for copy without destination")
		}
	})
}

// TestToolInterface 测试 Tool 接口实现
func TestToolInterface(t *testing.T) {
	tool := NewFileSystemTool(FileSystemConfig{
		AllowWrite:  true,
		AllowDelete: true,
	})
	
	t.Run("get name", func(t *testing.T) {
		name := tool.GetName()
		if name != "file_system" {
			t.Errorf("Expected name 'file_system', got '%s'", name)
		}
	})
	
	t.Run("get description", func(t *testing.T) {
		desc := tool.GetDescription()
		if desc == "" {
			t.Error("Description should not be empty")
		}
	})
	
	t.Run("get parameters", func(t *testing.T) {
		params := tool.GetParameters()
		if params.Type != "object" {
			t.Error("Parameters type should be 'object'")
		}
		
		if len(params.Properties) == 0 {
			t.Error("Parameters should have properties")
		}
		
		if len(params.Required) == 0 {
			t.Error("Parameters should have required fields")
		}
	})
	
	t.Run("to types tool", func(t *testing.T) {
		typesTool := tool.ToTypesTool()
		if typesTool.Name == "" {
			t.Error("TypesTool name should not be empty")
		}
		if typesTool.Description == "" {
			t.Error("TypesTool description should not be empty")
		}
	})
}

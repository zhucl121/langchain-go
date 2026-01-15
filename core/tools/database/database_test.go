package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) (*DatabaseTool, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	// 创建数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// 创建测试表
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	
	// 插入测试数据
	_, err = db.Exec(`
		INSERT INTO users (name, email, age) VALUES
		('Alice', 'alice@example.com', 30),
		('Bob', 'bob@example.com', 25),
		('Charlie', 'charlie@example.com', 35)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}
	
	db.Close()
	
	// 创建工具
	tool, err := NewDatabaseTool(DatabaseConfig{
		Type:             DBTypeSQLite,
		ConnectionString: dbPath,
		ReadOnly:         false,
		MaxRows:          100,
	})
	if err != nil {
		t.Fatalf("Failed to create database tool: %v", err)
	}
	
	// 清理函数
	cleanup := func() {
		tool.Close()
		os.Remove(dbPath)
	}
	
	return tool, cleanup
}

// TestDatabaseQuery 测试查询操作
func TestDatabaseQuery(t *testing.T) {
	tool, cleanup := setupTestDB(t)
	defer cleanup()
	
	ctx := context.Background()
	
	t.Run("select all", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "query",
			"sql":       "SELECT * FROM users",
		})
		
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		
		resultStr, ok := result.(string)
		if !ok {
			t.Fatal("Result is not a string")
		}
		
		if resultStr == "" {
			t.Error("Query result is empty")
		}
		
		// 检查是否包含测试数据
		if !contains(resultStr, "Alice") {
			t.Error("Result should contain 'Alice'")
		}
	})
	
	t.Run("select with where", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "query",
			"sql":       "SELECT name, email FROM users WHERE age > 28",
		})
		
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		
		resultStr := result.(string)
		if !contains(resultStr, "Alice") || !contains(resultStr, "Charlie") {
			t.Error("Result should contain Alice and Charlie")
		}
		if contains(resultStr, "Bob") {
			t.Error("Result should not contain Bob")
		}
	})
	
	t.Run("select count", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "query",
			"sql":       "SELECT COUNT(*) as count FROM users",
		})
		
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		
		resultStr := result.(string)
		if !contains(resultStr, "3") {
			t.Error("Count should be 3")
		}
	})
}

// TestDatabaseExecute 测试执行操作
func TestDatabaseExecute(t *testing.T) {
	tool, cleanup := setupTestDB(t)
	defer cleanup()
	
	ctx := context.Background()
	
	t.Run("insert", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "execute",
			"sql":       "INSERT INTO users (name, email, age) VALUES ('David', 'david@example.com', 28)",
		})
		
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		
		resultStr := result.(string)
		if !contains(resultStr, "1") {
			t.Error("Should affect 1 row")
		}
		
		// 验证插入
		queryResult, _ := tool.Execute(ctx, map[string]any{
			"operation": "query",
			"sql":       "SELECT * FROM users WHERE name = 'David'",
		})
		
		queryStr := queryResult.(string)
		if !contains(queryStr, "David") {
			t.Error("David should be in the database")
		}
	})
	
	t.Run("update", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "execute",
			"sql":       "UPDATE users SET age = 31 WHERE name = 'Alice'",
		})
		
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		
		resultStr := result.(string)
		if !contains(resultStr, "1") {
			t.Error("Should affect 1 row")
		}
	})
	
	t.Run("delete", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "execute",
			"sql":       "DELETE FROM users WHERE name = 'David'",
		})
		
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		
		resultStr := result.(string)
		if !contains(resultStr, "1") {
			t.Error("Should affect 1 row")
		}
	})
}

// TestDatabaseMetadata 测试元数据操作
func TestDatabaseMetadata(t *testing.T) {
	tool, cleanup := setupTestDB(t)
	defer cleanup()
	
	ctx := context.Background()
	
	t.Run("list tables", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "list_tables",
		})
		
		if err != nil {
			t.Fatalf("List tables failed: %v", err)
		}
		
		resultStr := result.(string)
		if !contains(resultStr, "users") {
			t.Error("Should list 'users' table")
		}
	})
	
	t.Run("describe table", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]any{
			"operation": "describe_table",
			"table":     "users",
		})
		
		if err != nil {
			t.Fatalf("Describe table failed: %v", err)
		}
		
		resultStr := result.(string)
		if !contains(resultStr, "name") || !contains(resultStr, "email") {
			t.Error("Should show table columns")
		}
	})
}

// TestDatabaseSecurity 测试安全限制
func TestDatabaseSecurity(t *testing.T) {
	tool, cleanup := setupTestDB(t)
	defer cleanup()
	
	ctx := context.Background()
	
	t.Run("prevent non-select in query", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "query",
			"sql":       "INSERT INTO users (name, email, age) VALUES ('Evil', 'evil@example.com', 99)",
		})
		
		if err == nil {
			t.Error("Should not allow INSERT in query operation")
		}
	})
	
	t.Run("read-only mode", func(t *testing.T) {
		// 创建只读工具
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "readonly.db")
		
		// 复制数据库
		db, _ := sql.Open("sqlite3", dbPath)
		db.Exec(`CREATE TABLE test (id INTEGER)`)
		db.Close()
		
		readOnlyTool, _ := NewDatabaseTool(DatabaseConfig{
			Type:             DBTypeSQLite,
			ConnectionString: dbPath,
			ReadOnly:         true,
		})
		defer readOnlyTool.Close()
		
		_, err := readOnlyTool.Execute(ctx, map[string]any{
			"operation": "execute",
			"sql":       "INSERT INTO test (id) VALUES (1)",
		})
		
		if err == nil {
			t.Error("Should not allow execute in read-only mode")
		}
	})
}

// TestDatabaseConfig 测试配置
func TestDatabaseConfig(t *testing.T) {
	t.Run("empty connection string", func(t *testing.T) {
		_, err := NewDatabaseTool(DatabaseConfig{
			Type: DBTypeSQLite,
		})
		
		if err == nil {
			t.Error("Should fail with empty connection string")
		}
	})
	
	t.Run("invalid connection string", func(t *testing.T) {
		_, err := NewDatabaseTool(DatabaseConfig{
			Type:             DBTypeSQLite,
			ConnectionString: "/nonexistent/path/db.sqlite",
		})
		
		if err == nil {
			t.Error("Should fail with invalid connection string")
		}
	})
	
	t.Run("default max rows", func(t *testing.T) {
		tool, cleanup := setupTestDB(t)
		defer cleanup()
		
		if tool.config.MaxRows != 100 {
			t.Errorf("Expected default MaxRows=100, got %d", tool.config.MaxRows)
		}
	})
}

// TestToolInterface 测试 Tool 接口
func TestToolInterface(t *testing.T) {
	tool, cleanup := setupTestDB(t)
	defer cleanup()
	
	t.Run("get name", func(t *testing.T) {
		name := tool.GetName()
		if name != "sqlite_database" {
			t.Errorf("Expected name 'sqlite_database', got '%s'", name)
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
	})
	
	t.Run("to types tool", func(t *testing.T) {
		typesTool := tool.ToTypesTool()
		if typesTool.Name == "" {
			t.Error("TypesTool name should not be empty")
		}
	})
}

// TestDatabaseErrors 测试错误处理
func TestDatabaseErrors(t *testing.T) {
	tool, cleanup := setupTestDB(t)
	defer cleanup()
	
	ctx := context.Background()
	
	t.Run("invalid operation", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "invalid_op",
		})
		
		if err == nil {
			t.Error("Should fail with invalid operation")
		}
	})
	
	t.Run("missing sql for query", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "query",
		})
		
		if err == nil {
			t.Error("Should fail with missing SQL")
		}
	})
	
	t.Run("invalid sql", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "query",
			"sql":       "INVALID SQL QUERY",
		})
		
		if err == nil {
			t.Error("Should fail with invalid SQL")
		}
	})
	
	t.Run("missing table for describe", func(t *testing.T) {
		_, err := tool.Execute(ctx, map[string]any{
			"operation": "describe_table",
		})
		
		if err == nil {
			t.Error("Should fail with missing table name")
		}
	})
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

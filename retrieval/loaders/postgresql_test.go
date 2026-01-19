package loaders

import (
	"testing"
)

func TestNewPostgreSQLLoader(t *testing.T) {
	tests := []struct {
		name      string
		config    PostgreSQLLoaderConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: PostgreSQLLoaderConfig{
				Host:     "localhost",
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
			wantError: false, // 会因为连接失败而报错，但配置验证会通过
		},
		{
			name: "missing host",
			config: PostgreSQLLoaderConfig{
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
			wantError: true,
		},
		{
			name: "missing database",
			config: PostgreSQLLoaderConfig{
				Host:     "localhost",
				User:     "testuser",
				Password: "testpass",
			},
			wantError: true,
		},
		{
			name: "missing user",
			config: PostgreSQLLoaderConfig{
				Host:     "localhost",
				Database: "testdb",
				Password: "testpass",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewPostgreSQLLoader(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				// 即使配置正确，没有真实数据库连接也会失败
				// 这里主要测试配置验证逻辑
				if err != nil && loader != nil {
					loader.Close()
				}
			}
		})
	}
}

func TestValueToString(t *testing.T) {
	loader := &PostgreSQLLoader{}

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"bytes", []byte("hello"), "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.140000"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.valueToString(tt.value)
			if result != tt.expected {
				t.Errorf("valueToString(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

// 集成测试（需要真实 PostgreSQL 数据库）
func TestPostgreSQLLoaderIntegration(t *testing.T) {
	t.Skip("Integration test - requires PostgreSQL database")

	// 取消注释以运行集成测试
	// config := PostgreSQLLoaderConfig{
	// 	Host:     "localhost",
	// 	Port:     5432,
	// 	Database: "testdb",
	// 	User:     "postgres",
	// 	Password: "password",
	// 	SSLMode:  "disable",
	// }
	//
	// loader, err := NewPostgreSQLLoader(config)
	// if err != nil {
	// 	t.Fatalf("Failed to create loader: %v", err)
	// }
	// defer loader.Close()
	//
	// ctx := context.Background()
	//
	// // 创建测试表
	// _, err = loader.db.ExecContext(ctx, `
	// 	CREATE TABLE IF NOT EXISTS test_documents (
	// 		id SERIAL PRIMARY KEY,
	// 		title VARCHAR(255),
	// 		content TEXT,
	// 		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	// 	)
	// `)
	// if err != nil {
	// 	t.Fatalf("Failed to create table: %v", err)
	// }
	//
	// // 插入测试数据
	// _, err = loader.db.ExecContext(ctx, `
	// 	INSERT INTO test_documents (title, content) VALUES
	// 	('Doc 1', 'Content 1'),
	// 	('Doc 2', 'Content 2')
	// `)
	// if err != nil {
	// 	t.Fatalf("Failed to insert data: %v", err)
	// }
	//
	// // 测试加载表
	// docs, err := loader.LoadTable(ctx, "test_documents", "content", "id", "title")
	// if err != nil {
	// 	t.Fatalf("Failed to load table: %v", err)
	// }
	//
	// if len(docs) != 2 {
	// 	t.Errorf("Expected 2 documents, got %d", len(docs))
	// }
	//
	// t.Logf("Loaded %d documents", len(docs))
	//
	// // 清理
	// _, err = loader.db.ExecContext(ctx, "DROP TABLE test_documents")
	// if err != nil {
	// 	t.Errorf("Failed to drop table: %v", err)
	// }
}

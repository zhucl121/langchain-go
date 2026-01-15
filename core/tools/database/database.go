package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	
	"langchain-go/pkg/types"
)

// DatabaseType 定义数据库类型
type DatabaseType string

const (
	// DBTypeSQLite SQLite 数据库
	DBTypeSQLite DatabaseType = "sqlite"
	
	// DBTypePostgreSQL PostgreSQL 数据库
	DBTypePostgreSQL DatabaseType = "postgresql"
	
	// DBTypeMySQL MySQL 数据库
	DBTypeMySQL DatabaseType = "mysql"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	// Type 数据库类型
	Type DatabaseType
	
	// ConnectionString 连接字符串
	// SQLite: "file:test.db?mode=rwc"
	// PostgreSQL: "postgres://user:pass@localhost/dbname?sslmode=disable"
	// MySQL: "user:pass@tcp(localhost:3306)/dbname"
	ConnectionString string
	
	// ReadOnly 是否只读模式
	ReadOnly bool
	
	// AllowedTables 允许访问的表列表（安全限制）
	// 如果为空，则允许访问所有表
	AllowedTables []string
	
	// MaxRows 单次查询最大返回行数
	MaxRows int
}

// DatabaseTool 数据库工具
type DatabaseTool struct {
	config DatabaseConfig
	db     *sql.DB
}

// NewDatabaseTool 创建数据库工具
func NewDatabaseTool(config DatabaseConfig) (*DatabaseTool, error) {
	if config.ConnectionString == "" {
		return nil, fmt.Errorf("connection string is required")
	}
	
	// 默认配置
	if config.MaxRows == 0 {
		config.MaxRows = 100
	}
	
	// 根据数据库类型获取驱动名称
	driverName := getDriverName(config.Type)
	
	// 打开数据库连接
	db, err := sql.Open(driverName, config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return &DatabaseTool{
		config: config,
		db:     db,
	}, nil
}

// getDriverName 获取数据库驱动名称
func getDriverName(dbType DatabaseType) string {
	switch dbType {
	case DBTypeSQLite:
		return "sqlite3"
	case DBTypePostgreSQL:
		return "postgres"
	case DBTypeMySQL:
		return "mysql"
	default:
		return string(dbType)
	}
}

// GetName 实现 Tool 接口
func (dt *DatabaseTool) GetName() string {
	return fmt.Sprintf("%s_database", dt.config.Type)
}

// GetDescription 实现 Tool 接口
func (dt *DatabaseTool) GetDescription() string {
	mode := "read and write"
	if dt.config.ReadOnly {
		mode = "read-only"
	}
	return fmt.Sprintf("Query and interact with a %s database (%s mode). Can execute SELECT queries, INSERT, UPDATE, DELETE statements, and get table information.", dt.config.Type, mode)
}

// GetParameters 实现 Tool 接口
func (dt *DatabaseTool) GetParameters() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
		"operation": {
			Type:        "string",
			Description: "Operation to perform: query, execute, list_tables, describe_table",
		},
			"sql": {
				Type:        "string",
				Description: "SQL statement to execute (for query/execute operations)",
			},
			"table": {
				Type:        "string",
				Description: "Table name (for describe_table operation)",
			},
		},
		Required: []string{"operation"},
	}
}

// Execute 实现 Tool 接口
func (dt *DatabaseTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation must be a string")
	}
	
	switch operation {
	case "query":
		sqlStr, ok := args["sql"].(string)
		if !ok {
			return nil, fmt.Errorf("sql must be specified for query operation")
		}
		return dt.executeQuery(ctx, sqlStr)
		
	case "execute":
		if dt.config.ReadOnly {
			return nil, fmt.Errorf("write operations are not allowed in read-only mode")
		}
		sqlStr, ok := args["sql"].(string)
		if !ok {
			return nil, fmt.Errorf("sql must be specified for execute operation")
		}
		return dt.executeStatement(ctx, sqlStr)
		
	case "list_tables":
		return dt.listTables(ctx)
		
	case "describe_table":
		table, ok := args["table"].(string)
		if !ok {
			return nil, fmt.Errorf("table must be specified for describe_table operation")
		}
		return dt.describeTable(ctx, table)
		
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// executeQuery 执行查询
func (dt *DatabaseTool) executeQuery(ctx context.Context, sqlStr string) (string, error) {
	// 验证 SQL（简单检查，防止非 SELECT 语句）
	trimmedSQL := strings.TrimSpace(strings.ToUpper(sqlStr))
	if !strings.HasPrefix(trimmedSQL, "SELECT") && !strings.HasPrefix(trimmedSQL, "WITH") {
		return "", fmt.Errorf("only SELECT queries are allowed in query operation")
	}
	
	// 检查表访问权限
	if err := dt.validateTableAccess(sqlStr); err != nil {
		return "", err
	}
	
	// 执行查询
	rows, err := dt.db.QueryContext(ctx, sqlStr)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	
	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}
	
	// 构建结果
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Query Results (%d columns):\n\n", len(columns)))
	result.WriteString(strings.Join(columns, " | "))
	result.WriteString("\n")
	result.WriteString(strings.Repeat("-", len(columns)*15))
	result.WriteString("\n")
	
	// 读取行
	rowCount := 0
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	
	for rows.Next() {
		if rowCount >= dt.config.MaxRows {
			result.WriteString(fmt.Sprintf("\n... (limited to %d rows)\n", dt.config.MaxRows))
			break
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		
		rowData := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				rowData[i] = "NULL"
			} else {
				rowData[i] = fmt.Sprintf("%v", val)
			}
		}
		
		result.WriteString(strings.Join(rowData, " | "))
		result.WriteString("\n")
		rowCount++
	}
	
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating rows: %w", err)
	}
	
	result.WriteString(fmt.Sprintf("\nTotal rows: %d\n", rowCount))
	
	return result.String(), nil
}

// executeStatement 执行语句（INSERT、UPDATE、DELETE）
func (dt *DatabaseTool) executeStatement(ctx context.Context, sqlStr string) (string, error) {
	// 检查表访问权限
	if err := dt.validateTableAccess(sqlStr); err != nil {
		return "", err
	}
	
	// 执行语句
	result, err := dt.db.ExecContext(ctx, sqlStr)
	if err != nil {
		return "", fmt.Errorf("execution failed: %w", err)
	}
	
	// 获取影响的行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return fmt.Sprintf("Successfully executed. Rows affected: %d", rowsAffected), nil
}

// listTables 列出所有表
func (dt *DatabaseTool) listTables(ctx context.Context) (string, error) {
	var query string
	
	switch dt.config.Type {
	case DBTypeSQLite:
		query = "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
	case DBTypePostgreSQL:
		query = "SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename"
	case DBTypeMySQL:
		query = "SHOW TABLES"
	default:
		return "", fmt.Errorf("unsupported database type for list_tables: %s", dt.config.Type)
	}
	
	rows, err := dt.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()
	
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return "", fmt.Errorf("failed to scan table name: %w", err)
		}
		
		// 过滤允许的表
		if len(dt.config.AllowedTables) > 0 {
			allowed := false
			for _, allowedTable := range dt.config.AllowedTables {
				if table == allowedTable {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		
		tables = append(tables, table)
	}
	
	if len(tables) == 0 {
		return "No tables found (or no tables accessible)", nil
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Available tables (%d):\n\n", len(tables)))
	for i, table := range tables {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, table))
	}
	
	return result.String(), nil
}

// describeTable 描述表结构
func (dt *DatabaseTool) describeTable(ctx context.Context, table string) (string, error) {
	// 验证表访问权限
	if err := dt.validateTableName(table); err != nil {
		return "", err
	}
	
	var query string
	
	switch dt.config.Type {
	case DBTypeSQLite:
		query = fmt.Sprintf("PRAGMA table_info(%s)", table)
	case DBTypePostgreSQL:
		query = fmt.Sprintf(`
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = '%s'
			ORDER BY ordinal_position`, table)
	case DBTypeMySQL:
		query = fmt.Sprintf("DESCRIBE %s", table)
	default:
		return "", fmt.Errorf("unsupported database type for describe_table: %s", dt.config.Type)
	}
	
	rows, err := dt.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()
	
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Table: %s\n\n", table))
	result.WriteString(strings.Join(columns, " | "))
	result.WriteString("\n")
	result.WriteString(strings.Repeat("-", len(columns)*20))
	result.WriteString("\n")
	
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		
		rowData := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				rowData[i] = "NULL"
			} else {
				rowData[i] = fmt.Sprintf("%v", val)
			}
		}
		
		result.WriteString(strings.Join(rowData, " | "))
		result.WriteString("\n")
	}
	
	return result.String(), nil
}

// validateTableAccess 验证表访问权限
func (dt *DatabaseTool) validateTableAccess(sqlStr string) error {
	if len(dt.config.AllowedTables) == 0 {
		return nil
	}
	
	// 简单的表名提取（实际项目中应使用 SQL 解析器）
	upperSQL := strings.ToUpper(sqlStr)
	
	for _, table := range dt.config.AllowedTables {
		if strings.Contains(upperSQL, strings.ToUpper(table)) {
			return nil
		}
	}
	
	return fmt.Errorf("SQL statement references tables not in allowed list")
}

// validateTableName 验证表名
func (dt *DatabaseTool) validateTableName(table string) error {
	if len(dt.config.AllowedTables) == 0 {
		return nil
	}
	
	for _, allowed := range dt.config.AllowedTables {
		if table == allowed {
			return nil
		}
	}
	
	return fmt.Errorf("access to table %s is not allowed", table)
}

// Close 关闭数据库连接
func (dt *DatabaseTool) Close() error {
	if dt.db != nil {
		return dt.db.Close()
	}
	return nil
}

// ToTypesTool 实现 Tool 接口
func (dt *DatabaseTool) ToTypesTool() types.Tool {
	return types.Tool{
		Name:        dt.GetName(),
		Description: dt.GetDescription(),
		Parameters:  dt.GetParameters(),
	}
}

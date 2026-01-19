package loaders

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zhucl121/langchain-go/pkg/types"
)

// PostgreSQLLoader 从 PostgreSQL 数据库加载数据
//
// 支持的功能:
//   - 从表加载数据
//   - 执行自定义 SQL 查询
//   - 分页加载大量数据
//   - 自定义列映射
//   - 元数据提取
//
// 使用示例:
//
//	config := loaders.PostgreSQLLoaderConfig{
//	    Host:     "localhost",
//	    Port:     5432,
//	    Database: "mydb",
//	    User:     "postgres",
//	    Password: "password",
//	}
//	loader := loaders.NewPostgreSQLLoader(config)
//	docs, _ := loader.LoadTable(ctx, "documents", "content")
//
type PostgreSQLLoader struct {
	config PostgreSQLLoaderConfig
	db     *sql.DB
}

// PostgreSQLLoaderConfig PostgreSQL 加载器配置
type PostgreSQLLoaderConfig struct {
	// Host 数据库主机
	Host string
	
	// Port 数据库端口（默认 5432）
	Port int
	
	// Database 数据库名称
	Database string
	
	// User 用户名
	User string
	
	// Password 密码
	Password string
	
	// SSLMode SSL 模式 (disable, require, verify-ca, verify-full)
	SSLMode string
	
	// ConnectTimeout 连接超时（秒）
	ConnectTimeout int
	
	// MaxOpenConns 最大打开连接数
	MaxOpenConns int
	
	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int
	
	// PageSize 分页大小（批量加载时）
	PageSize int
}

// NewPostgreSQLLoader 创建新的 PostgreSQL 加载器
//
// 注意：需要安装 PostgreSQL 驱动
// import _ "github.com/lib/pq"
//
func NewPostgreSQLLoader(config PostgreSQLLoaderConfig) (*PostgreSQLLoader, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("postgresql loader: host is required")
	}
	
	if config.Database == "" {
		return nil, fmt.Errorf("postgresql loader: database is required")
	}
	
	if config.User == "" {
		return nil, fmt.Errorf("postgresql loader: user is required")
	}
	
	// 设置默认值
	if config.Port == 0 {
		config.Port = 5432
	}
	
	if config.SSLMode == "" {
		config.SSLMode = "disable"
	}
	
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 10
	}
	
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 10
	}
	
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 5
	}
	
	if config.PageSize == 0 {
		config.PageSize = 100
	}
	
	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		config.Host, config.Port, config.User, config.Password,
		config.Database, config.SSLMode, config.ConnectTimeout)
	
	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("postgresql loader: failed to open database: %w", err)
	}
	
	// 配置连接池
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.ConnectTimeout)*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgresql loader: failed to ping database: %w", err)
	}
	
	return &PostgreSQLLoader{
		config: config,
		db:     db,
	}, nil
}

// Close 关闭数据库连接
func (l *PostgreSQLLoader) Close() error {
	if l.db != nil {
		return l.db.Close()
	}
	return nil
}

// LoadTable 从表加载数据
//
// 参数:
//   - tableName: 表名
//   - contentColumn: 作为文档内容的列名
//   - metadataColumns: 要包含在元数据中的列名（可选）
//
func (l *PostgreSQLLoader) LoadTable(ctx context.Context, tableName, contentColumn string, metadataColumns ...string) ([]types.Document, error) {
	if tableName == "" {
		return nil, fmt.Errorf("postgresql loader: table name is required")
	}
	
	if contentColumn == "" {
		return nil, fmt.Errorf("postgresql loader: content column is required")
	}
	
	// 构建查询
	columns := []string{contentColumn}
	columns = append(columns, metadataColumns...)
	
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), tableName)
	
	return l.LoadQuery(ctx, query, contentColumn)
}

// LoadQuery 执行自定义查询并加载数据
//
// 参数:
//   - query: SQL 查询语句
//   - contentColumn: 作为文档内容的列名
//
func (l *PostgreSQLLoader) LoadQuery(ctx context.Context, query, contentColumn string) ([]types.Document, error) {
	if query == "" {
		return nil, fmt.Errorf("postgresql loader: query is required")
	}
	
	if contentColumn == "" {
		return nil, fmt.Errorf("postgresql loader: content column is required")
	}
	
	rows, err := l.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgresql loader: query failed: %w", err)
	}
	defer rows.Close()
	
	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("postgresql loader: failed to get columns: %w", err)
	}
	
	// 查找内容列索引
	contentIdx := -1
	for i, col := range columns {
		if col == contentColumn {
			contentIdx = i
			break
		}
	}
	
	if contentIdx == -1 {
		return nil, fmt.Errorf("postgresql loader: content column %q not found", contentColumn)
	}
	
	var documents []types.Document
	
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		// 扫描行
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("postgresql loader: failed to scan row: %w", err)
		}
		
		// 提取内容
		content := l.valueToString(values[contentIdx])
		
		// 构建元数据
		metadata := make(map[string]interface{})
		for i, col := range columns {
			if i != contentIdx {
				metadata[col] = values[i]
			}
		}
		
		documents = append(documents, types.Document{
			PageContent: content,
			Metadata:    metadata,
		})
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgresql loader: rows error: %w", err)
	}
	
	return documents, nil
}

// LoadTablePaginated 分页加载表数据（用于大表）
//
// 参数:
//   - tableName: 表名
//   - contentColumn: 作为文档内容的列名
//   - orderBy: 排序列（用于分页）
//   - metadataColumns: 要包含在元数据中的列名（可选）
//
func (l *PostgreSQLLoader) LoadTablePaginated(ctx context.Context, tableName, contentColumn, orderBy string, metadataColumns ...string) (<-chan types.Document, <-chan error) {
	docChan := make(chan types.Document, l.config.PageSize)
	errChan := make(chan error, 1)
	
	go func() {
		defer close(docChan)
		defer close(errChan)
		
		offset := 0
		
		for {
			// 构建分页查询
			columns := []string{contentColumn}
			columns = append(columns, metadataColumns...)
			
			query := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s LIMIT %d OFFSET %d",
				strings.Join(columns, ", "), tableName, orderBy, l.config.PageSize, offset)
			
			// 执行查询
			docs, err := l.LoadQuery(ctx, query, contentColumn)
			if err != nil {
				errChan <- err
				return
			}
			
			// 如果没有更多数据，退出
			if len(docs) == 0 {
				break
			}
			
			// 发送文档
			for _, doc := range docs {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case docChan <- doc:
				}
			}
			
			// 如果返回的文档少于页面大小，说明已经到最后一页
			if len(docs) < l.config.PageSize {
				break
			}
			
			offset += l.config.PageSize
		}
	}()
	
	return docChan, errChan
}

// LoadWithFilter 使用 WHERE 条件加载数据
func (l *PostgreSQLLoader) LoadWithFilter(ctx context.Context, tableName, contentColumn, whereClause string, metadataColumns ...string) ([]types.Document, error) {
	columns := []string{contentColumn}
	columns = append(columns, metadataColumns...)
	
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s",
		strings.Join(columns, ", "), tableName, whereClause)
	
	return l.LoadQuery(ctx, query, contentColumn)
}

// GetTableSchema 获取表结构信息
func (l *PostgreSQLLoader) GetTableSchema(ctx context.Context, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`
	
	rows, err := l.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("postgresql loader: failed to get schema: %w", err)
	}
	defer rows.Close()
	
	var columns []ColumnInfo
	
	for rows.Next() {
		var col ColumnInfo
		var nullable, defaultValue sql.NullString
		
		if err := rows.Scan(&col.Name, &col.DataType, &nullable, &defaultValue); err != nil {
			return nil, err
		}
		
		col.Nullable = nullable.String == "YES"
		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}
		
		columns = append(columns, col)
	}
	
	return columns, nil
}

// ==================== 辅助方法 ====================

func (l *PostgreSQLLoader) valueToString(value interface{}) string {
	if value == nil {
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ==================== 类型定义 ====================

// ColumnInfo 列信息
type ColumnInfo struct {
	Name         string
	DataType     string
	Nullable     bool
	DefaultValue string
}

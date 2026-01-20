package neo4j

import (
	"time"
)

// Config Neo4j 驱动器配置
type Config struct {
	// URI Neo4j 连接地址
	// 格式: bolt://host:port 或 neo4j://host:port
	// 示例: "bolt://localhost:7687"
	URI string

	// Username 用户名
	// 默认: "neo4j"
	Username string

	// Password 密码
	Password string

	// Database 数据库名称
	// 默认: "neo4j"
	Database string

	// MaxConnectionPoolSize 最大连接池大小
	// 默认: 100
	MaxConnectionPoolSize int

	// ConnectionAcquisitionTimeout 获取连接超时时间
	// 默认: 60s
	ConnectionAcquisitionTimeout time.Duration

	// MaxConnectionLifetime 连接最大生命周期
	// 默认: 1小时
	MaxConnectionLifetime time.Duration

	// MaxTransactionRetryTime 事务最大重试时间
	// 默认: 30s
	MaxTransactionRetryTime time.Duration

	// Encrypted 是否使用加密连接
	// 默认: false
	Encrypted bool

	// TrustStrategy TLS 信任策略
	// 默认: TrustSystemCAs
	TrustStrategy TrustStrategy
}

// TrustStrategy TLS 信任策略
type TrustStrategy string

const (
	// TrustSystemCAs 信任系统 CA
	TrustSystemCAs TrustStrategy = "trust_system_cas"

	// TrustAllCertificates 信任所有证书（不推荐用于生产环境）
	TrustAllCertificates TrustStrategy = "trust_all_certificates"
)

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		URI:                          "bolt://localhost:7687",
		Username:                     "neo4j",
		Password:                     "password",
		Database:                     "neo4j",
		MaxConnectionPoolSize:        100,
		ConnectionAcquisitionTimeout: 60 * time.Second,
		MaxConnectionLifetime:        1 * time.Hour,
		MaxTransactionRetryTime:      30 * time.Second,
		Encrypted:                    false,
		TrustStrategy:                TrustSystemCAs,
	}
}

// Validate 验证配置
func (c Config) Validate() error {
	if c.URI == "" {
		return ErrInvalidConfig("URI is required")
	}
	if c.Username == "" {
		return ErrInvalidConfig("Username is required")
	}
	if c.Password == "" {
		return ErrInvalidConfig("Password is required")
	}
	if c.Database == "" {
		return ErrInvalidConfig("Database is required")
	}
	if c.MaxConnectionPoolSize <= 0 {
		return ErrInvalidConfig("MaxConnectionPoolSize must be positive")
	}
	return nil
}

// ErrInvalidConfig 无效配置错误
type ErrInvalidConfig string

func (e ErrInvalidConfig) Error() string {
	return "neo4j: invalid config: " + string(e)
}

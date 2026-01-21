package nebula

import (
	"fmt"
	"time"
)

// Config NebulaGraph 驱动器配置
type Config struct {
	// Addresses NebulaGraph 图服务地址列表
	// 格式: []string{"host1:port1", "host2:port2"}
	// 默认: []string{"127.0.0.1:9669"}
	Addresses []string

	// Username 用户名
	// 默认: "root"
	Username string

	// Password 密码
	// 默认: "nebula"
	Password string

	// Space 图空间名称
	// 必需字段
	Space string

	// Timeout 连接超时时间
	// 默认: 30s
	Timeout time.Duration

	// IdleTime 连接空闲时间
	// 默认: 60s
	IdleTime time.Duration

	// MaxConnPoolSize 最大连接池大小
	// 默认: 100
	MaxConnPoolSize int

	// MinConnPoolSize 最小连接池大小
	// 默认: 10
	MinConnPoolSize int

	// ReconnectInterval 重连间隔
	// 默认: 1s
	ReconnectInterval time.Duration
}

// DefaultConfig 返回默认配置
//
// 默认配置：
//   - Addresses: ["127.0.0.1:9669"]
//   - Username: "root"
//   - Password: "nebula"
//   - Space: "langchain"
//   - Timeout: 30s
//   - IdleTime: 60s
//   - MaxConnPoolSize: 100
//   - MinConnPoolSize: 10
//
func DefaultConfig() Config {
	return Config{
		Addresses:         []string{"127.0.0.1:9669"},
		Username:          "root",
		Password:          "nebula",
		Space:             "langchain",
		Timeout:           30 * time.Second,
		IdleTime:          60 * time.Second,
		MaxConnPoolSize:   100,
		MinConnPoolSize:   10,
		ReconnectInterval: 1 * time.Second,
	}
}

// Validate 验证配置
//
// 验证规则：
//   - Addresses 不能为空
//   - Username 不能为空
//   - Password 不能为空
//   - Space 不能为空
//   - MaxConnPoolSize 必须 > 0
//   - MinConnPoolSize 必须 > 0
//   - Timeout 必须 > 0
//
func (c *Config) Validate() error {
	if len(c.Addresses) == 0 {
		return fmt.Errorf("nebula: addresses is required")
	}

	if c.Username == "" {
		return fmt.Errorf("nebula: username is required")
	}

	if c.Password == "" {
		return fmt.Errorf("nebula: password is required")
	}

	if c.Space == "" {
		return fmt.Errorf("nebula: space is required")
	}

	if c.MaxConnPoolSize <= 0 {
		c.MaxConnPoolSize = 100
	}

	if c.MinConnPoolSize <= 0 {
		c.MinConnPoolSize = 10
	}

	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	if c.IdleTime <= 0 {
		c.IdleTime = 60 * time.Second
	}

	if c.ReconnectInterval <= 0 {
		c.ReconnectInterval = 1 * time.Second
	}

	// MinConnPoolSize 不能大于 MaxConnPoolSize
	if c.MinConnPoolSize > c.MaxConnPoolSize {
		c.MinConnPoolSize = c.MaxConnPoolSize
	}

	return nil
}

// WithAddresses 设置地址列表
func (c Config) WithAddresses(addresses []string) Config {
	c.Addresses = addresses
	return c
}

// WithUsername 设置用户名
func (c Config) WithUsername(username string) Config {
	c.Username = username
	return c
}

// WithPassword 设置密码
func (c Config) WithPassword(password string) Config {
	c.Password = password
	return c
}

// WithSpace 设置空间名称
func (c Config) WithSpace(space string) Config {
	c.Space = space
	return c
}

// WithTimeout 设置超时时间
func (c Config) WithTimeout(timeout time.Duration) Config {
	c.Timeout = timeout
	return c
}

// WithPoolSize 设置连接池大小
func (c Config) WithPoolSize(min, max int) Config {
	c.MinConnPoolSize = min
	c.MaxConnPoolSize = max
	return c
}

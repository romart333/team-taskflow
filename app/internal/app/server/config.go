package server

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the root service configuration loaded from YAML with ENV overrides.
type Config struct {
	HTTP       HTTPConfig       `yaml:"http"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	Redis      RedisConfig      `yaml:"redis"`
	Auth       AuthConfig       `yaml:"auth"`
	Cache      CacheConfig      `yaml:"cache"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Email      EmailConfig      `yaml:"email"`
	Pagination PaginationConfig `yaml:"pagination"`
	Log        LogConfig        `yaml:"log"`
}

type HTTPConfig struct {
	Addr            string        `yaml:"addr" env:"HTTP_ADDR"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT"`
}

type MySQLConfig struct {
	Host            string        `yaml:"host" env:"MYSQL_HOST"`
	Port            int           `yaml:"port" env:"MYSQL_PORT"`
	User            string        `yaml:"user" env:"MYSQL_USER"`
	Password        string        `yaml:"password" env:"MYSQL_PASSWORD"`
	Database        string        `yaml:"database" env:"MYSQL_DATABASE"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"MYSQL_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"MYSQL_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"MYSQL_CONN_MAX_LIFETIME"`
}

// DSN builds the go-sql-driver/mysql connection string.
func (c MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

type RedisConfig struct {
	Addr     string `yaml:"addr" env:"REDIS_ADDR"`
	Password string `yaml:"password" env:"REDIS_PASSWORD"`
	DB       int    `yaml:"db" env:"REDIS_DB"`
}

type AuthConfig struct {
	JWTSecret  string        `yaml:"jwt_secret" env:"AUTH_JWT_SECRET"`
	TokenTTL   time.Duration `yaml:"token_ttl" env:"AUTH_TOKEN_TTL"`
	BcryptCost int           `yaml:"bcrypt_cost" env:"AUTH_BCRYPT_COST"`
}

type CacheConfig struct {
	TaskListTTL time.Duration `yaml:"task_list_ttl" env:"CACHE_TASK_LIST_TTL"`
}

type RateLimitConfig struct {
	Enabled  bool          `yaml:"enabled" env:"RATE_LIMIT_ENABLED"`
	Requests int           `yaml:"requests" env:"RATE_LIMIT_REQUESTS"`
	Window   time.Duration `yaml:"window" env:"RATE_LIMIT_WINDOW"`
}

type EmailConfig struct {
	BaseURL             string        `yaml:"base_url" env:"EMAIL_BASE_URL"`
	RequestTimeout      time.Duration `yaml:"request_timeout" env:"EMAIL_REQUEST_TIMEOUT"`
	BreakerMaxRequests  uint32        `yaml:"breaker_max_requests" env:"EMAIL_BREAKER_MAX_REQUESTS"`
	BreakerInterval     time.Duration `yaml:"breaker_interval" env:"EMAIL_BREAKER_INTERVAL"`
	BreakerTimeout      time.Duration `yaml:"breaker_timeout" env:"EMAIL_BREAKER_TIMEOUT"`
	BreakerMaxFailures  uint32        `yaml:"breaker_max_failures" env:"EMAIL_BREAKER_MAX_FAILURES"`
}

type PaginationConfig struct {
	DefaultPageSize int `yaml:"default_page_size" env:"PAGINATION_DEFAULT_PAGE_SIZE"`
	MaxPageSize     int `yaml:"max_page_size" env:"PAGINATION_MAX_PAGE_SIZE"`
}

type LogConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL"`
}

// LoadConfig reads configuration from the YAML file at path and applies ENV overrides.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("reading config %q: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.HTTP.Addr == "" {
		return fmt.Errorf("http.addr is required")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required")
	}
	if c.MySQL.Host == "" || c.MySQL.Database == "" {
		return fmt.Errorf("mysql.host and mysql.database are required")
	}
	if c.Pagination.DefaultPageSize <= 0 || c.Pagination.MaxPageSize <= 0 {
		return fmt.Errorf("pagination sizes must be positive")
	}
	return nil
}

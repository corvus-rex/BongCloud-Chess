package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration object for the application.
// All sub-configs are populated from environment variables or a .env file.
type Config struct {
	App       AppConfig
	Mongo     MongoConfig
	Redis     RedisConfig
	WebSocket WebSocketConfig
	Server    ServerConfig
	Log       LogConfig
}

type AppConfig struct {
	Env     string `mapstructure:"APP_ENV"`
	Port    int    `mapstructure:"APP_PORT"`
	Name    string `mapstructure:"APP_NAME"`
	Version string `mapstructure:"APP_VERSION"`
}

type MongoConfig struct {
	URI             string        `mapstructure:"MONGO_URI"`
	Database        string        `mapstructure:"MONGO_DATABASE"`
	Timeout         time.Duration `mapstructure:"-"`
	TimeoutSeconds  int           `mapstructure:"MONGO_TIMEOUT_SECONDS"`
	MaxPoolSize     uint64        `mapstructure:"MONGO_MAX_POOL_SIZE"`
	MinPoolSize     uint64        `mapstructure:"MONGO_MIN_POOL_SIZE"`
}

type RedisConfig struct {
	Host               string        `mapstructure:"REDIS_HOST"`
	Port               int           `mapstructure:"REDIS_PORT"`
	Password           string        `mapstructure:"REDIS_PASSWORD"`
	DB                 int           `mapstructure:"REDIS_DB"`
	PoolSize           int           `mapstructure:"REDIS_POOL_SIZE"`
	DialTimeout        time.Duration `mapstructure:"-"`
	DialTimeoutSeconds int           `mapstructure:"REDIS_DIAL_TIMEOUT_SECONDS"`
	ReadTimeout        time.Duration `mapstructure:"-"`
	ReadTimeoutSeconds int           `mapstructure:"REDIS_READ_TIMEOUT_SECONDS"`
	WriteTimeout        time.Duration `mapstructure:"-"`
	WriteTimeoutSeconds int           `mapstructure:"REDIS_WRITE_TIMEOUT_SECONDS"`
}

// Addr returns the host:port address for the Redis server.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type WebSocketConfig struct {
	ReadBufferSize         int           `mapstructure:"WS_READ_BUFFER_SIZE"`
	WriteBufferSize        int           `mapstructure:"WS_WRITE_BUFFER_SIZE"`
	HandshakeTimeout       time.Duration `mapstructure:"-"`
	HandshakeTimeoutSeconds int          `mapstructure:"WS_HANDSHAKE_TIMEOUT_SECONDS"`
	PingInterval           time.Duration `mapstructure:"-"`
	PingIntervalSeconds    int           `mapstructure:"WS_PING_INTERVAL_SECONDS"`
	PongWait               time.Duration `mapstructure:"-"`
	PongWaitSeconds        int           `mapstructure:"WS_PONG_WAIT_SECONDS"`
}

type ServerConfig struct {
	ReadTimeout         time.Duration `mapstructure:"-"`
	ReadTimeoutSeconds  int           `mapstructure:"SERVER_READ_TIMEOUT_SECONDS"`
	WriteTimeout        time.Duration `mapstructure:"-"`
	WriteTimeoutSeconds int           `mapstructure:"SERVER_WRITE_TIMEOUT_SECONDS"`
	IdleTimeout         time.Duration `mapstructure:"-"`
	IdleTimeoutSeconds  int           `mapstructure:"SERVER_IDLE_TIMEOUT_SECONDS"`
	ShutdownTimeout        time.Duration `mapstructure:"-"`
	ShutdownTimeoutSeconds int           `mapstructure:"SERVER_SHUTDOWN_TIMEOUT_SECONDS"`
}

type LogConfig struct {
	Level    string `mapstructure:"LOG_LEVEL"`
	Encoding string `mapstructure:"LOG_ENCODING"`
}

// Load reads configuration from environment variables and an optional .env file.
// Environment variables always take precedence over the .env file.
func Load() (*Config, error) {
	v := viper.New()

	// Automatically read from environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Attempt to load .env file (non-fatal if missing in production)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		}
	}

	setDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Convert raw second values into time.Duration fields
	applyDurations(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// App
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", 8080)
	v.SetDefault("APP_NAME", "bongcloud-chess")
	v.SetDefault("APP_VERSION", "0.0.1")

	// Mongo
	v.SetDefault("MONGO_TIMEOUT_SECONDS", 10)
	v.SetDefault("MONGO_MAX_POOL_SIZE", 100)
	v.SetDefault("MONGO_MIN_POOL_SIZE", 5)

	// Redis
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("REDIS_POOL_SIZE", 20)
	v.SetDefault("REDIS_DIAL_TIMEOUT_SECONDS", 5)
	v.SetDefault("REDIS_READ_TIMEOUT_SECONDS", 3)
	v.SetDefault("REDIS_WRITE_TIMEOUT_SECONDS", 3)

	// WebSocket
	v.SetDefault("WS_READ_BUFFER_SIZE", 1024)
	v.SetDefault("WS_WRITE_BUFFER_SIZE", 1024)
	v.SetDefault("WS_HANDSHAKE_TIMEOUT_SECONDS", 10)
	v.SetDefault("WS_PING_INTERVAL_SECONDS", 30)
	v.SetDefault("WS_PONG_WAIT_SECONDS", 60)

	// Server
	v.SetDefault("SERVER_READ_TIMEOUT_SECONDS", 15)
	v.SetDefault("SERVER_WRITE_TIMEOUT_SECONDS", 15)
	v.SetDefault("SERVER_IDLE_TIMEOUT_SECONDS", 60)
	v.SetDefault("SERVER_SHUTDOWN_TIMEOUT_SECONDS", 30)

	// Logging
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_ENCODING", "json")
}

func applyDurations(cfg *Config) {
	cfg.Mongo.Timeout = time.Duration(cfg.Mongo.TimeoutSeconds) * time.Second

	cfg.Redis.DialTimeout = time.Duration(cfg.Redis.DialTimeoutSeconds) * time.Second
	cfg.Redis.ReadTimeout = time.Duration(cfg.Redis.ReadTimeoutSeconds) * time.Second
	cfg.Redis.WriteTimeout = time.Duration(cfg.Redis.WriteTimeoutSeconds) * time.Second

	cfg.WebSocket.HandshakeTimeout = time.Duration(cfg.WebSocket.HandshakeTimeoutSeconds) * time.Second
	cfg.WebSocket.PingInterval = time.Duration(cfg.WebSocket.PingIntervalSeconds) * time.Second
	cfg.WebSocket.PongWait = time.Duration(cfg.WebSocket.PongWaitSeconds) * time.Second

	cfg.Server.ReadTimeout = time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second
	cfg.Server.WriteTimeout = time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second
	cfg.Server.IdleTimeout = time.Duration(cfg.Server.IdleTimeoutSeconds) * time.Second
	cfg.Server.ShutdownTimeout = time.Duration(cfg.Server.ShutdownTimeoutSeconds) * time.Second
}

func validate(cfg *Config) error {
	if cfg.Mongo.URI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}
	if cfg.Mongo.Database == "" {
		return fmt.Errorf("MONGO_DATABASE is required")
	}
	if cfg.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	return nil
}

// IsDevelopment returns true when running in a development environment.
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true when running in a production environment.
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
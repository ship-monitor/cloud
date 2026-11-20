package config

import (
	"flag"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/ship-monitor/cloud/internal/v2/adapters/clickhousedb"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/pkg/configer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Mode        string
	Host        string
	Port        string
	SecurityKey string
	DB          db.Config
	ClickHouse  clickhousedb.Config
	NatsURL     string
	ValkeyURL   string
	CorsOrigins []string

	LogLevel zapcore.Level
}

func (c *Config) IsDevelopment() bool {
	return strings.Contains(strings.ToUpper(c.Mode), "DEV")
}

func New() *Config {
	_ = godotenv.Load()
	level := zap.LevelFlag("log-level", zap.InfoLevel, "Sets the logger level")

	flag.Parse()

	return &Config{
		Host:        configer.GetOrDefault("HOST", "127.0.0.1"),
		Port:        configer.GetOrDefault("PORT", "6767"),
		SecurityKey: os.Getenv("SECURITY_KEY"),
		Mode:        configer.GetOrDefault("MODE", "DEVELOPMENT"),
		DB: db.Config{
			DSN: os.Getenv("DB_DSN"),
		},
		ClickHouse: clickhousedb.Config{
			DSN: os.Getenv("CLICKHOUSE_DSN"),
		},
		NatsURL:     os.Getenv("NATS_URL"),
		ValkeyURL:   os.Getenv("VALKEY_URL"),
		CorsOrigins: configer.GetStrings("CORS_ORIGINS", ","),
		LogLevel:    *level,
	}
}

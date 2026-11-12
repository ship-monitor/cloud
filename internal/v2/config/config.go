package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/ship-monitor/cloud/internal/v2/adapters/clickhousedb"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
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
}

func (c *Config) IsDevelopment() bool {
	return strings.Contains(c.Mode, "DEV")
}

func New() *Config {
	_ = godotenv.Load()

	return &Config{
		Host:        getOrDefault("HOST", "127.0.0.1"),
		Port:        os.Getenv("PORT"),
		SecurityKey: os.Getenv("SECURITY_KEY"),
		Mode:        getOrDefault("MODE", "DEVELOPMENT"),
		DB: db.Config{
			DSN: os.Getenv("DB_DSN"),
		},
		ClickHouse: clickhousedb.Config{
			DSN: os.Getenv("CLICKHOUSE_DSN"),
		},
		NatsURL:     os.Getenv("NATS_URL"),
		ValkeyURL:   os.Getenv("VALKEY_URL"),
		CorsOrigins: getStrings("CORS_ORIGINS", ","),
	}
}

func getOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	return ""
}

func getStrings(key, separator string) []string {
	val := os.Getenv(key)

	splitVal := strings.Split(val, separator)

	valStrings := []string{}

	for _, s := range splitVal {
		s = strings.TrimSpace(s)
		if s != "" {
			valStrings = append(valStrings, s)
		}
	}

	return valStrings
}

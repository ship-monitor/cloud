package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
)

type Config struct {
	Port        string
	devel       string
	SecurityKey string
	DB          db.Config
	NatsURL     string
	ValkeyURL   string
}

func (c *Config) IsDevelopment() bool {
	return c.devel != ""
}

func New() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:        os.Getenv("PORT"),
		SecurityKey: os.Getenv("SECURITY_KEY"),
		devel:       os.Getenv("DEVEL"),
		DB: db.Config{
			DSN: os.Getenv("DB_DSN"),
		},
		NatsURL:   os.Getenv("NATS_URL"),
		ValkeyURL: os.Getenv("VALKEY_URL"),
	}
}

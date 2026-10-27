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
			Driver: os.Getenv("DB_DRIVER"),
			DSN:    os.Getenv("DB_DSN"),
		},
	}
}

package config

import (
	"strings"

	"charm.land/log/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var Config *viper.Viper

func Setup() {
	_ = godotenv.Load()

	viper.SetConfigName("ship")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Warn("Failed to read config file, using environment variables only", "error", err)
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	Config = viper.GetViper()
}

func SecurityKey() []byte {
	return []byte(viper.GetString("jwt-security-key"))
}

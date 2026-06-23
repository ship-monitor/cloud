package config

import (
	"strings"

	"charm.land/log/v2"
	"github.com/joho/godotenv"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var Config *viper.Viper

func Setup() {
	_ = godotenv.Load()

	viper.SetConfigName("ship")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")

	err := viper.ReadInConfig()
	if err != nil {
		log.Warn("Failed to read config file, using environment variables only", "error", err)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	Config = viper.GetViper()

	flag.Parse()

	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		log.Fatal("Failed to bind flags", "error", err)
	}
}

func SecurityKey() []byte {
	return []byte(viper.GetString("jwt-security-key"))
}

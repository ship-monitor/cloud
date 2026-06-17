package config

import (
	"strings"

	"charm.land/log/v2"
	"github.com/joho/godotenv"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var Config *viper.Viper

var developmentMode = flag.Bool("devel", false, "Enable development mode")

func Setup() {
	_ = godotenv.Load()
	viper.SetConfigName("ship")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")
	if err := viper.ReadInConfig(); err != nil {
		log.Warn("Failed to read config file, using environment variables only", "error", err)
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	Config = viper.GetViper()

	flag.Parse()
	viper.BindPFlags(flag.CommandLine)
}

func SecurityKey() []byte {
	return []byte(viper.GetString("jwt-security-key"))
}

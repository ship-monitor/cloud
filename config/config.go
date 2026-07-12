package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func NewConfig(lc fx.Lifecycle) (*viper.Viper, error) {
	_ = godotenv.Load()

	viper.SetConfigName("ship")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")

	if os.Getenv("SHIP_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("SHIP_CONFIG"))
	}

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	flag.Bool("devel", false, "Specify to run app in development mode")

	flag.Parse()

	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		return nil, fmt.Errorf("failed bind flags: %w", err)
	}

	return viper.GetViper(), nil
}

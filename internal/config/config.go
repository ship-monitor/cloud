package config

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/ship-monitor/cloud/pkg/middleware"
	"github.com/ship-monitor/cloud/workers/server"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// Config is struct holding all config keys.
type Config struct {
	DatabaseURL string `mapstructure:"database-url"`
	RabbitMQURL string `mapstructure:"rabbitmq-url"`
	Redis       struct {
		URL      string `mapstructure:"url"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`

	SpiceDB struct {
		URL              string `mapstructure:"url"`
		GRPCPresharedKey string `mapstructure:"grpc-preshared-key"`
	} `mapstructure:"spicedb"`
}

type ConfigurationOutput struct {
	fx.Out

	Viper        *viper.Viper
	Config       *Config
	ServerConfig *server.Config
	Cookie       *middleware.CookieConfig
	CORS         *middleware.CORSConfig
}

func NewConfig(lc fx.Lifecycle) (ConfigurationOutput, error) {
	_ = godotenv.Load()

	viper.SetConfigName("ship")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")

	if os.Getenv("SHIP_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("SHIP_CONFIG"))
	}

	err := viper.ReadInConfig()
	if err != nil {
		return ConfigurationOutput{}, fmt.Errorf(
			"failed to read config: %w",
			err,
		)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	flag.Bool("devel", false, "Specify to run app in development mode")

	flag.Parse()

	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		return ConfigurationOutput{}, fmt.Errorf("failed bind flags: %w", err)
	}

	conf := Config{} //nolint:exhaustruct_v5

	err = viper.Unmarshal(&conf)
	if err != nil {
		return ConfigurationOutput{}, fmt.Errorf(
			"failed to unmarshal config: %w",
			err,
		)
	}

	cors := middleware.CORSConfig{
		AllowedOrigins: viper.GetStringSlice("cors.allow-origins"),
	}

	return ConfigurationOutput{
		Out: fx.Out{},

		Viper:  viper.GetViper(),
		Config: &conf,
		ServerConfig: &server.Config{
			CORS: cors,
			Port: viper.GetViper().GetInt("http.server.port"),
			ReadHeaderTimeout: viper.GetDuration(
				"http.server.read-header-timeout",
			),
		},
		Cookie: &middleware.CookieConfig{
			Name:     viper.GetString("auth.session.cookie.name"),
			Path:     viper.GetString("auth.session.cookie.path"),
			Domain:   viper.GetString("auth.session.cookie.domain"),
			Secure:   viper.GetBool("auth.session.cookie.secure"),
			SameSite: http.SameSiteNoneMode,
		},
		CORS: &cors,
	}, nil
}

package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const maxAge = 12 * time.Hour

func AllMethods() []string {
	return []string{
		http.MethodGet,
		http.MethodConnect,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}
}

type CORS struct {
	AllowedOrigins []string
}

func NewCORS() *CORS {
	return &CORS{
		AllowedOrigins: viper.GetStringSlice("cors.allow-origins"),
	}
}

func (c *CORS) Middleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: c.AllowedOrigins,
		AllowMethods: AllMethods(),
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           maxAge,
	})
}

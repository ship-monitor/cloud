package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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

type CORSConfig struct {
	AllowedOrigins []string
}

type CORS struct {
	conf *CORSConfig
}

func NewCORS(conf *CORSConfig) *CORS {
	return &CORS{
		conf: conf,
	}
}

func (c *CORS) Middleware() gin.HandlerFunc {
	return cors.New(cors.Config{ //nolint:exhaustruct_v5
		AllowOrigins: c.conf.AllowedOrigins,
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

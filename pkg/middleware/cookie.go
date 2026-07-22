package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CookieConfig struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

type CookieManager struct {
	config CookieConfig
}

func NewCookieManager(config CookieConfig) *CookieManager {
	return &CookieManager{config: config}
}

func (m *CookieManager) Read(c *gin.Context) (string, error) {
	token, err := c.Cookie(m.config.Name)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (m *CookieManager) Set(
	c *gin.Context,
	token string,
	expiresAt time.Time,
) {
	maxAge := max(int(time.Until(expiresAt).Seconds()), 0)

	c.SetSameSite(m.config.SameSite)

	c.SetCookie(
		m.config.Name,
		token,
		maxAge,
		m.config.Path,
		m.config.Domain,
		m.config.Secure,
		m.config.HTTPOnly,
	)
}

func (m *CookieManager) Clear(c *gin.Context) {
	c.SetSameSite(m.config.SameSite)

	c.SetCookie(
		m.config.Name,
		"",
		-1,
		m.config.Path,
		m.config.Domain,
		m.config.Secure,
		m.config.HTTPOnly,
	)
}

package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type CookieConfig struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

type AuthCookieManager struct {
	config CookieConfig
}

func NewAuthCookieManager(conf *viper.Viper) *AuthCookieManager {
	return &AuthCookieManager{config: CookieConfig{
		Name:     conf.GetString("auth.session.cookie.name"),
		Path:     conf.GetString("auth.session.cookie.path"),
		Domain:   conf.GetString("auth.session.cookie.domain"),
		Secure:   conf.GetBool("auth.session.cookie.secure"),
		HTTPOnly: true,
		SameSite: http.SameSiteNoneMode,
	}}
}

func (m *AuthCookieManager) Read(c *gin.Context) (string, error) {
	token, err := c.Cookie(m.config.Name)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (m *AuthCookieManager) Set(
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

func (m *AuthCookieManager) Clear(c *gin.Context) {
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

package middleware

import (
	"fmt"
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
		return "", fmt.Errorf("get cookie: %w", err)
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

	http.SetCookie(
		c.Writer, &http.Cookie{
			Name:     m.config.Name,
			MaxAge:   maxAge,
			Path:     m.config.Path,
			Domain:   m.config.Domain,
			Value:    token,
			Secure:   m.config.Secure,
			HttpOnly: m.config.HTTPOnly,
			SameSite: m.config.SameSite,
		},
	)
}

func (m *AuthCookieManager) Clear(c *gin.Context) {
	c.SetSameSite(m.config.SameSite)

	http.SetCookie(
		c.Writer, &http.Cookie{
			Name:     m.config.Name,
			MaxAge:   -1,
			Path:     m.config.Path,
			Domain:   m.config.Domain,
			Value:    "",
			Secure:   m.config.Secure,
			HttpOnly: m.config.HTTPOnly,
			SameSite: m.config.SameSite,
		},
	)
}

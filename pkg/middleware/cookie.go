package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
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

var _ handlers.CookieManager = (*AuthCookieManager)(nil)

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

func (m *AuthCookieManager) Read(c *echo.Context) (string, error) {
	token, err := c.Cookie(m.config.Name)
	if err != nil {
		return "", fmt.Errorf("get cookie: %w", err)
	}

	return token.Value, nil
}

func (m *AuthCookieManager) Set(
	c *echo.Context,
	token string,
	expiresAt time.Time,
) {
	maxAge := max(int(time.Until(expiresAt).Seconds()), 0)

	c.SetCookie(
		&http.Cookie{
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

func (m *AuthCookieManager) Clear(c *echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     m.config.Name,
		MaxAge:   -1,
		Path:     m.config.Path,
		Domain:   m.config.Domain,
		Value:    "",
		Secure:   m.config.Secure,
		HttpOnly: m.config.HTTPOnly,
		SameSite: m.config.SameSite,
	})
}

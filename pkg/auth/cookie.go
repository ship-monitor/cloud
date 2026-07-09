package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const DefaultSessionCookieName = "ship_session"

type CookieOptions struct {
	Name   string
	Domain string
	// Cookie TTL
	MaxAge   time.Duration
	Secure   bool
	SameSite http.SameSite
}

func SessionTTL(config *viper.Viper) time.Duration {
	ttl := config.GetDuration("session.ttl")
	if ttl <= 0 {
		return DefaultSessionTTL
	}

	return ttl
}

func CookieOptionsFromConfig(config *viper.Viper) CookieOptions {
	name := config.GetString("session.cookie-name")
	if name == "" {
		name = DefaultSessionCookieName
	}

	return CookieOptions{
		Name:     name,
		Domain:   config.GetString("session.cookie-domain"),
		MaxAge:   SessionTTL(config),
		Secure:   !config.GetBool("devel"),
		SameSite: sameSiteFromConfig(config.GetString("session.same-site")),
	}
}

func SetSessionCookie(ctx *gin.Context, opts CookieOptions, sessionID string) {
	if opts.Name == "" {
		opts.Name = DefaultSessionCookieName
	}

	if opts.MaxAge <= 0 {
		opts.MaxAge = DefaultSessionTTL
	}

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     opts.Name,
		Value:    sessionID,
		Path:     "/",
		Domain:   opts.Domain,
		MaxAge:   int(opts.MaxAge.Seconds()),
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

func ClearSessionCookie(ctx *gin.Context, opts CookieOptions) {
	if opts.Name == "" {
		opts.Name = DefaultSessionCookieName
	}

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     opts.Name,
		Value:    "",
		Path:     "/",
		Domain:   opts.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

func sameSiteFromConfig(value string) http.SameSite {
	switch value {
	case "strict", "Strict", "STRICT":
		return http.SameSiteStrictMode
	case "none", "None", "NONE":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

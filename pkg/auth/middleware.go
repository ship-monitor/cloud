package auth

import (
	"errors"
	"fmt"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

const (
	// Specify key to search [Session] in [gin.Context].
	sessionContextKey = "ship-auth-session"
	// Specify key to search [Middleware] in [gin.Context].
	middlewareContextKey = "ship-auth-middleware"
)

var (
	ErrNoSessionCookie = errors.New("session cookie not specified")
	ErrNoSessionInCtx  = errors.New(
		"no session in context, probably not authenticated",
	)
	ErrUnexpectedSessionType = errors.New(
		"session in context is of unexpected type",
	)
	ErrNoMiddlewareInCtx        = errors.New("no middleware in context")
	ErrUnexpectedMiddlewareType = errors.New(
		"middleware in context is of unexpected type",
	)
)

type Middleware struct {
	log        *log.Logger
	sessions   SessionStore
	cookieName string
}

// NewMiddleware returns a new [Middleware] with the default session and
// SpiceDB configuration from viper.
func NewMiddleware(sessions SessionStore, config *viper.Viper) *Middleware {
	viper.SetDefault("session.cookie-name", DefaultSessionCookieName)

	cookieName := viper.GetString("session.cookie-name")

	return &Middleware{
		log:        log.WithPrefix("Auth Middleware"),
		sessions:   sessions,
		cookieName: cookieName,
	}
}

func GetMiddleware(ctx *gin.Context) *Middleware {
	middleware, ok := ctx.Get(middlewareContextKey)
	if !ok {
		abortRequest(ctx, ErrNoMiddlewareInCtx)
	}

	m, ok := middleware.(*Middleware)
	if !ok {
		abortRequest(ctx, ErrUnexpectedMiddlewareType)
	}

	return m
}

func (m *Middleware) WithMiddleware(ctx *gin.Context) {
	m.addToContext(ctx)
}

func (m *Middleware) WithAuthenticationRequired(ctx *gin.Context) {
	m.addToContext(ctx)

	sessionID, err := ctx.Cookie(m.cookieName)
	if err != nil {
		err := fmt.Errorf("bad credentials: %w", ErrNoSessionCookie)
		m.log.Error("No session cookie", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseErr(err),
		)

		return
	}

	stored, err := m.sessions.Get(ctx.Request.Context(), sessionID)
	if err != nil {
		m.log.Error("Failed load session", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseErr(fmt.Errorf("bad credentials: %w", err)),
		)

		return
	}

	session := &Session{
		ID:     sessionID,
		UserID: stored.UserID,
		Email:  stored.Email,
		c:      ctx,
	}

	ctx.Set(sessionContextKey, session)
}

func (m *Middleware) addToContext(ctx *gin.Context) {
	ctx.Set(middlewareContextKey, m)
}

func GetSession(ctx *gin.Context) *Session {
	session, ok := ctx.Get(sessionContextKey)
	if !ok {
		abortRequest(ctx, ErrNoSessionInCtx)
	}

	s, ok := session.(*Session)
	if !ok {
		abortRequest(ctx, ErrUnexpectedSessionType)
	}

	return s
}

func abortRequest(ctx *gin.Context, err error) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, requests.ResponseErr(err))
	_ = ctx.Error(err)

	panic(fmt.Sprintf("aborting request: %s", err))
}

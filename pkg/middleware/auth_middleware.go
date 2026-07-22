package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type AuthService interface {
	Authenticate(
		ctx context.Context,
		token string,
	) (*domain.Principal, error)
}

type AuthMiddleware struct {
	authService AuthService
	cookies     *CookieManager
}

func NewAuthMiddleware(auth AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: auth,
		cookies: NewCookieManager(CookieConfig{
			Name:     "__Host-session",
			Path:     "/",
			Domain:   "",
			Secure:   true,
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		}),
	}
}

type principalContextKey struct{}

func PrincipalFromContext(ctx *gin.Context) (*domain.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(*domain.Principal)

	return principal, ok
}

func addPrincipalToContext(ctx *gin.Context, p *domain.Principal) {
	ctx.Set(principalContextKey{}, p)
}

func MustPrincipal(c *gin.Context) *domain.Principal {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		panic("session middleware is not registered")
	}

	return principal
}

func (s *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := s.cookies.Read(c)
		if err != nil {
			s.abortUnauthorized(c)

			return
		}

		principal, err := s.authService.Authenticate(
			c.Request.Context(),
			token,
		)
		switch {
		case errors.Is(err, services.ErrUnauthenticated),
			errors.Is(err, services.ErrSessionExpired),
			errors.Is(err, services.ErrSessionRevoked):
			s.cookies.Clear(c)
			s.abortUnauthorized(c)
		case err != nil:
			c.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				requests.ResponseErr(err),
			)
		default:
			addPrincipalToContext(c, principal)

			c.Next()
		}
	}
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := m.cookies.Read(c)
		if err != nil {
			c.Next()

			return
		}

		principal, err := m.authService.Authenticate(
			c.Request.Context(),
			token,
		)
		switch {
		case errors.Is(err, services.ErrUnauthenticated), // TODO: remove service dependency
			errors.Is(err, services.ErrSessionExpired),
			errors.Is(err, services.ErrSessionRevoked):
			m.cookies.Clear(c)
			c.Next()

		case err != nil:
			c.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				requests.ResponseErr(err),
			)
		default:
			addPrincipalToContext(c, principal)
			c.Next()
		}
	}
}

func (a *AuthMiddleware) abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(
		http.StatusUnauthorized,
		requests.ResponseBad("unauthenticated"),
	)
}

package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"charm.land/log/v2"
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

	Logout(ctx context.Context, token string) error
}

type AuthMiddleware struct {
	authService AuthService
	cookies     *AuthCookieManager
	logger      *log.Logger
}

func NewAuthMiddleware(
	auth AuthService,
	cookies *AuthCookieManager,
	logger *log.Logger,
) *AuthMiddleware {
	return &AuthMiddleware{
		authService: auth,
		cookies:     cookies,
		logger:      logger.WithPrefix("AuthMiddleware"),
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
			s.abortUnauthorized(c, fmt.Errorf("read cookie: %w", err))

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
			s.abortUnauthorized(c, fmt.Errorf("service error: %w", err))
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

// Logout deletes session cookies and invoke [AuthService.Logout].
func (a *AuthMiddleware) Logout(ctx *gin.Context) error {
	token, err := a.cookies.Read(ctx)
	if err != nil {
		return fmt.Errorf("read cookie: %w", err)
	}

	a.cookies.Clear(ctx)

	if err := a.authService.Logout(ctx.Request.Context(), token); err != nil {
		return fmt.Errorf("auth service logout: %w", err)
	}

	return nil
}

func (a *AuthMiddleware) abortUnauthorized(c *gin.Context, err error) {
	log.Error("Abortin unauthorized", "error", err)

	c.AbortWithStatusJSON(
		http.StatusUnauthorized,
		requests.ResponseBad("unauthenticated"),
	)
}

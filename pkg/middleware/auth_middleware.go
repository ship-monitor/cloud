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

var (
	ErrNoPrincipalInCtx = errors.New("no value in context")
	ErrUnknownValueType = errors.New("unknown value type")
)

func PrincipalFromContext(ctx *gin.Context) (*domain.Principal, error) {
	val, ok := ctx.Get(principalContextKey{})
	if !ok {
		return nil, ErrNoPrincipalInCtx
	}

	principal, ok := val.(*domain.Principal)
	if !ok {
		return nil, fmt.Errorf("%w: value type %T", ErrUnknownValueType, val)
	}

	return principal, nil
}

func AddToContext(ctx *gin.Context, p *domain.Principal) {
	ctx.Set(principalContextKey{}, p)
}

func MustPrincipal(c *gin.Context) *domain.Principal {
	principal, err := PrincipalFromContext(c)
	if err != nil {
		panic("principal is not registered in context: " + err.Error())
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
			AddToContext(c, principal)

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
			AddToContext(c, principal)
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

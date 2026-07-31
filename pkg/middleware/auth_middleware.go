package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"charm.land/log/v2"
	"github.com/labstack/echo/v5"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionRevoked  = errors.New("session revoked")
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

var _ handlers.AuthMiddleware = (*AuthMiddleware)(nil)

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

const principalContextKey = "principal-context-key"

var (
	ErrNoPrincipalInCtx = errors.New("no value in context")
	ErrUnknownValueType = errors.New("unknown value type")
)

func (m *AuthMiddleware) PrincipalFromContext(
	ctx *echo.Context,
) (*domain.Principal, error) {
	principal, err := echo.ContextGet[*domain.Principal](
		ctx,
		principalContextKey,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoPrincipalInCtx, err)
	}

	return principal, nil
}

func AddToContext(ctx *echo.Context, p *domain.Principal) {
	ctx.Set(principalContextKey, p)
}

func (m *AuthMiddleware) MustPrincipal(c *echo.Context) *domain.Principal {
	principal, err := m.PrincipalFromContext(c)
	if err != nil {
		panic("principal is not registered in context, error: " + err.Error())
	}

	return principal
}

func (s *AuthMiddleware) RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token, err := s.cookies.Read(c)
			if err != nil {
				return c.JSON(
					http.StatusUnauthorized,
					requests.ResponseErr(err),
				)
			}

			principal, err := s.authService.Authenticate(
				c.Request().Context(),
				token,
			)
			switch {
			case errors.Is(err, ErrUnauthenticated),
				errors.Is(err, ErrSessionExpired),
				errors.Is(err, ErrSessionRevoked):
				s.cookies.Clear(c)

				return c.JSON(
					http.StatusUnauthorized,
					requests.ResponseErr(err),
				)
			case err != nil:
				return c.JSON(
					http.StatusServiceUnavailable,
					requests.ResponseErr(err),
				)
			default:
				AddToContext(c, principal)

				return next(c)
			}
		}
	}
}

// Logout deletes session cookies and invoke [AuthService.Logout].
func (a *AuthMiddleware) Logout(ctx *echo.Context) error {
	token, err := a.cookies.Read(ctx)
	if err != nil {
		return fmt.Errorf("read cookie: %w", err)
	}

	a.cookies.Clear(ctx)

	if err := a.authService.Logout(ctx.Request().Context(), token); err != nil {
		return fmt.Errorf("auth service logout: %w", err)
	}

	return nil
}

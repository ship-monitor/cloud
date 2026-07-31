package handlers

import (
	"context"
	"errors"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type AuthService interface {
	StartEmailConfirmation(ctx context.Context, userID uuid.UUID) error
	ConfirmEmail(ctx context.Context, userID uuid.UUID, token string) error
	Register(ctx context.Context, data domain.RegisterUserData) error
	Login(ctx context.Context, email, password string) (*domain.User, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	ChangePassword(
		ctx context.Context,
		userID uuid.UUID,
		oldPassword, newPassword string,
	) error
	ChangeEmail(ctx context.Context, userID uuid.UUID, newEmail string) error
}

var _ pkg.Handler = (*AuthHandlers)(nil)

type AuthHandlers struct {
	logger      *log.Logger
	authService AuthService
	sessions    *services.Sessions
	middleware  *middleware.AuthMiddleware
	cookie      *middleware.AuthCookieManager
}

func NewAuthHandlers(
	authService AuthService,
	sessions *services.Sessions,
	middleware *middleware.AuthMiddleware,
	cookie *middleware.AuthCookieManager,
) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
		sessions:    sessions,
		logger:      log.WithPrefix("Auth handlers"),
		middleware:  middleware,
		cookie:      cookie,
	}
}

// SetupRoutes implements [pkg.Handler].
func (a *AuthHandlers) SetupRoutes(router *echo.Group) {
	auth := router.Group("/api/auth")
	auth.POST("/register", a.HandleRegister)
	auth.POST("/login", a.HandleLogin)
	auth.POST("/logout", a.HandleLogout, a.middleware.RequireAuth())

	users := router.Group("/api/users", a.middleware.RequireAuth())
	users.GET("/me", a.HandleGetMe)
	users.GET("/:id", a.HandleGetUser)
	users.POST("/set-password", a.HandleUserSetPassword)
	users.POST("/set-email", a.HandleUserSetEmail)
	users.POST("/start-email-confirmation", a.HandleStartEmailConfirmation)
	users.POST("/confirm-email/:token", a.HandleConfirmEmail)

	sessions := router.Group("/api/sessions", a.middleware.RequireAuth())
	sessions.GET("", a.HandleListSessions)
	sessions.DELETE("/:id", a.HandleRevokeSession)
	sessions.DELETE("", a.HandleRevokeOtherSessions)
}

func (a *AuthHandlers) HandleRegister(c *echo.Context) error {
	var request domain.RegisterUserData

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	err := a.authService.Register(c.Request().Context(), request)
	switch {
	case errors.Is(err, services.ErrEmailTaken):
		return c.JSON(http.StatusConflict, requests.ResponseErr(err))
	case errors.Is(err, services.ErrInvalidRegisterData):
		return c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	case err != nil:
		return c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		return c.NoContent(http.StatusCreated)
	}
}

func (a *AuthHandlers) HandleLogin(c *echo.Context) error {
	var request struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	user, err := a.authService.Login(
		c.Request().Context(),
		request.Email,
		request.Password,
	)
	switch {
	case errors.Is(err, services.ErrBadCredentials):
		log.Error("Invalid credentials", "error", err, "email", request.Email)

		return c.JSON(
			http.StatusUnauthorized,
			requests.ResponseBad("invalid credentials"),
		)

	case err != nil:
		log.Error("Internal error", "error", err)

		return c.JSON(
			http.StatusUnauthorized,
			requests.ResponseBad("invalid credentials"),
		)
	}

	session, err := a.sessions.Create(
		c.Request().Context(),
		user.ID,
		services.ClientInfo{
			UserAgent: c.Request().UserAgent(),
			ClientIP:  c.RealIP(),
		},
	)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	}

	a.cookie.Set(c, session.Token, session.Session.ExpiresAt)

	return c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (a *AuthHandlers) HandleLogout(c *echo.Context) error {
	if err := a.middleware.Logout(c); err != nil {
		return c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	} else {
		return c.NoContent(http.StatusOK)
	}
}

type sessionItem struct {
	domain.Session

	Current bool `json:"current"`
}

// HandleListSessions returns all active sessions of the current user.
func (a *AuthHandlers) HandleListSessions(c *echo.Context) error {
	principal := middleware.MustPrincipal(c)

	list, err := a.sessions.List(c.Request().Context(), principal.UserID)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	}

	items := make([]sessionItem, 0, len(list))
	for _, s := range list {
		items = append(items, sessionItem{
			Session: s,
			Current: s.ID == principal.SessionID,
		})
	}

	return c.JSON(http.StatusOK, gin.H{"sessions": items})
}

// HandleRevokeSession terminates a single session by its ID.
func (a *AuthHandlers) HandleRevokeSession(c *echo.Context) error {
	principal := middleware.MustPrincipal(c)

	var request struct {
		SessionID uuid.UUID `param:"id"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	err := a.sessions.RevokeByID(
		c.Request().Context(),
		principal.UserID,
		request.SessionID,
	)
	switch {
	case errors.Is(err, services.ErrSessionNotFound):
		return c.NoContent(http.StatusNotFound)
	case err != nil:
		return c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		return c.NoContent(http.StatusOK)
	}
}

// HandleRevokeOtherSessions terminates every session except the current one.
func (a *AuthHandlers) HandleRevokeOtherSessions(c *echo.Context) error {
	principal := middleware.MustPrincipal(c)

	err := a.sessions.RevokeOthers(
		c.Request().Context(),
		principal.UserID,
		principal.SessionID,
	)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	}

	return c.NoContent(http.StatusOK)
}

func (a *AuthHandlers) HandleStartEmailConfirmation(ctx *echo.Context) error {
	session := middleware.MustPrincipal(ctx)

	err := a.authService.StartEmailConfirmation(
		ctx.Request().Context(),
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrEmailAlreadyConfirmed):
		return ctx.JSON(http.StatusNotModified, requests.ResponseErr(err))
	case err != nil:
		return ctx.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		return ctx.NoContent(http.StatusOK)
	}
}

func (a *AuthHandlers) HandleConfirmEmail(ctx *echo.Context) error {
	session := middleware.MustPrincipal(ctx)

	token := ctx.Param("token")

	err := a.authService.ConfirmEmail(
		ctx.Request().Context(),
		session.UserID,
		token,
	)
	switch {
	case errors.Is(err, services.ErrEmailAlreadyConfirmed):
		return ctx.JSON(http.StatusNotModified, requests.ResponseErr(err))
	case err != nil:
		return ctx.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		return ctx.NoContent(http.StatusOK)
	}
}

func (a *AuthHandlers) HandleGetMe(ctx *echo.Context) error {
	principal := middleware.MustPrincipal(ctx)

	user, err := a.authService.GetUser(
		ctx.Request().Context(),
		principal.UserID,
	)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, requests.ResponseErr(err))
	}

	return ctx.JSON(http.StatusOK, UserResponse{User: user})
}

func (a *AuthHandlers) HandleGetUser(ctx *echo.Context) error {
	var request struct {
		ID uuid.UUID `param:"id"`
	}

	user, err := a.authService.GetUser(ctx.Request().Context(), request.ID)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, requests.ResponseErr(err))
	}

	return ctx.JSON(http.StatusOK, UserResponse{User: user})
}

func (a *AuthHandlers) HandleUserSetPassword(ctx *echo.Context) error {
	session := middleware.MustPrincipal(ctx)

	var request struct {
		OldPassword string `binding:"required" json:"oldPassword"`
		Password    string `binding:"required" json:"password"`
	}
	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	err := a.authService.ChangePassword(
		ctx.Request().Context(),
		session.UserID,
		request.OldPassword,
		request.Password,
	)
	if err != nil {
		return ctx.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(err),
		)
	}

	return ctx.NoContent(http.StatusOK)
}

var ErrNotAllowedSetEmail = errors.New("not allowed to set email for this user")

func (a *AuthHandlers) HandleUserSetEmail(ctx *echo.Context) error {
	session := middleware.MustPrincipal(ctx)

	var request struct {
		Email string `binding:"required" json:"email"`
	}
	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	err := a.authService.ChangeEmail(
		ctx.Request().Context(),
		session.UserID,
		request.Email,
	)
	if err != nil {
		return ctx.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(err),
		)
	}

	return ctx.NoContent(http.StatusOK)
}

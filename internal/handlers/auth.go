package handlers

import (
	"context"
	"errors"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
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
	logger        *log.Logger
	authService   AuthService
	sessions      auth.SessionStore
	cookieOptions auth.CookieOptions
	middleware    auth.Middleware
}

func NewAuthHandlers(
	authService AuthService,
	sessions auth.SessionStore,
	cookieOptions auth.CookieOptions,
	middleware auth.Middleware,
) *AuthHandlers {
	return &AuthHandlers{
		authService:   authService,
		sessions:      sessions,
		cookieOptions: cookieOptions,
		logger:        log.WithPrefix("Auth handlers"),
		middleware:    middleware,
	}
}

// SetupRoutes implements [pkg.Handler].
func (a *AuthHandlers) SetupRoutes(router gin.IRouter) {
	auth := router.Group("/api/auth")
	auth.POST("/register", a.HandleRegister)
	auth.POST("/login", a.HandleLogin)
	auth.POST(
		"/refresh",
		a.middleware.WithAuthenticationRequired,
		a.HandleRefresh,
	)
	auth.POST("/logout", a.middleware.WithMiddleware, a.HandleLogout)
	users := router.Group("/api/users", a.middleware.WithAuthenticationRequired)
	users.GET("/me", a.HandleGetUser)
	users.GET("/:id", a.HandleGetUser)
	users.POST("/:id/set-password", a.HandleUserSetPassword)
	users.POST("/:id/set-email", a.HandleUserSetEmail)
	users.POST("/start-email-confirmation", a.HandleStartEmailConfirmation)
	users.POST("/confirm-email/:token", a.HandleConfirmEmail)
}

func (a *AuthHandlers) HandleRegister(c *gin.Context) {
	var request domain.RegisterUserData
	a.bindJSON(c, &request)

	err := a.authService.Register(c.Request.Context(), request)
	switch {
	case errors.Is(err, services.ErrEmailTaken):
		c.AbortWithStatusJSON(http.StatusConflict, requests.ResponseErr(err))
	case errors.Is(err, services.ErrInvalidRegisterData):
		c.AbortWithStatusJSON(http.StatusBadRequest, requests.ResponseErr(err))
	case err != nil:
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		c.Status(http.StatusCreated)
	}
}

func (a *AuthHandlers) HandleLogin(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	a.bindJSON(c, &request)

	user, err := a.authService.Login(
		c.Request.Context(),
		request.Email,
		request.Password,
	)
	switch {
	case errors.Is(err, services.ErrBadCredentials):
		log.Error("Invalid credentials", "email", request.Email)
		c.JSON(
			http.StatusUnauthorized,
			requests.ResponseBad("invalid credentials"),
		)

		return
	case err != nil:
		log.Error(
			"Failed get user by email",
			"error",
			err,
			"email",
			request.Email,
		)
		c.JSON(
			http.StatusUnauthorized,
			requests.ResponseBad("invalid credentials"),
		)

		return
	}

	sessionID, err := a.sessions.Create(
		c.Request.Context(),
		user.ID,
		user.Email,
	)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)

		return
	}

	auth.SetSessionCookie(c, a.cookieOptions, sessionID)

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (a *AuthHandlers) HandleRefresh(c *gin.Context) {
	session := auth.GetSession(c)

	stored, err := a.sessions.Refresh(c.Request.Context(), session.ID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseErr(err),
		)

		return
	}

	auth.SetSessionCookie(c, a.cookieOptions, session.ID)

	c.JSON(http.StatusOK, gin.H{
		"session": stored,
	})
}

func (a *AuthHandlers) HandleLogout(c *gin.Context) {
	sessionID, err := c.Cookie(a.cookieOptions.Name)
	if err == nil {
		if err := a.sessions.Delete(
			c.Request.Context(),
			sessionID,
		); err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				requests.ResponseErr(err),
			)

			return
		}
	}

	auth.ClearSessionCookie(c, a.cookieOptions)
	c.Status(http.StatusOK)
}

func (a *AuthHandlers) HandleStartEmailConfirmation(ctx *gin.Context) {
	session := auth.GetSession(ctx)

	err := a.authService.StartEmailConfirmation(
		ctx.Request.Context(),
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrEmailAlreadyConfirmed):
		ctx.JSON(http.StatusNotModified, requests.ResponseErr(err))
	case err != nil:
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		ctx.Status(http.StatusOK)
	}
}

func (a *AuthHandlers) HandleConfirmEmail(ctx *gin.Context) {
	session := auth.GetSession(ctx)

	token := ctx.Param("token")

	err := a.authService.ConfirmEmail(
		ctx.Request.Context(),
		session.UserID,
		token,
	)
	switch {
	case errors.Is(err, services.ErrEmailAlreadyConfirmed):
		ctx.JSON(http.StatusNotModified, requests.ResponseErr(err))
	case err != nil:
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		ctx.Status(http.StatusOK)
	}
}

func (a *AuthHandlers) HandleGetUser(ctx *gin.Context) {
	session := auth.GetSession(ctx)

	id, err := requests.GetParamUUID(ctx, "id")
	if errors.Is(err, requests.ErrNoParam) {
		id = session.UserID
	}

	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	user, err := a.authService.GetUser(ctx.Request.Context(), id)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, requests.ResponseErr(err))

		return
	}

	ctx.JSON(http.StatusOK, gin.H{"user": user})
}

func (a *AuthHandlers) HandleUserSetPassword(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatus(http.StatusNotFound)

		return
	}

	var request struct {
		OldPassword string `binding:"required" json:"oldPassword"`
		Password    string `binding:"required" json:"password"`
	}
	a.bindJSON(ctx, &request)

	err := a.authService.ChangePassword(
		ctx.Request.Context(),
		session.UserID,
		request.OldPassword,
		request.Password,
	)
	if err != nil {
		ctx.AbortWithStatusJSON(
			http.StatusBadRequest,
			requests.ResponseErr(err),
		)

		return
	}

	ctx.Status(http.StatusOK)
}

var ErrNotAllowedSetEmail = errors.New("not allowed to set email for this user")

func (a *AuthHandlers) HandleUserSetEmail(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatusJSON(
			http.StatusForbidden,
			requests.ResponseErr(ErrNotAllowedSetEmail),
		)

		return
	}

	var request struct {
		Email string `binding:"required" json:"email"`
	}
	a.bindJSON(ctx, &request)

	err := a.authService.ChangeEmail(
		ctx.Request.Context(),
		session.UserID,
		request.Email,
	)
	if err != nil {
		ctx.AbortWithStatusJSON(
			http.StatusBadRequest,
			requests.ResponseErr(err),
		)

		return
	}

	ctx.Status(http.StatusOK)
}

func (a *AuthHandlers) bindJSON(ctx *gin.Context, data any) {
	err := ctx.ShouldBindJSON(data)
	if err != nil {
		if vErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"details":          "body validation fails",
				"validationErrors": mapErrors(vErrors),
			})
			a.logger.Error("Validation fails", "error", err)
		}

		a.logger.Error("Failed to bind JSON", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusBadRequest,
			requests.BadResponse{Details: err.Error()},
		)

		return
	}
}

func mapErrors(vErrors validator.ValidationErrors) []gin.H {
	res := make([]gin.H, 0, len(vErrors))

	for _, err := range vErrors {
		res = append(res, gin.H{
			"field":       err.Field(),
			"error":       err.Error(),
			"actualTag":   err.ActualTag(),
			"tag":         err.Tag(),
			"structField": err.StructField(),
		})
	}

	return res
}

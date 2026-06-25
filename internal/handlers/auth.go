package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

const (
	tokenTTL        = time.Minute * 5
	refreshTokenTTL = time.Hour * 24
)

type AuthService interface {
	StartEmailConfirmation(ctx context.Context, userID uuid.UUID) error
	ConfirmEmail(ctx context.Context, userID uuid.UUID, token string) error
	Register(ctx context.Context, data domain.RegisterUserData) error
	Login(ctx context.Context, email, password string) (*domain.User, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
	ChangeEmail(ctx context.Context, userID uuid.UUID, newEmail string) error
}

type AuthHandlers struct {
	logger      *log.Logger
	authService AuthService
}

func NewAuthHandlers(authService AuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
		logger:      log.WithPrefix("Auth handlers"),
	}
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, requests.ResponseErr(err))
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

	user, err := a.authService.Login(c.Request.Context(), request.Email, request.Password)
	switch {
	case errors.Is(err, services.ErrBadCredentials):
		log.Error("Invalid credentials", "email", request.Email)
		c.JSON(http.StatusUnauthorized, requests.ResponseBad("invalid credentials"))

		return
	case err != nil:
		log.Error("Failed get user by email", "error", err, "email", request.Email)
		c.JSON(http.StatusUnauthorized, requests.ResponseBad("invalid credentials"))

		return
	}

	token, refreshToken := createTokens(user.ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"token":        token,
		"refreshToken": refreshToken,
	})
}

func (a *AuthHandlers) HandleRefresh(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	a.bindJSON(c, &request)

	middleware := auth.GetMiddleware(c)

	claims, err := middleware.ParseToken(request.RefreshToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, requests.ResponseErr(err))

		return
	}

	user, err := a.authService.GetUser(c.Request.Context(), claims.UserID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseBad("user specified token not found"),
		)

		return
	}

	token, refreshToken := createTokens(user.ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"refreshToken": refreshToken,
	})
}

func (a *AuthHandlers) HandleStartEmailConfirmation(ctx *gin.Context) {
	session := auth.GetSession(ctx)

	err := a.authService.StartEmailConfirmation(ctx.Request.Context(), session.UserID)
	switch {
	case errors.Is(err, services.ErrEmailAlreadyConfirmed):
		ctx.JSON(http.StatusNotModified, requests.ResponseErr(err))
	case err != nil:
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		ctx.Status(http.StatusOK)
	}
}

func (a *AuthHandlers) HandleConfirmEmail(ctx *gin.Context) {
	session := auth.GetSession(ctx)

	token := ctx.Param("token")

	err := a.authService.ConfirmEmail(ctx.Request.Context(), session.UserID, token)
	switch {
	case errors.Is(err, services.ErrEmailAlreadyConfirmed):
		ctx.JSON(http.StatusNotModified, requests.ResponseErr(err))
	case err != nil:
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		ctx.Status(http.StatusOK)
	}
}

func (a *AuthHandlers) HandleGetUser(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
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
		ctx.AbortWithStatusJSON(http.StatusBadRequest, requests.ResponseErr(err))

		return
	}

	ctx.Status(http.StatusOK)
}

func (a *AuthHandlers) HandleUserSetEmail(ctx *gin.Context) {
	id := requests.MustGetParamUUID(ctx, "id")

	session := auth.GetSession(ctx)
	if session.UserID != id {
		ctx.AbortWithStatusJSON(
			http.StatusForbidden,
			requests.ResponseErr(errors.New("not allowed to set email for this user")),
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
		ctx.AbortWithStatusJSON(http.StatusBadRequest, requests.ResponseErr(err))

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

func createTokens(userID uuid.UUID, email string) (string, string) {
	token := createJWT(userID, email)
	refreshToken := createRefreshJWT(userID)

	return token, refreshToken
}

func newJWT(claims auth.Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	signed, err := token.SignedString(config.SecurityKey())
	if err != nil {
		panic(fmt.Errorf("failed sign JWT: %w", err))
	}

	return signed
}

func createJWT(userID uuid.UUID, email string) string {
	claims := auth.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: &jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		},
	}

	return newJWT(claims)
}

func createRefreshJWT(userID uuid.UUID) string {
	claims := auth.Claims{
		UserID: userID,
		RegisteredClaims: &jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenTTL)),
		},
	}

	return newJWT(claims)
}

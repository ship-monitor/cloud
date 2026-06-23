package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
)

const (
	tokenTTL        = time.Minute * 5
	refreshTokenTTL = time.Hour * 24
)

type registerUserResponse struct {
	User *data.User `json:"user"`
}

func (a *AuthHandlers) HandleRegister(c *gin.Context) {
	var request struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	a.bindJSON(c, &request)

	user, err := data.NewUser(
		request.Name, request.Email, request.Password,
	)
	if err != nil {
		a.logger.Error("Failed create new user", "error", err)

		if errors.Is(err, data.ErrEmailAlreadyTaken) {
			a.logger.Error("Email already taken", "email", request.Email)
			c.AbortWithStatusJSON(http.StatusConflict,
				requests.ResponseBad("email already taken"))

			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.JSON(http.StatusCreated, registerUserResponse{User: user})
}

func (a *AuthHandlers) HandleLogin(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	a.bindJSON(c, &request)

	user, err := data.GetUserByEmail(request.Email)
	if err != nil {
		log.Error("Failed get user by email", "error", err, "email", request.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, requests.ResponseBad("invalid credentials"))

		return
	}

	if !user.CheckPassword(request.Password) {
		log.Error("Invalid password", "email", request.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, requests.ResponseBad("invalid credentials"))

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
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			requests.ResponseErr(fmt.Errorf("invalid refresh token: %w", err)),
		)

		return
	}

	user, err := data.GetUser(c.Request.Context(), claims.UserID)
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

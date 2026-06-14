package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/config"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
)

func mapErrors(vErrors validator.ValidationErrors) []gin.H {
	res := []gin.H{}

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

func bindJSON(ctx *gin.Context, data any) {
	if err := ctx.ShouldBindJSON(data); err != nil {
		if vErrors, ok := err.(validator.ValidationErrors); ok {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"details":          "body validation fails",
				"validationErrors": mapErrors(vErrors),
			})
			// panic("validation fails")
			log.Error("Validation fails", "error", err)
		}
		log.Error("Failed to bind JSON", "error", err)
		ctx.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{"details": "failed get JSON body: " + err.Error()},
		)

		return
	}
}

func HandleRegister(c *gin.Context) {
	var request struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	bindJSON(c, &request)

	user, err := data.NewUser(
		request.Name, request.Email, request.Password,
	)
	if err != nil {
		if errors.Is(err, data.ErrEmailAlreadyTaken) {
			log.Error("Email already taken", "email", request.Email)
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"details": "email already taken",
			})

			return
		}
		log.Error("Failed create new user", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{})

		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func HandleLogin(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	bindJSON(c, &request)

	user, err := data.GetUserByEmail(request.Email)
	if err != nil {
		log.Error("Failed get user by email", "error", err, "email", request.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"details": "invalid credentials"})

		return
	}

	if !user.ComparePassword(request.Password) {
		log.Error("Invalid password", "email", request.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"details": "invalid credentials",
		})

		return
	}

	token, refreshToken := createTokens(user.ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"token":        token,
		"refreshToken": refreshToken,
	})
}

func HandleRefresh(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	bindJSON(c, &request)

	middleware := auth.GetMiddleware(c)

	claims, err := middleware.ParseToken(request.RefreshToken)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			gin.H{"details": "invalid refresh token: " + err.Error()},
		)

		return
	}

	user, err := data.GetUser(claims.UserID)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			gin.H{"details": "user specified in token not found"},
		)

		return
	}

	token, refreshToken := createTokens(user.ID, user.Email)

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"refreshToken": refreshToken,
	})
}

const (
	tokenTTL        = time.Minute * 5
	refreshTokenTTL = time.Hour * 24
)

func createTokens(userID uuid.UUID, email string) (token, refreshToken string) {
	token = createJWT(userID, email)
	refreshToken = createRefreshJWT(userID)

	return token, refreshToken
}

func newJWT(claims auth.Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := token.SignedString(config.SecurityKey())
	if err != nil {
		panic(fmt.Errorf("failed sign JWT: %s", err))
	}

	return signed
}

func createJWT(userID uuid.UUID, email string) (token string) {
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

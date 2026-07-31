package handlers

import (
	"context"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
)

type UserService interface {
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

type UserHandler struct {
	users      UserService
	middleware AuthMiddleware
	logger     *log.Logger
}

type UserResponse struct {
	User *domain.User `json:"user"`
}

var _ pkg.Handler = (*UserHandler)(nil)

func NewUser(
	users UserService,
	middleware AuthMiddleware,
	logger *log.Logger,
) *UserHandler {
	return &UserHandler{
		users:      users,
		middleware: middleware,
		logger:     logger.WithPrefix("UserHandler"),
	}
}

// SetupRoutes implements [pkg.Handler].
func (u *UserHandler) SetupRoutes(router *echo.Group) {
	// u.logger.Info("Here")
	// router.GET(
	// 	"/api/users/me",
	// 	u.middleware.WithAuthenticationRequired,
	// 	u.HandleGetMe,
	// )
}

// func (u *UserHandler) HandleGetMe(c *gin.Context) {
// 	session := auth.GetSession(c)

// 	user, err := u.users.GetUser(c.Request.Context(), session.UserID)
// 	switch {
// 	case err != nil:
// 		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
// 	default:
// 		c.JSON(http.StatusOK, UserResponse{User: user})
// 	}
// }

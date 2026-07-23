package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type DevicesService interface {
	ConnectDevice(
		ctx context.Context,
		deviceID, userID uuid.UUID,
		password, name string,
	) error
	GetStates(
		ctx context.Context,
		deviceID, userID uuid.UUID,
		state string, historyLength int,
	) ([]domain.StateRecord, error)
	SendCommand(
		ctx context.Context,
		deviceID, userID uuid.UUID,
		command string,
		args any,
	) error
}

var _ pkg.Handler = (*DevicesHandlers)(nil)

type DevicesHandlers struct {
	middleware *middleware.AuthMiddleware
	devices    DevicesService
}

func NewDevice(
	devices DevicesService,
	middleware *middleware.AuthMiddleware,
) *DevicesHandlers {
	return &DevicesHandlers{
		devices:    devices,
		middleware: middleware,
	}
}

// SetupRoutes implements [pkg.Handler].
func (d *DevicesHandlers) SetupRoutes(router gin.IRouter) {
	router.POST(
		"/api/v2/devices/connect",
		d.middleware.RequireAuth(),
		d.HandleConnectDevice,
	)
	router.GET(
		"/api/v2/devices/:id/state/:state",
		d.middleware.RequireAuth(),
		d.HandleGetState,
	)
	router.POST(
		"/api/v2/devices/:id/command",
		d.middleware.RequireAuth(),
		d.HandleSendCommand,
	)
}

type ConnectDeviceRequest struct {
	DeviceID uuid.UUID `json:"deviceId" binding:"required"`
	Password string    `json:"password" binding:"required"`
	Name     string    `json:"name" binding:"required"`
}

func (d *DevicesHandlers) HandleConnectDevice(c *gin.Context) {
	principal := middleware.MustPrincipal(c)

	var request ConnectDeviceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			requests.ResponseErr(err),
		)

		return
	}

	err := d.devices.ConnectDevice(
		c.Request.Context(),
		request.DeviceID,
		principal.UserID,
		request.Password,
		request.Name,
	)
	switch {
	case errors.Is(err, services.ErrDeviceNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, requests.ResponseErr(err))
	case errors.Is(err, services.ErrInvalidDevicePassword):
		c.AbortWithStatusJSON(http.StatusForbidden, requests.ResponseErr(err))
	case errors.Is(err, services.ErrAlreadyConnected):
		c.AbortWithStatusJSON(http.StatusConflict, requests.ResponseErr(err))
	case err != nil:
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		c.Status(http.StatusCreated)
	}
}

func (d *DevicesHandlers) HandleGetState(c *gin.Context) {
	deviceID := requests.MustGetParamUUID(c, "id")
	state := c.Param("state")
	session := middleware.MustPrincipal(c)

	historyLength := 0

	q, ok := c.GetQuery("history")
	if ok {
		length, err := strconv.ParseInt(q, 10, 64)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				requests.ResponseErr(
					fmt.Errorf("history query param: %w", err),
				),
			)
		} else {
			historyLength = int(length)
		}
	}

	states, err := d.devices.GetStates(
		c.Request.Context(),
		deviceID,
		session.UserID,
		state,
		historyLength,
	)

	switch {
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		c.JSON(http.StatusOK, gin.H{"result": states})
	}
}

func (d *DevicesHandlers) HandleSendCommand(c *gin.Context) {
	deviceID := requests.MustGetParamUUID(c, "id")
	session := middleware.MustPrincipal(c)

	var req struct {
		Command string         `json:"command" binding:"required"`
		Args    map[string]any `json:"args"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, requests.ResponseErr(err))

		return
	}

	err := d.devices.SendCommand(
		c.Request.Context(),
		deviceID,
		session.UserID,
		req.Command,
		req.Args,
	)

	switch {
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		c.Status(http.StatusOK)
	}
}

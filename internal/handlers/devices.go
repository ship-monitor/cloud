package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
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
	RenameDevice(
		ctx context.Context,
		applicant *domain.Principal,
		deviceID uuid.UUID,
		name string,
	) error
}

var _ pkg.Handler = (*DevicesHandlers)(nil)

type DevicesHandlers struct {
	middleware AuthMiddleware
	devices    DevicesService
}

func NewDevice(
	devices DevicesService,
	middleware AuthMiddleware,
) *DevicesHandlers {
	return &DevicesHandlers{
		devices:    devices,
		middleware: middleware,
	}
}

// SetupRoutes implements [pkg.Handler].
func (d *DevicesHandlers) SetupRoutes(router *echo.Group) {
	devRoutes := router.Group("/api/v2/devices/", d.middleware.RequireAuth())
	devRoutes.POST(
		"/api/v2/devices/connect",
		d.HandleConnectDevice,
	)
	devRoutes.GET(
		"/api/v2/devices/:id/state/:state",
		d.HandleGetState,
	)
	devRoutes.POST(
		"/api/v2/devices/:id/command",
		d.HandleSendCommand,
	)
	devRoutes.PATCH(
		"/api/v2/devices/:id",
		d.HandlePatchDevice,
	)
}

type ConnectDeviceRequest struct {
	DeviceID uuid.UUID `json:"deviceId" binding:"required"`
	Password string    `json:"password" binding:"required"`
	Name     string    `json:"name" binding:"required"`
}

var (
	ErrDeviceNotFound         = errors.New("device not found")
	ErrInvalidDevicePassword  = errors.New("invalid device password")
	ErrDeviceAlreadyConnected = errors.New("device already connected")
)

func (d *DevicesHandlers) HandleConnectDevice(c *echo.Context) error {
	principal := d.middleware.MustPrincipal(c)

	var request ConnectDeviceRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(err),
		)
	}

	err := d.devices.ConnectDevice(
		c.Request().Context(),
		request.DeviceID,
		principal.UserID,
		request.Password,
		request.Name,
	)
	switch {
	case errors.Is(err, ErrDeviceNotFound):
		return c.JSON(http.StatusNotFound, requests.ResponseErr(err))
	case errors.Is(err, ErrInvalidDevicePassword):
		return c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case errors.Is(err, domain.ErrDeviceAlreadyConnected):
		return c.JSON(http.StatusConflict, requests.ResponseErr(err))
	case err != nil:
		return c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		return c.NoContent(http.StatusCreated)
	}
}

func (d *DevicesHandlers) HandleGetState(c *echo.Context) error {
	var request struct {
		DeviceID uuid.UUID `param:"id"`
		State    string    `param:"state"`
		History  int       `query:"history"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	session := d.middleware.MustPrincipal(c)

	states, err := d.devices.GetStates(
		c.Request().Context(),
		request.DeviceID,
		session.UserID,
		request.State,
		request.History,
	)

	switch {
	case err != nil:
		return c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		return c.JSON(http.StatusOK, gin.H{"result": states})
	}
}

func (d *DevicesHandlers) HandleSendCommand(c *echo.Context) error {
	session := d.middleware.MustPrincipal(c)

	var req struct {
		DeviceID uuid.UUID      `param:"id"`
		Command  string         `json:"command" binding:"required"`
		Args     map[string]any `json:"args"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	err := d.devices.SendCommand(
		c.Request().Context(),
		req.DeviceID,
		session.UserID,
		req.Command,
		req.Args,
	)

	switch {
	case errors.Is(err, domain.ErrForbidden):
		return c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		return c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		return c.NoContent(http.StatusOK)
	}
}

func (d *DevicesHandlers) HandlePatchDevice(c *echo.Context) error {
	var request struct {
		Name     string    `json:"name" binding:"required"`
		DeviceID uuid.UUID `param:"id" binding:"required"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	}

	principal := d.middleware.MustPrincipal(c)

	err := d.devices.RenameDevice(
		c.Request().Context(),
		principal,
		request.DeviceID,
		request.Name,
	)

	switch {
	case errors.Is(err, domain.ErrForbidden):
		return c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		return c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		return c.NoContent(http.StatusOK)
	}
}

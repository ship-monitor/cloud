package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type UpdateDeviceRequest struct {
	Name string `binding:"required" json:"name"`
}

type ConnectDeviceRequest struct {
	DeviceID uuid.UUID `binding:"required" json:"deviceId"`
	Name     string    `json:"name"`
}

type SendCommandRequest struct {
	Command string         `binding:"required" json:"command"`
	Args    map[string]any `json:"args,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

type DeviceResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	CreatedAt      time.Time `json:"createdAt"`
	Name           string    `json:"name"`
}

type SendCommandResponse struct {
	RequestError string         `json:"requestError,omitempty"`
	CommandError string         `json:"commandError,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
}

type OrgDevicesService interface {
	ConnectDevice(
		ctx context.Context,
		deviceID, organizationID, userID uuid.UUID,
		name string,
	) error
	DisconnectDevice(ctx context.Context, deviceID, userID uuid.UUID) error
	RenameDevice(
		ctx context.Context,
		deviceID, userID uuid.UUID,
		name string,
	) error
	GetDevice(
		ctx context.Context,
		deviceID, userID uuid.UUID,
	) (*domain.OrganizationDevice, error)
	GetDevices(
		ctx context.Context,
		organizationID, userID uuid.UUID,
	) ([]domain.OrganizationDevice, error)
}

var _ pkg.Handler = (*OrgDevicesHandler)(nil)

type OrgDevicesHandler struct {
	devices    OrgDevicesService
	middleware *middleware.AuthMiddleware
}

func NewOrgDevice(
	devices OrgDevicesService,
	middleware *middleware.AuthMiddleware,
) *OrgDevicesHandler {
	return &OrgDevicesHandler{
		devices:    devices,
		middleware: middleware,
	}
}

// SetupRoutes implements [pkg.Handler].
func (h *OrgDevicesHandler) SetupRoutes(router gin.IRouter) {
	router.Use(h.middleware.RequireAuth())

	router.GET("/api/devices/:id", h.HandleGetDevice)
	router.PATCH("/api/devices/:id", h.HandlePatchDevice)
	router.DELETE("/api/devices/:id", h.HandleDisconnectDevice)

	// Deprecated methods
	router.POST("/api/organizations/:id/devices", h.HandleConnectDevice)
	router.GET("/api/organizations/:id/devices", h.HandleListDevices)
	router.GET("/api/organizations/:id/devices/:deviceId", h.HandleGetDevice)
	router.PATCH(
		"/api/organizations/:id/devices/:deviceId",
		h.HandlePatchDevice,
	)
	router.DELETE(
		"/api/organizations/:id/devices/:deviceId",
		h.HandleDisconnectDevice,
	)
}

func (h *OrgDevicesHandler) HandleGetDevice(c *gin.Context) {
	devID := deviceIDFromRoute(c)
	session := middleware.MustPrincipal(c)

	device, err := h.devices.GetDevice(
		c.Request.Context(),
		devID,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrNotMember):
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		if !ensureDeviceInRouteOrganization(c, device) {
			return
		}

		c.JSON(http.StatusOK, deviceToDTO(device))
	}
}

func (h *OrgDevicesHandler) HandlePatchDevice(c *gin.Context) {
	devID := deviceIDFromRoute(c)
	session := middleware.MustPrincipal(c)

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(fmt.Errorf("bad request: %w", err)),
		)

		return
	}

	if _, ok := organizationIDFromRoute(c); ok {
		device, err := h.devices.GetDevice(
			c.Request.Context(),
			devID,
			session.UserID,
		)
		switch {
		case errors.Is(err, services.ErrNotMember):
			c.JSON(http.StatusForbidden, requests.ResponseErr(err))

			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))

			return
		}

		if !ensureDeviceInRouteOrganization(c, device) {
			return
		}
	}

	err := h.devices.RenameDevice(
		c.Request.Context(),
		devID,
		session.UserID,
		req.Name,
	)
	switch {
	case errors.Is(err, services.ErrEmptyDeviceName):
		c.AbortWithStatusJSON(http.StatusBadRequest, requests.ResponseErr(err))
	case errors.Is(err, services.ErrNotMember):
		c.AbortWithStatusJSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(fmt.Errorf("internal server error: %w", err)),
		)
	default:
		c.Status(http.StatusOK)
	}
}

func (h *OrgDevicesHandler) HandleConnectDevice(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := middleware.MustPrincipal(c)

	var req ConnectDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, requests.ResponseErr(err))

		return
	}

	err := h.devices.ConnectDevice(
		c.Request.Context(),
		req.DeviceID,
		orgID,
		session.UserID,
		req.Name,
	)
	switch {
	case errors.Is(err, services.ErrAlreadyConnected):
		c.AbortWithStatusJSON(http.StatusConflict, requests.ResponseErr(err))
	case errors.Is(err, services.ErrNotMember):
		c.AbortWithStatusJSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		c.Status(http.StatusCreated)
	}
}

func (h *OrgDevicesHandler) HandleListDevices(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := middleware.MustPrincipal(c)

	devices, err := h.devices.GetDevices(
		c.Request.Context(),
		orgID,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrNotMember):
		c.JSON(http.StatusForbidden, requests.ResponseErr((err)))
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		resp := make([]DeviceResponse, 0, len(devices))
		for i := range devices {
			resp = append(resp, deviceToDTO(&devices[i]))
		}

		c.JSON(http.StatusOK, gin.H{"devices": resp})
	}
}

func (h *OrgDevicesHandler) HandleDisconnectDevice(c *gin.Context) {
	devID := deviceIDFromRoute(c)
	session := middleware.MustPrincipal(c)

	if _, ok := organizationIDFromRoute(c); ok {
		device, err := h.devices.GetDevice(
			c.Request.Context(),
			devID,
			session.UserID,
		)
		switch {
		case errors.Is(err, services.ErrNotMember):
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				requests.ResponseErr(err),
			)

			return
		case err != nil:
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				requests.ResponseErr(err),
			)

			return
		}

		if !ensureDeviceInRouteOrganization(c, device) {
			return
		}
	}

	err := h.devices.DisconnectDevice(
		c.Request.Context(),
		devID,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrNotMember):
		c.AbortWithStatusJSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		c.Status(http.StatusOK)
	}
}

func deviceIDFromRoute(c *gin.Context) uuid.UUID {
	if _, ok := c.Params.Get("deviceId"); ok {
		return requests.MustGetParamUUID(c, "deviceId")
	}

	return requests.MustGetParamUUID(c, "id")
}

func organizationIDFromRoute(c *gin.Context) (uuid.UUID, bool) {
	if _, ok := c.Params.Get("deviceId"); !ok {
		return uuid.Nil, false
	}

	return requests.MustGetParamUUID(c, "id"), true
}

func ensureDeviceInRouteOrganization(
	c *gin.Context,
	device *domain.OrganizationDevice,
) bool {
	orgID, ok := organizationIDFromRoute(c)
	if !ok {
		return true
	}

	if device.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, requests.ResponseErr(ErrDeviceNotFound))

		return false
	}

	return true
}

var ErrDeviceNotFound = errors.New("device not found")

func deviceToDTO(d *domain.OrganizationDevice) DeviceResponse {
	return DeviceResponse{
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		Name:           d.Name,
		CreatedAt:      d.CreatedAt,
	}
}

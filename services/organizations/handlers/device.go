package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

func (h *HTTPHandler) HandleGetDevice(c *gin.Context) {
	devID := deviceIDFromRoute(c)
	session := auth.GetSession(c)

	device, err := h.devices.GetDevice(c.Request.Context(), devID, session.UserID)
	switch {
	case errors.Is(err, services.ErrNotMember):
		c.JSON(http.StatusForbidden, dto.Error(err))

		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	if !ensureDeviceInRouteOrganization(c, device) {
		return
	}

	c.JSON(http.StatusOK, deviceToDTO(device))
}

func (h *HTTPHandler) HandlePatchDevice(c *gin.Context) {
	devID := deviceIDFromRoute(c)
	session := auth.GetSession(c)

	var req dto.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(fmt.Errorf("bad request: %w", err)))

		return
	}

	if _, ok := organizationIDFromRoute(c); ok {
		device, err := h.devices.GetDevice(c.Request.Context(), devID, session.UserID)
		switch {
		case errors.Is(err, services.ErrNotMember):
			c.JSON(http.StatusForbidden, dto.Error(err))

			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, dto.Error(err))

			return
		}

		if !ensureDeviceInRouteOrganization(c, device) {
			return
		}
	}

	err := h.devices.RenameDevice(c.Request.Context(), devID, session.UserID, req.Name)
	switch {
	case errors.Is(err, services.ErrEmptyDeviceName):
		c.AbortWithStatusJSON(http.StatusBadRequest, dto.Error(err))
	case errors.Is(err, services.ErrNotMember):
		c.AbortWithStatusJSON(http.StatusForbidden, dto.Error(err))
	case err != nil:
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			dto.Error(fmt.Errorf("internal server error: %w", err)),
		)
	default:
		c.Status(http.StatusOK)
	}
}

func (h *HTTPHandler) HandleConnectDevice(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req dto.ConnectDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))

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
		c.AbortWithStatusJSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		c.Status(http.StatusCreated)
	}
}

func (h *HTTPHandler) HandleListDevices(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	devices, err := h.devices.GetDevices(c.Request.Context(), orgID, session.UserID)
	switch {
	case errors.Is(err, services.ErrNotMember):
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		resp := make([]dto.DeviceResponse, 0, len(devices))
		for i := range devices {
			resp = append(resp, deviceToDTO(&devices[i]))
		}

		c.JSON(http.StatusOK, gin.H{"devices": resp})
	}
}

func (h *HTTPHandler) HandleDisconnectDevice(c *gin.Context) {
	devID := deviceIDFromRoute(c)
	session := auth.GetSession(c)

	if _, ok := organizationIDFromRoute(c); ok {
		device, err := h.devices.GetDevice(c.Request.Context(), devID, session.UserID)
		switch {
		case errors.Is(err, services.ErrNotMember):
			c.AbortWithStatusJSON(http.StatusForbidden, dto.Error(err))

			return
		case err != nil:
			c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Error(err))

			return
		}

		if !ensureDeviceInRouteOrganization(c, device) {
			return
		}
	}

	err := h.devices.DisconnectDevice(c.Request.Context(), devID, session.UserID)
	switch {
	case errors.Is(err, services.ErrNotMember):
		c.AbortWithStatusJSON(http.StatusForbidden, dto.Error(err))
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Error(err))
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

func ensureDeviceInRouteOrganization(c *gin.Context, device *domain.OrganizationDevice) bool {
	orgID, ok := organizationIDFromRoute(c)
	if !ok {
		return true
	}

	if device.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))

		return false
	}

	return true
}

func deviceToDTO(d *domain.OrganizationDevice) dto.DeviceResponse {
	return dto.DeviceResponse{
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		Name:           d.Name,
		CreatedAt:      d.CreatedAt,
	}
}

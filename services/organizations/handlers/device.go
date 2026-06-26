package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/commands"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

func (h *HTTPHandler) HandleGetDevice(c *gin.Context) {
	devID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	device, err := h.devices.GetDevice(c.Request.Context(), devID, session.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, dto.DeviceResponse{
		ID:             device.ID,
		Name:           device.Name,
		CreatedAt:      device.CreatedAt,
		OrganizationID: device.OrganizationID,
	})
}

func (h *HTTPHandler) HandlePatchDevice(c *gin.Context) {
	devID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req dto.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(fmt.Errorf("bad request: %w", err)))

		return
	}

	err := h.devices.RenameDevice(c.Request.Context(), devID, session.UserID, req.Name)

	switch {
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

// HandlePatchDevice
//
// Deprecated: use [HTTPHandler.HandlePatchDevice].
func HandlePatchDevice(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdministrator {
		c.JSON(
			http.StatusForbidden,
			dto.Error(errors.New("only owner or administrator can connect devices")),
		)

		return
	}

	var req dto.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))

		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	device, err := data.GetDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("no device")))

		return
	}

	if err := data.SetName(c.Request.Context(), device, req.Name); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			dto.Error(fmt.Errorf("failed set device name: %w", err)),
		)

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device": device,
	})
}

func (h *HTTPHandler) HandleConnectDevice(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req dto.ConnectDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))

		return
	}

	err := h.devices.ConnectDevice(c.Request.Context(), req.DeviceID, orgID, session.UserID)
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

func HandleListDevices(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	if !isMember {
		log.Warn("Access denied for list devices", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	devices, err := data.ListDevices(c.Request.Context(), orgID)
	if err != nil {
		log.Error(
			"Failed to list devices",
			"error",
			err,
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	resp := make([]dto.DeviceResponse, 0, len(devices))
	for i := range devices {
		resp = append(resp, deviceToDTO(&devices[i]))
	}

	c.JSON(http.StatusOK, gin.H{"devices": resp})
}

// HandleGetDevice
//
// Deprecated: use [HTTPHandler.HandleGetDevice].
func HandleGetDevice(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("Invalid organization id in get device request", "id", c.Param("id"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		log.Warn("Invalid device id in get device request", "deviceId", c.Param("deviceId"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid device id")))

		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil {
		log.Error(
			"Failed to check membership for get device",
			"error",
			err,
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	if !isMember {
		log.Warn("Access denied for get device", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	device, err := data.GetDeviceDTO(c.Request.Context(), deviceID)
	if err != nil || device.OrganizationID != orgID {
		log.Warn(
			"Device not found",
			"organization",
			orgID,
			"device",
			deviceID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))

		return
	}

	c.JSON(http.StatusOK, device)
}

func (h *HTTPHandler) HandleDisconnectDevice(ctx *gin.Context) {
	devID := requests.MustGetParamUUID(ctx, "id")
	session := auth.GetSession(ctx)

	err := h.devices.DisconnectDevice(ctx.Request.Context(), devID, session.UserID)
	switch {
	case errors.Is(err, services.ErrNotMember):
		ctx.AbortWithStatusJSON(http.StatusForbidden, dto.Error(err))
	case err != nil:
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, dto.Error(err))
	default:
		ctx.Status(http.StatusOK)
	}
}

// HandleDisconnect Device
//
// Deprecated: use [HTTPHandler.HandleDisconnectDevice].
func HandleDisconnectDevice(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	deviceID := requests.MustGetParamUUID(c, "deviceId")

	session := auth.GetSession(c)

	_, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		log.Warn(
			"Access denied for disconnect device — not a member",
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	device, err := data.GetDeviceDTO(c.Request.Context(), deviceID)
	if err != nil || device.OrganizationID != orgID {
		log.Warn(
			"Device not found for disconnect",
			"organization",
			orgID,
			"device",
			deviceID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))

		return
	}

	if err := data.DisconnectDevice(c.Request.Context(), deviceID); err != nil {
		log.Error(
			"Failed to disconnect device",
			"error",
			err,
			"organization",
			orgID,
			"device",
			deviceID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.Status(http.StatusOK)
}

func (h *HTTPHandler) HandleSendCommand(c *gin.Context) {
	deviceID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req dto.SendCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(fmt.Errorf("invalid request: %w", err)))

		return
	}

	resp, err := h.devices.SendCommand(
		c.Request.Context(),
		deviceID,
		session.UserID,
		req.Command,
		req.Args,
	)

	switch {
	case errors.Is(err, services.ErrNotMember):
		c.AbortWithStatusJSON(http.StatusForbidden, dto.Error(err))
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Error(err))
	case resp.RequestError != "" || resp.CommandError != "":
		c.AbortWithStatusJSON(http.StatusBadRequest, resp)
	default:
		c.JSON(http.StatusOK, resp)
	}
}

// HandleSendCommand
//
// Deprecated: use [HTTPHandler.HandleSendCommand].
func HandleSendCommand(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	deviceID := requests.MustGetParamUUID(c, "deviceId")
	session := auth.GetSession(c)

	if _, err := data.GetMember(orgID, session.UserID); err != nil {
		c.JSON(
			http.StatusForbidden,
			dto.Error(errors.New("access denied: not a member of organization")),
		)

		return
	}

	device, err := data.GetDeviceDTO(c.Request.Context(), deviceID)
	if err != nil || device.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))

		return
	}

	var req dto.SendCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(fmt.Errorf("invalid request: %w", err)))

		return
	}

	cmd := commands.NewCommand(deviceID.String(), req.Command, req.Args)
	result := commands.SendCommand(c.Request.Context(), cmd)

	if result.RequestError != "" {
		c.JSON(http.StatusBadGateway, dto.SendCommandResponse{
			RequestError: result.RequestError,
		})

		return
	}

	c.JSON(http.StatusOK, dto.SendCommandResponse{
		CommandError: result.CommandError,
		Data:         result.Data,
	})
}

func deviceToDTO(d *domain.OrganizationDevice) dto.DeviceResponse {
	return dto.DeviceResponse{
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		Name:           d.Name,
		CreatedAt:      d.CreatedAt,
	}
}

package organizations

import (
	"errors"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/commands"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

func HandleConnectDevice(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("Invalid organization id in connect device request", "id", c.Param("id"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		log.Warn("Access denied for connect device — not a member", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		log.Warn("Insufficient role for connect device", "organization", orgID, "user", session.UserID, "role", member.Role)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can connect devices")))
		return
	}

	var req dto.ConnectDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid connect device request", "error", err, "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))
		return
	}

	if err := data.ConnectDevice(req.DeviceID, orgID); err != nil {
		log.Warn("Failed to connect device", "error", err, "organization", orgID, "device", req.DeviceID)
		c.JSON(http.StatusBadRequest, dto.Error(err))
		return
	}

	device, err := data.GetDevice(req.DeviceID)
	if err != nil {
		log.Error("Failed to get device after connect", "error", err, "organization", orgID, "device", req.DeviceID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusCreated, deviceToDTO(device))
}

func HandleListDevices(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("Invalid organization id in list devices request", "id", c.Param("id"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil {
		log.Error("Failed to check membership for list devices", "error", err, "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}
	if !isMember {
		log.Warn("Access denied for list devices", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}

	devices, err := data.ListDevices(orgID)
	if err != nil {
		log.Error("Failed to list devices", "error", err, "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	resp := make([]dto.DeviceResponse, 0, len(devices))
	for i := range devices {
		resp = append(resp, deviceToDTO(&devices[i]))
	}

	c.JSON(http.StatusOK, gin.H{"devices": resp})
}

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
		log.Error("Failed to check membership for get device", "error", err, "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}
	if !isMember {
		log.Warn("Access denied for get device", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}

	device, err := data.GetDevice(deviceID)
	if err != nil || device.OrganizationID != orgID {
		log.Warn("Device not found", "organization", orgID, "device", deviceID, "user", session.UserID)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))
		return
	}

	c.JSON(http.StatusOK, deviceToDTO(device))
}

func HandleDisconnectDevice(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("Invalid organization id in disconnect device request", "id", c.Param("id"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		log.Warn("Invalid device id in disconnect device request", "deviceId", c.Param("deviceId"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid device id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		log.Warn("Access denied for disconnect device — not a member", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		log.Warn("Insufficient role for disconnect device", "organization", orgID, "user", session.UserID, "role", member.Role)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can disconnect devices")))
		return
	}

	device, err := data.GetDevice(deviceID)
	if err != nil || device.OrganizationID != orgID {
		log.Warn("Device not found for disconnect", "organization", orgID, "device", deviceID, "user", session.UserID)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))
		return
	}

	if err := data.DisconnectDevice(deviceID); err != nil {
		log.Error("Failed to disconnect device", "error", err, "organization", orgID, "device", deviceID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandleSendCommand(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("Invalid organization id in send command request", "id", c.Param("id"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		log.Warn("Invalid device id in send command request", "deviceId", c.Param("deviceId"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid device id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		log.Warn("Access denied for send command — not a member", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		log.Warn("Insufficient role for send command", "organization", orgID, "user", session.UserID, "role", member.Role)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can send commands")))
		return
	}

	device, err := data.GetDevice(deviceID)
	if err != nil || device.OrganizationID != orgID {
		log.Warn("Device not found for send command", "organization", orgID, "device", deviceID, "user", session.UserID)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))
		return
	}

	var req dto.SendCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid send command request", "error", err, "organization", orgID, "device", deviceID)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))
		return
	}

	result := commands.SendCommand(deviceID, req.Command, req.Args)

	if result.RequestError != "" {
		log.Warn("Command delivery failed", "organization", orgID, "device", deviceID, "command", req.Command, "error", result.RequestError)
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

func deviceToDTO(d *data.OrganizationDevice) dto.DeviceResponse {
	return dto.DeviceResponse{
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		CreatedAt:      d.CreatedAt,
	}
}

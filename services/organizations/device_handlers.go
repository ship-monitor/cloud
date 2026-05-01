package organizations

import (
	"errors"
	"net/http"

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
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can connect devices")))
		return
	}

	var req dto.ConnectDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))
		return
	}

	if err := data.ConnectDevice(req.DeviceID, orgID); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(err))
		return
	}

	device, err := data.GetDevice(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusCreated, deviceToDTO(device))
}

func HandleListDevices(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}

	devices, err := data.ListDevices(orgID)
	if err != nil {
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
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid device id")))
		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}

	device, err := data.GetDevice(deviceID)
	if err != nil || device.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))
		return
	}

	c.JSON(http.StatusOK, deviceToDTO(device))
}

func HandleDisconnectDevice(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid device id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can disconnect devices")))
		return
	}

	device, err := data.GetDevice(deviceID)
	if err != nil || device.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))
		return
	}

	if err := data.DisconnectDevice(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandleSendCommand(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid device id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can send commands")))
		return
	}

	device, err := data.GetDevice(deviceID)
	if err != nil || device.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("device not found")))
		return
	}

	var req dto.SendCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))
		return
	}

	result := commands.SendCommand(deviceID, req.Command, req.Args)

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

func deviceToDTO(d *data.OrganizationDevice) dto.DeviceResponse {
	return dto.DeviceResponse{
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		CreatedAt:      d.CreatedAt,
	}
}

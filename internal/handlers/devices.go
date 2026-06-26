package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type DevicesService interface {
	GetStates(
		ctx context.Context,
		deviceID, userID uuid.UUID,
		state string, historyLength int,
	) ([]domain.StateRecord, error)
}

type DevicesHandlers struct {
	devices DevicesService
}

func NewDevicesHandlers(devices DevicesService) *DevicesHandlers {
	return &DevicesHandlers{
		devices: devices,
	}
}

func (d *DevicesHandlers) HandleGetState(c *gin.Context) {
	deviceID := requests.MustGetParamUUID(c, "id")
	state := c.Param("state")
	session := auth.GetSession(c)

	historyLength := 0

	q, ok := c.GetQuery("history")
	if ok {
		length, err := strconv.ParseInt(q, 10, 64)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				requests.ResponseErr(fmt.Errorf("history query param: %w", err)),
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

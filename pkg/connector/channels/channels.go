package channels

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	healthChannelFormat          = "device.%s.health"
	connectRequestChannelFormat  = "device.%s.connect.request"
	connectResponseChannelFormat = "device.%s.connect.response"
)

// HealthChannelW is wildcard channel for heartbeat messages.
func HealthChannelW() string {
	return fmt.Sprintf(healthChannelFormat, "*")
}

// HealthChannel builds channel string for specified deviceID.
func HealthChannel(deviceID uuid.UUID) string {
	return fmt.Sprintf(healthChannelFormat, deviceID.String())
}

// ConnectRequestChannelW is wildcard channel for connect requests.
//	func ConnectRequestChannelW() string {
//		return fmt.Sprintf(connectRequestChannelFormat, "*")
//	}

// ConnectRequestChannel builds channel string for specified deviceID.
func ConnectRequestChannel(deviceID uuid.UUID) string {
	return fmt.Sprintf(connectRequestChannelFormat, deviceID.String())
}

// ConnectResponseChannelW is wildcard channel for connect responses.
func ConnectResponseChannelW() string {
	return fmt.Sprintf(connectResponseChannelFormat, "*")
}

// ConnectResponseChannel builds channel string for specified deviceID.
func ConnectResponseChannel(deviceID uuid.UUID) string {
	return fmt.Sprintf(connectResponseChannelFormat, deviceID.String())
}

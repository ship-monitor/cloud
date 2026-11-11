package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
)

type TokenService interface {
	ParseToken(ctx context.Context, token string) (*auth.Claims, error)
	CreateTokens(ctx context.Context, user *user.User) (string, string, error)
	RefreshTokens(
		ctx context.Context,
		refreshToken string,
	) (string, string, error)
}

type UserService interface {
	Register(
		ctx context.Context,
		email, password, name string,
	) (user.User, error)
	Login(ctx context.Context, email, password string) (user.User, error)
}

// DeviceService methods are executed on behalf of the given actor;
// devices of other users are reported as not found.
type DeviceService interface {
	Get(
		ctx context.Context,
		actor auth.Actor,
		id uuid.UUID,
	) (devicedomain.Device, error)
	ListByOwner(
		ctx context.Context,
		actor auth.Actor,
		paging devicedomain.Paging,
	) ([]devicedomain.Device, error)
	Connect(
		ctx context.Context,
		actor auth.Actor,
		id uuid.UUID,
		securityKey string,
	) error
	Remove(ctx context.Context, actor auth.Actor, id uuid.UUID) error
	Rename(
		ctx context.Context,
		actor auth.Actor,
		id uuid.UUID,
		name string,
	) error
	IsDeviceOnline(
		ctx context.Context,
		actor auth.Actor,
		id uuid.UUID,
	) (online bool, lastSeen time.Time, err error)
}

type SensorService interface {
	GetSensors(
		ctx context.Context,
		actor auth.Actor,
		deviceID uuid.UUID,
	) ([]sensor.Sensor, error)
	GetSensor(
		ctx context.Context,
		actor auth.Actor,
		sensorID string,
		deviceID uuid.UUID,
	) (*sensor.Sensor, error)
	Rename(
		ctx context.Context,
		actor auth.Actor,
		id string,
		deviceID uuid.UUID,
		displayName string,
	) error
}

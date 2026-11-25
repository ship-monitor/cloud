package di

import (
	"fmt"
	"time"

	"github.com/ship-monitor/cloud/internal/v2/adapters/publisher"
	"github.com/ship-monitor/cloud/internal/v2/config"
	"github.com/ship-monitor/cloud/internal/v2/repositories/alive"
	"github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	devicemetricsrepo "github.com/ship-monitor/cloud/internal/v2/repositories/devicemetrics"
	notificationrepo "github.com/ship-monitor/cloud/internal/v2/repositories/notification"
	sensorrepo "github.com/ship-monitor/cloud/internal/v2/repositories/sensor"
	sensormetricsrepo "github.com/ship-monitor/cloud/internal/v2/repositories/sensormetrics"
	userrepo "github.com/ship-monitor/cloud/internal/v2/repositories/user"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	metricsservice "github.com/ship-monitor/cloud/internal/v2/services/metrics"
	"github.com/ship-monitor/cloud/internal/v2/services/notification"
	"github.com/ship-monitor/cloud/internal/v2/services/sensor"
	"github.com/ship-monitor/cloud/internal/v2/services/token"
	"github.com/ship-monitor/cloud/internal/v2/services/user"
	"go.uber.org/zap"
)

func newTokenSerice(config *config.Config) (*token.Service, error) {
	tokenTTL := time.Minute
	refreshTTL := time.Hour * 48 //nolint:mnd

	if config.IsDevelopment() {
		tokenTTL = refreshTTL
	}

	service, err := token.New(
		&token.Config{
			SecretKey:       config.SecurityKey,
			TokenTTL:        tokenTTL,
			RefreshTokenTTL: refreshTTL,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("new token service: %w", err)
	}

	return service, nil
}

func newUserService(repo *userrepo.Repository) *user.Service {
	return user.New(repo)
}

func newDeviceService(
	repo *devicerepo.Repository,
	alive *alive.Repo,
	pub *publisher.P,
	conn *connect.Repo,
) *device.Service {
	return device.New(repo, alive, pub, conn)
}

func newSensorsService(
	repo *sensorrepo.Repo,
	devices *device.Service,
	logger *zap.Logger,
) *sensor.Service {
	return sensor.New(repo, devices, logger)
}

func newMetricsService(
	devices *device.Service,
	deviceMetrics *devicemetricsrepo.Repository,
	sensorMetrics *sensormetricsrepo.Repository,
	logger *zap.Logger,
) *metricsservice.Service {
	return metricsservice.New(
		devices,
		deviceMetrics,
		sensorMetrics,
		logger,
	)
}

func newNotificationService(
	repo *notificationrepo.Repository,
	devices *device.Service,
) *notification.Service {
	return notification.New(repo, devices)
}

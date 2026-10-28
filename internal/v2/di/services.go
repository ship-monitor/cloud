package di

import (
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/config"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	userrepo "github.com/ship-monitor/cloud/internal/v2/repositories/user"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/ship-monitor/cloud/internal/v2/services/token"
	"github.com/ship-monitor/cloud/internal/v2/services/user"
)

func newTokenSerice(config *config.Config) (*token.Service, error) {
	s, err := token.New(&token.Config{SecretKey: config.SecurityKey})
	if err != nil {
		return nil, fmt.Errorf("new token service: %w", err)
	}

	return s, nil
}

func newUserService(repo *userrepo.Repository) *user.Service {
	return user.New(repo)
}

func newDeviceService(repo *devicerepo.Repository) *device.Service {
	return device.New(repo)
}

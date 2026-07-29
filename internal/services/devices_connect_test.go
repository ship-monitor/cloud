package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
)

type deviceRepositoryStub struct {
	device       *domain.Device
	getErr       error
	connectErr   error
	connectCalls int
	connectedTo  uuid.UUID
	connectedAs  string
}

func (s *deviceRepositoryStub) GetDevice(
	context.Context,
	domain.DeviceID,
) (*domain.Device, error) {
	return s.device, s.getErr
}

func (s *deviceRepositoryStub) ConnectDevice(
	_ context.Context,
	_ domain.DeviceID,
	userID uuid.UUID,
	name string,
) (*domain.Device, error) {
	s.connectCalls++
	s.connectedTo = userID
	s.connectedAs = name

	return s.device, s.connectErr
}

func (s *deviceRepositoryStub) RenameDevice(
	_ context.Context,
	_ domain.DeviceID,
	_ string,
) error {
	return nil
}

func TestConnectDevice(t *testing.T) {
	t.Parallel()

	deviceID := uuid.New()
	userID := uuid.New()
	password := "secret"

	tests := []struct {
		name       string
		repository *deviceRepositoryStub
		password   string
		wantErr    error
		wantCalls  int
	}{
		{
			name: "connected",
			repository: &deviceRepositoryStub{device: &domain.Device{
				ID:           deviceID,
				PasswordHash: domain.HashPassword(password),
			}},
			password:  password,
			wantCalls: 1,
		},
		{
			name:       "not found",
			repository: &deviceRepositoryStub{getErr: sql.ErrNoRows},
			password:   password,
			wantErr:    services.ErrDeviceNotFound,
		},
		{
			name: "invalid password",
			repository: &deviceRepositoryStub{device: &domain.Device{
				ID:           deviceID,
				PasswordHash: domain.HashPassword(password),
			}},
			password: "wrong",
			wantErr:  services.ErrInvalidDevicePassword,
		},
		{
			name: "already connected",
			repository: &deviceRepositoryStub{device: &domain.Device{
				ID:           deviceID,
				PasswordHash: domain.HashPassword(password),
				OwnerID:      new(uuid.UUID),
			}},
			password: password,
			wantErr:  domain.ErrDeviceAlreadyConnected,
		},
		{
			name: "concurrent connection",
			repository: &deviceRepositoryStub{
				device: &domain.Device{
					ID:           deviceID,
					PasswordHash: domain.HashPassword(password),
				},
				connectErr: domain.ErrDeviceAlreadyConnected,
			},
			password:  password,
			wantErr:   domain.ErrDeviceAlreadyConnected,
			wantCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := services.NewDevices(nil, test.repository, nil, nil, nil)
			err := service.ConnectDevice(
				t.Context(),
				deviceID,
				userID,
				test.password,
				"",
			)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("error = %v, want %v", err, test.wantErr)
			}

			if test.repository.connectCalls != test.wantCalls {
				t.Errorf(
					"connect calls = %d, want %d",
					test.repository.connectCalls,
					test.wantCalls,
				)
			}

			if test.wantErr == nil {
				if test.repository.connectedTo != userID {
					t.Errorf(
						"connected user = %s, want %s",
						test.repository.connectedTo,
						userID,
					)
				}

				if test.repository.connectedAs == "" {
					t.Error("generated device name is empty")
				}
			}
		})
	}
}

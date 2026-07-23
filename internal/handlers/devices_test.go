package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
)

type devicesServiceStub struct {
	connect func(
		ctx context.Context,
		deviceID, userID uuid.UUID,
		password, name string,
	) error
}

func (s *devicesServiceStub) ConnectDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	password, name string,
) error {
	return s.connect(ctx, deviceID, userID, password, name)
}

func (*devicesServiceStub) GetStates(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	string,
	int,
) ([]domain.StateRecord, error) {
	panic("unexpected GetStates call")
}

func (*devicesServiceStub) SendCommand(
	context.Context,
	uuid.UUID,
	uuid.UUID,
	string,
	any,
) error {
	panic("unexpected SendCommand call")
}

func TestHandleConnectDevice(t *testing.T) {
	t.Parallel()

	deviceID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantCall   bool
	}{
		{
			name:       "connected",
			body:       `{"deviceId":"` + deviceID.String() + `","password":"secret","name":"Bridge"}`,
			wantStatus: http.StatusCreated,
			wantCall:   true,
		},
		{
			name:       "invalid request",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "device not found",
			body:       `{"deviceId":"` + deviceID.String() + `","password":"secret"}`,
			serviceErr: services.ErrDeviceNotFound,
			wantStatus: http.StatusNotFound,
			wantCall:   true,
		},
		{
			name:       "invalid password",
			body:       `{"deviceId":"` + deviceID.String() + `","password":"wrong"}`,
			serviceErr: services.ErrInvalidDevicePassword,
			wantStatus: http.StatusForbidden,
			wantCall:   true,
		},
		{
			name:       "already connected",
			body:       `{"deviceId":"` + deviceID.String() + `","password":"secret"}`,
			serviceErr: services.ErrAlreadyConnected,
			wantStatus: http.StatusConflict,
			wantCall:   true,
		},
		{
			name:       "service failure",
			body:       `{"deviceId":"` + deviceID.String() + `","password":"secret"}`,
			serviceErr: services.ErrForbidden,
			wantStatus: http.StatusInternalServerError,
			wantCall:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			called := false
			service := &devicesServiceStub{
				connect: func(
					_ context.Context,
					gotDeviceID, gotUserID uuid.UUID,
					password, name string,
				) error {
					called = true

					if gotDeviceID != deviceID {
						t.Errorf(
							"device ID = %s, want %s",
							gotDeviceID,
							deviceID,
						)
					}

					if gotUserID != userID {
						t.Errorf("user ID = %s, want %s", gotUserID, userID)
					}

					if test.name == "connected" &&
						(password != "secret" || name != "Bridge") {
						t.Errorf(
							"credentials = %q, %q, want secret, Bridge",
							password,
							name,
						)
					}

					return test.serviceErr
				},
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				"/api/v2/devices",
				strings.NewReader(test.body),
			)
			ctx.Request.Header.Set("Content-Type", "application/json")
			middleware.AddToContext(ctx, &domain.Principal{UserID: userID})

			handlers.NewDevice(service, nil).HandleConnectDevice(ctx)

			if ctx.Writer.Status() != test.wantStatus {
				t.Errorf(
					"status = %d, want %d",
					ctx.Writer.Status(),
					test.wantStatus,
				)
			}

			if called != test.wantCall {
				t.Errorf("service called = %t, want %t", called, test.wantCall)
			}
		})
	}
}

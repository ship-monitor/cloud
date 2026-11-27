package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/handlers/queue"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type connectDeviceRepo struct {
	deviceservice.Repository

	created []domain.Device
}

func (r *connectDeviceRepo) Create(_ context.Context, d domain.Device) error {
	r.created = append(r.created, d)

	return nil
}

type connectRequestRepo struct {
	deviceservice.ConnRequestRepository

	request domain.ConnectRequest
}

func (r *connectRequestRepo) Get(
	_ context.Context,
	id uuid.UUID,
) (domain.ConnectRequest, error) {
	if id != r.request.ID {
		return domain.ConnectRequest{}, device.ErrNotFound
	}

	return r.request, nil
}

func (r *connectRequestRepo) SetApproved(
	_ context.Context,
	id uuid.UUID,
	approved bool,
) error {
	if id != r.request.ID {
		return device.ErrNotFound
	}

	r.request.Approved = approved

	return nil
}

func TestHandler_HandleConnectResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		approve    bool
		responder  string
		wantCreate bool
	}{
		{
			name: "approved by requested device", approve: true,
			responder: "matching", wantCreate: true,
		},
		{name: "rejected by requested device", responder: "matching"},
		{name: "approval without identity", approve: true},
		{
			name: "approval by another device", approve: true,
			responder: "different",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := domain.ConnectRequest{
				ID: uuid.New(), DeviceID: uuid.New(), UserID: uuid.New(),
				CreatedAt: time.Now(),
			}
			repo := &connectDeviceRepo{}
			requests := &connectRequestRepo{request: request}
			service := deviceservice.New(repo, nil, nil, requests)
			handler := queue.New(nil, service, zap.NewNop())
			ctx := t.Context()

			switch tt.responder {
			case "matching":
				ctx = domain.WithDeviceID(ctx, request.DeviceID)
			case "different":
				ctx = domain.WithDeviceID(ctx, uuid.New())
			}

			err := handler.HandleConnectResponse(ctx,
				&asyncapi.ConnectResponseMessage{
					Payload: asyncapi.ConnectResponseSchema{
						Approve: tt.approve, RequestID: request.ID,
					},
				},
			)
			if tt.responder == "matching" {
				require.NoError(t, err)
			}

			if tt.wantCreate {
				require.Len(t, repo.created, 1)
				assert.Equal(t, request.DeviceID, repo.created[0].ID)
				assert.Equal(t, request.UserID, repo.created[0].OwnerID)
				assert.True(t, requests.request.Approved)
			} else {
				assert.Empty(t, repo.created)
				assert.False(t, requests.request.Approved)
			}
		})
	}
}

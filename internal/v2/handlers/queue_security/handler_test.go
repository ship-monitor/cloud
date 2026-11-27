package queuesecurity_test

import (
	"context"
	"testing"

	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	queuesecurity "github.com/ship-monitor/cloud/internal/v2/handlers/queue_security"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"go.uber.org/zap"
)

func TestHandler_HandleMessageRejectsUnauthenticatedIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		authorization string
	}{
		{
			name:          "caller-controlled device UUID",
			authorization: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:          "zero UUID",
			authorization: "00000000-0000-0000-0000-000000000000",
		},
		{name: "empty authorization", authorization: ""},
		{name: "whitespace authorization", authorization: "   "},
		{name: "invalid UUID", authorization: "not-a-uuid"},
		{
			name:          "bearer prefix without verified credential",
			authorization: "Bearer 550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := queuesecurity.New(zap.NewNop())

			ctx, err := handler.HandleMessage(
				context.Background(),
				asyncapi.IdentificationHeadersSchema{
					Authorization: tt.authorization,
				},
			)
			if err == nil {
				t.Error("unauthenticated identity was accepted; want an error")
			}

			if ctx != nil {
				if id, ok := device.DeviceIDFromContext(ctx); ok {
					t.Errorf(
						"unauthenticated device ID %s was added to context",
						id,
					)
				}
			}
		})
	}
}

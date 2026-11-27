package subscriber_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type declarationSecurity struct {
	deviceID uuid.UUID
}

func (s declarationSecurity) HandleMessage(
	ctx context.Context,
	_ asyncapi.IdentificationHeadersSchema,
) (context.Context, error) {
	return device.WithDeviceID(ctx, s.deviceID), nil
}

type declarationRecorder struct {
	calls    int
	deviceID uuid.UUID
	sensors  []sensor.SensorDecl
}

func (r *declarationRecorder) DeclareSensors(
	_ context.Context,
	deviceID uuid.UUID,
	sensors []sensor.SensorDecl,
) error {
	r.calls++
	r.deviceID = deviceID
	r.sensors = sensors

	return nil
}

//nolint:exhaustruct_v5
func TestHandleSensorDeclaration_RequiresSensorsArray(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload string
		valid   bool
	}{
		{name: "missing", payload: `{}`, valid: false},
		{name: "null", payload: `{"sensors":null}`, valid: false},
		{name: "empty array", payload: `{"sensors":[]}`, valid: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var msg asyncapi.SensorDeclarationMessage
			require.NoError(
				t,
				json.Unmarshal([]byte(testCase.payload), &msg.Payload),
			)

			deviceID := uuid.New()
			recorder := &declarationRecorder{}
			sub := subscriber.New(
				nil, zap.NewNop(), nil, declarationSecurity{deviceID: deviceID},
				recorder, nil, nil, nil,
			)

			err := sub.HandleSensorDeclaration(t.Context(), msg)
			if !testCase.valid {
				require.Zero(
					t,
					recorder.calls,
					"malformed declaration must not sync",
				)
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, 1, recorder.calls)
			require.Equal(t, deviceID, recorder.deviceID)
			require.NotNil(t, recorder.sensors)
			require.Empty(t, recorder.sensors)
		})
	}
}

//nolint:tagliatelle
package messages

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/pkg/headers"
)

type (
	Heartbeat struct {
		Payload HeartbeatPayload `validate:"required"`
		Headers HeartbeatHeaders `validate:"required"`
	}
	HeartbeatPayload struct {
		Timestamp int64 `json:"timestamp" validate:"required,min=0"`
	}
	HeartbeatHeaders struct {
		DeviceID uuid.UUID `header:"device-id" validate:"required"`
	}
)

func UnmarshallHeartbeat(msg *nats.Msg) (*Heartbeat, error) {
	var payload HeartbeatPayload

	err := json.Unmarshal(msg.Data, &payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshall payload: %w", err)
	}

	var header HeartbeatHeaders

	err = headers.NewDecoder(&msg.Header).Decode(&header)
	if err != nil {
		return nil, fmt.Errorf("decode headers: %w", err)
	}

	heartbeat := Heartbeat{
		Payload: payload,
		Headers: header,
	}

	err = validator.New().Struct(&heartbeat)
	if err != nil {
		return nil, fmt.Errorf("invalid message: %w", err)
	}

	return &heartbeat, nil
}

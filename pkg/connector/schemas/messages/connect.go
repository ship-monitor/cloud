package messages

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/pkg/headers"
)

type ConnectRequest struct {
	DeviceID    uuid.UUID `json:"deviceId" validate:"required"`
	RequestID   uuid.UUID `json:"requestId" validate:"required"`
	SecurityKey string    `json:"securityKey" validate:"required"`
}

func UnmarshalConnectRequest(msg *nats.Msg) (*ConnectRequest, error) {
	var request ConnectRequest

	err := json.Unmarshal(msg.Data, &request)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	err = validator.New().Struct(&request)
	if err != nil {
		return nil, fmt.Errorf("invalid message: %w", err)
	}

	return &request, nil
}

type (
	ConnectResponse struct {
		Payload ConnectResponsePayload `validate:"required"`
		Header  ConnectResponseHeader  `validate:"required"`
	}
	ConnectResponsePayload struct {
		Approve   bool      `json:"allow" validate:"required"`
		RequestID uuid.UUID `json:"requestId" validate:"required"`
	}
	ConnectResponseHeader struct {
		IdentificationHeader
	}
)

func UnmarshalConnectResponse(msg *nats.Msg) (*ConnectResponse, error) {
	var payload ConnectResponsePayload

	err := json.Unmarshal(msg.Data, &payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshall payload: %w", err)
	}

	var header ConnectResponseHeader

	err = headers.NewDecoder(&msg.Header).Decode(&header)
	if err != nil {
		return nil, fmt.Errorf("decode headers: %w", err)
	}

	response := ConnectResponse{
		Payload: payload,
		Header:  header,
	}

	err = validator.New().Struct(&response)
	if err != nil {
		return nil, fmt.Errorf("invalid message: %w", err)
	}

	return &response, nil
}

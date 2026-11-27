package regression

import (
	"context"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
)

type ConnectRepo struct {
	deviceservice.Repository
}

//nolint:exhaustruct_v5
func NewConnectRepo() *ConnectRepo {
	return &ConnectRepo{}
}

func (ConnectRepo) Exists(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

type ConnectRequestsRepo struct {
	deviceservice.ConnRequestRepository

	request device.ConnectRequest
}

//nolint:exhaustruct_v5
func NewConnectRequestsRepo() *ConnectRequestsRepo {
	return &ConnectRequestsRepo{}
}

func (r *ConnectRequestsRepo) Request() device.ConnectRequest {
	return r.request
}

func (r *ConnectRequestsRepo) Create(
	_ context.Context,
	request device.ConnectRequest,
) error {
	r.request = request

	return nil
}

type Publisher struct {
	deviceservice.Publisher

	called   bool
	deviceID uuid.UUID
	message  *asyncapi.ConnectRequestMessage
}

//nolint:exhaustruct_v5
func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Called() bool {
	return p.called
}

func (p *Publisher) Message() *asyncapi.ConnectRequestMessage {
	return p.message
}

func (p *Publisher) DeviceID() uuid.UUID {
	return p.deviceID
}

func (p *Publisher) PublishConnectRequest(
	_ context.Context,
	deviceID uuid.UUID,
	message *asyncapi.ConnectRequestMessage,
) error {
	p.called = true
	p.deviceID = deviceID
	p.message = message

	return nil
}

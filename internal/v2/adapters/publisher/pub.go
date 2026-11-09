package publisher

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
)

type P struct {
	controller *asyncapi.AppController
}

func New(
	controller *asyncapi.AppController,
) *P {
	return &P{
		controller: controller,
	}
}

func (p *P) PublishConnectRequest(
	ctx context.Context,
	id uuid.UUID,
	request *asyncapi.ConnectRequestMessage,
) error {
	err := p.controller.SendAsPublishConnectRequestOperation(
		ctx,
		asyncapi.ConnectRequestsChannelParameters{
			DeviceId: id.String(),
		},
		*request,
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

package publisher

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/pkg/connector/channels"
	"github.com/ship-monitor/cloud/pkg/connector/messages"
	"github.com/ship-monitor/cloud/pkg/connector/parameters"
	natsGen "github.com/ship-monitor/cloud/pkg/connector/proto/nats"
)

type P struct {
	conn *nats.Conn
}

func New(
	conn *nats.Conn,
) *P {
	return &P{
		conn: conn,
	}
}

func (p *P) PublishConnectRequest(
	ctx context.Context,
	request *messages.ConnectRequestOut,
) error {
	msg := nats.NewMsg(
		channels.ConnectRequestsAddress(channels.ConnectRequestsParameters{
			DeviceID: parameters.DeviceID(request.Headers.DeviceID.String()),
		}).String(),
	)

	err := request.MarshalEnvelopeNats(natsGen.NewEnvelopeOut(msg.Data))
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	err = p.conn.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

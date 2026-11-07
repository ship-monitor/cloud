package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/pkg/connector/channels"
	"github.com/ship-monitor/cloud/pkg/connector/schemas/messages"
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
	r *messages.ConnectRequest,
) error {
	msg := nats.NewMsg(channels.ConnectRequestChannel(r.DeviceID))

	data, err := json.Marshal(&r)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	msg.Data = data

	err = p.conn.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

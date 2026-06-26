package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/log/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/amqputils"
)

const (
	QueueDurable          = true
	QueueDeleteWhenUnused = false
	QueueExclusive        = false
	QueueNoWait           = false
)

type DeviceID string

type RecordsService interface {
	AddRecord(ctx context.Context, record domain.StateRecord) error
}

type QueueWorker struct {
	conn    *amqp.Connection
	log     *log.Logger
	redis   *redis.Client
	service RecordsService
}

func NewQueue(
	conn *amqp.Connection,
	rdb *redis.Client,
	service RecordsService,
	logger *log.Logger,
) *QueueWorker {
	return &QueueWorker{
		conn:    conn,
		log:     logger,
		redis:   rdb,
		service: service,
	}
}

func (q *QueueWorker) Serve(ctx context.Context) error {
	ch, err := q.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	defer func() {
		if err := ch.Close(); err != nil {
			q.log.Error("Failed to close channel", "error", err)
		}
	}()

	exchangeName := "amq.topic"

	err = ch.ExchangeDeclare(
		exchangeName,
		amqp.ExchangeTopic,
		QueueDurable,
		QueueDeleteWhenUnused,
		false,
		QueueNoWait,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	statesQ, err := ch.QueueDeclare(
		"device_state",
		QueueDurable, QueueDeleteWhenUnused, QueueExclusive, QueueNoWait, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	bindingKey := "devices.*.state"

	err = ch.QueueBind(statesQ.Name, bindingKey, exchangeName, false, nil)
	if err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	messages, cancel, err := amqputils.ConsumeMessages(ctx, ch, statesQ)
	if err != nil {
		return fmt.Errorf("consume messages: %w", err)
	}
	defer cancel()

	for msg := range messages {
		go func(ctx context.Context, msg amqp.Delivery) {
			if err := q.handleDelivery(ctx, msg); err != nil {
				q.log.Error("Failed to handle message", "error", err)
			}
		}(ctx, msg)
	}

	return nil
}

const MessageHandlerTimeout = time.Second * 5

func (q *QueueWorker) handleDelivery(ctx context.Context, msg amqp.Delivery) error {
	ctx, cancel := context.WithTimeout(ctx, MessageHandlerTimeout)
	defer cancel()

	done := make(chan error)

	go func() {
		if err := q.handleMessage(ctx, msg); err != nil {
			q.log.Error("Failed to handle message", "error", err)

			done <- fmt.Errorf("handle message: %w", err)

			return
		}

		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("handle delivery: %w", err)
		}

		if err := msg.Ack(false); err != nil {
			q.log.Error("Failed to ack message", "error", err)
		}

		return nil
	case <-ctx.Done():
		q.log.Error("Message handler context cancelled", "error", ctx.Err())

		if err := ctx.Err(); err != nil {
			return fmt.Errorf("handle delivery: context cancelled: %w", err)
		}
	}

	return nil
}

func (q *QueueWorker) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	deviceID, err := getDeviceIDFromRoutingKey(msg.RoutingKey)
	if err != nil {
		return fmt.Errorf("get sender device id: %w", err)
	}

	q.log.Info("New message", "msg", string(msg.Body), "device", deviceID)

	var stateMessage StateMessage
	if err := json.Unmarshal(msg.Body, &stateMessage); err != nil {
		return fmt.Errorf("unmarshal state message: %w", err)
	}

	err = q.service.AddRecord(ctx, domain.StateRecord{
		DeviceID:  string(deviceID),
		State:     stateMessage.State,
		Value:     stateMessage.Value,
		Timestamp: stateMessage.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("add state record: %w", err)
	}

	return nil
}

type StateMessage struct {
	State     string    `json:"state"`
	Value     any       `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

const routingKeyParts = 3

func getDeviceIDFromRoutingKey(routingKey string) (DeviceID, error) {
	parts := strings.Split(routingKey, ".")
	if len(parts) >= routingKeyParts {
		return DeviceID(parts[1]), nil
	} else {
		return "", fmt.Errorf("invalid routing key: %s", routingKey)
	}
}

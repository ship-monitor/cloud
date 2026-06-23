package amqputils

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishJSON publishes content to channel as JSON and returns correlation ID.
func PublishJSON(
	ctx context.Context,
	ch *amqp.Channel,
	queue amqp.Queue,
	message any,
) (string, error) {
	correlationID := uuid.New().String()

	err := publishJSON(ctx, ch, queue, message, correlationID)
	if err != nil {
		return correlationID, fmt.Errorf("publish JSON: %w", err)
	}

	return correlationID, nil
}

func PublishAnswerJSON(ctx context.Context,
	ch *amqp.Channel,
	queue amqp.Queue,
	message any,
	correlationID string,
) error {
	err := publishJSON(ctx, ch, queue, message, correlationID)
	if err != nil {
		return fmt.Errorf("publish answer JSON: %w", err)
	}

	return nil
}

func publishJSON(
	ctx context.Context,
	ch *amqp.Channel,
	queue amqp.Queue,
	message any,
	correlationID string,
) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	if err := ch.PublishWithContext(
		ctx,
		"",         // default exchange
		queue.Name, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			CorrelationId: correlationID,
		},
	); err != nil {
		return fmt.Errorf("publish to channel: %w", err)
	}

	return nil
}

type CancelConsumerFunc = func()

func ConsumeMessages(
	ctx context.Context,
	ch *amqp.Channel,
	queue amqp.Queue,
) (<-chan amqp.Delivery, CancelConsumerFunc, error) {
	consumerID := uuid.New().String()

	messages, err := ch.ConsumeWithContext(
		ctx,
		queue.Name,
		consumerID,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("consume messages: %w", err)
	}

	cancel := func() {
		err := ch.Cancel(consumerID, false)
		if err != nil {
			log.Error("Failed to cancel consumer", "consumerID", consumerID, "error", err)
		}
	}

	return messages, cancel, nil
}

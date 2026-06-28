package services

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/amqputils"
)

const (
	QueueDurable          = true
	QueueDeleteWhenUnused = false
	QueueExclusive        = false
	QueueNoWait           = false
)

type TopicPublisher struct {
	q      *amqp.Connection
	logger *log.Logger
}

func NewTopicPublisher(q *amqp.Connection, logger *log.Logger) *TopicPublisher {
	return &TopicPublisher{q: q, logger: logger.WithPrefix("topic publisher")}
}

func (q *TopicPublisher) PublishJSON(
	ctx context.Context,
	topic string,
	data any,
) error {
	ch, err := q.q.Channel()
	if err != nil {
		return fmt.Errorf("faield declare channel: %w", err)
	}
	defer func() {
		if err := ch.Close(); err != nil {
			q.logger.Error("Failed close channel", "error", err)
		}
	}()

	exchangeName := "amq.topic"

	_, err = ch.QueueDeclare(
		topic,
		QueueDurable, QueueDeleteWhenUnused, QueueExclusive, QueueNoWait, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	err = ch.PublishWithContext(
		ctx,
		exchangeName,
		topic,
		false,
		false,
		amqp.Publishing{
			ContentType:     amqputils.ContentJSON,
			ContentEncoding: amqputils.ContentEncodingUTF8,
			Body:            jsonData,
		})
	if err != nil {
		return fmt.Errorf("failed publish JSON: %w", err)
	}

	return nil
}

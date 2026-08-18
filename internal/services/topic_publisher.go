package services

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ship-monitor/cloud/pkg/amqputils"
)

const (
	QueueDurable          = true
	QueueDeleteWhenUnused = false
	QueueExclusive        = false
	QueueNoWait           = false
)

type TopicPublisher struct {
	connection *amqp.Connection
	logger     *log.Logger
}

func NewTopicPublisher(q *amqp.Connection, logger *log.Logger) *TopicPublisher {
	return &TopicPublisher{
		connection: q,
		logger:     logger.WithPrefix("topic publisher"),
	}
}

func (q *TopicPublisher) PublishJSON(
	ctx context.Context,
	topic string,
	data any,
) error {
	ch, err := q.connection.Channel()
	if err != nil {
		return fmt.Errorf("faield declare channel: %w", err)
	}

	defer q.closeCh(ch)

	exchangeName := "amq.topic"

	queue, err := ch.QueueDeclare(
		topic+"_queue",
		QueueDurable,
		QueueDeleteWhenUnused,
		QueueExclusive,
		QueueNoWait,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	err = ch.QueueBind(
		queue.Name,
		topic,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("bind queue: %w", err)
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

func (q *TopicPublisher) closeCh(ch *amqp.Channel) {
	if err := ch.Close(); err != nil {
		q.logger.Error("Failed close channel", "error", err)
	}
}

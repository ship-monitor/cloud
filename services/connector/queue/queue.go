package queue

import (
	"context"
	"fmt"

	"charm.land/log/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/amqputils"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
)

func GetRabbitMQUrl() string {
	url := viper.GetString("services.connector.rabbitmq-url")
	if url == "" {
		log.Error("RabbitMQ URL is not set", "key", "services.connector.rabbitmq-url")
	}

	return url
}

type MessageHandlerFunc func(*amqp.Delivery) error

func connect(url string) (*amqp.Connection, error) {
	connection, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	} else {
		return connection, nil
	}
}

func Close(c *amqp.Connection) {
	if c == nil {
		return
	}

	err := c.Close()
	if err != nil {
		log.Error("Failed to close connection", "error", err)
	}
}

type Queue struct {
	conn            *amqp.Connection
	requests        amqp.Queue
	responses       amqp.Queue
	messageHandlers []MessageHandlerFunc
	channel         *amqp.Channel
}

func NewQueue() (*Queue, error) {
	conn, err := connect(GetRabbitMQUrl())
	if err != nil {
		return nil, fmt.Errorf("connect to queue: %w", err)
	}

	return &Queue{
		conn: conn,
	}, nil
}

func (q *Queue) AddHandler(h MessageHandlerFunc) {
	q.messageHandlers = append(q.messageHandlers, h)
}

func (q *Queue) SendResponse(
	ctx context.Context,
	requestId string,
	response *connections.ToCloudResponse,
) error {
	err := amqputils.PublishAnswerJSON(ctx, q.channel, q.responses, response, requestId)
	if err != nil {
		return fmt.Errorf("send response: %w", err)
	}

	return nil
}

func (q *Queue) Serve(ctx context.Context) error {
	if ch, err := q.conn.Channel(); err != nil {
		return fmt.Errorf("open amqp channel: %w", err)
	} else {
		q.channel = ch
	}

	if req, res, err := q.setupQueues(); err != nil {
		return fmt.Errorf("setup queues: %w", err)
	} else {
		q.requests = req
		q.responses = res
	}

	messages, cancel, err := amqputils.ConsumeMessages(ctx, q.channel, q.requests)
	if err != nil {
		return fmt.Errorf("register consumer: %w", err)
	}
	defer cancel()

	for m := range messages {
		go q.handleMessage(ctx, &m)
	}

	return nil
}

func (q *Queue) handleMessage(ctx context.Context, msg *amqp.Delivery) {
	for _, handler := range q.messageHandlers {
		go func() {
			err := handler(msg)
			if err != nil {
				log.Error(
					"Error while handling message, sending internal error back",
					"correlationId",
					msg.CorrelationId,
					"error",
					err,
				)

				err2 := q.sendError(ctx, msg.CorrelationId, err)
				if err2 != nil {
					log.Error("Failed to send error response", "error", err2)
				}
			}
		}()
	}

	<-ctx.Done()
}

func (q *Queue) sendError(ctx context.Context, requestId string, err error) error {
	log.Error("Sending error response", "error", err, "requestId", requestId)
	response := &connections.ToCloudResponse{
		RequestError: err.Error(),
	}

	return q.SendResponse(ctx, requestId, response)
}

const (
	QueueDurable          = true
	QueueDeleteWhenUnused = false
	QueueExclusive        = false
	QueueNoWait           = false
)

func declareQueue(ch *amqp.Channel, name string) (amqp.Queue, error) {
	queue, err := ch.QueueDeclare(
		name,
		QueueDurable,
		QueueDeleteWhenUnused,
		QueueExclusive,
		QueueNoWait,
		nil,
	)
	if err != nil {
		return queue, fmt.Errorf("declare queue %q: %w", name, err)
	}

	return queue, nil
}

// setupQueues returns requests and responses queue and error.
func (q *Queue) setupQueues() (amqp.Queue, amqp.Queue, error) {
	requests, err := declareQueue(q.channel, "requests-1")
	if err != nil {
		return requests, amqp.Queue{}, err
	}

	responses, err := declareQueue(q.channel, "responses-1")
	if err != nil {
		return requests, responses, err
	}

	return requests, responses, nil
}

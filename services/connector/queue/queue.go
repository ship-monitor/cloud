package queue

import (
	"context"
	"fmt"
	"sync"

	"charm.land/log/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/amqputils"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
)

var (
	conn      *amqp.Connection
	channel   *amqp.Channel
	requests  amqp.Queue
	responses amqp.Queue
)

func GetRabbitMQUrl() string {
	url := viper.GetString("services.connector.rabbitmq-url")
	if url == "" {
		log.Error("RabbitMQ URL is not set", "key", "services.connector.rabbitmq-url")
	}

	return url
}

type MessageHandlerFunc func(*amqp.Delivery) error

var messageHandlers = []MessageHandlerFunc{}

func AddHandler(h MessageHandlerFunc) {
	messageHandlers = append(messageHandlers, h)
}

func closeConnection() {
	if conn == nil {
		return
	}
	err := conn.Close()
	if err != nil {
		log.Error("Failed to close connection", "error", err)
	}
}

func Connect(url string) (*amqp.Connection, error) {
	if connection, err := amqp.Dial(url); err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	} else {
		return connection, nil
	}
}

func Close(c *amqp.Connection) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		log.Error("Failed to close connection", "error", err)
	}
}

func Serve(connection *amqp.Connection) error {
	conn = connection

	if ch, err := conn.Channel(); err != nil {
		return fmt.Errorf("open channel: %w", err)
	} else {
		channel = ch
	}

	if req, res, err := setupQueues(channel); err != nil {
		return fmt.Errorf("setup queues: %w", err)
	} else {
		requests = req
		responses = res
	}
	messages, cancel, err := amqputils.ConsumeMessages(context.TODO(), channel, requests)
	if err != nil {
		return fmt.Errorf("register consumer: %w", err)
	}
	defer cancel()

	for m := range messages {
		go handleMessage(&m)
	}

	return nil
}

func handleMessage(msg *amqp.Delivery) {
	wg := sync.WaitGroup{}

	for _, handler := range messageHandlers {
		wg.Go(func() {
			if err := handler(msg); err != nil {
				log.Error(
					"Error while handling message, sending internal error back",
					"correlationId",
					msg.CorrelationId,
					"error",
					err,
				)

				if err2 := sendError(msg.CorrelationId, err); err2 != nil {
					log.Error("Failed to send error response", "error", err2)
				}
			}
		})
	}
	wg.Wait()
}

func sendError(requestId string, err error) error {
	log.Error("Sending error response", "error", err, "requestId", requestId)
	response := &connections.ToCloudResponse{
		RequestError: err.Error(),
	}

	return SendResponse(requestId, response)
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
		return queue, fmt.Errorf("declare queue: %w", err)
	}

	return queue, nil
}

func setupQueues(channel *amqp.Channel) (requests, responses amqp.Queue, err error) {
	requests, err = declareQueue(channel, "requests-1")
	if err != nil {
		return requests, responses, fmt.Errorf("declare %q queue: %w", "requests-1", err)
	}

	responses, err = declareQueue(channel, "responses-1")
	if err != nil {
		return requests, responses, fmt.Errorf("declare %q queue: %w", "responses-1", err)
	}

	return requests, responses, nil
}

func SendResponse(requestId string, response *connections.ToCloudResponse) error {
	err := amqputils.PublishAnswerJSON(context.TODO(), channel, responses, response, requestId)
	if err != nil {
		return fmt.Errorf("send response: %w", err)
	}

	return nil
}

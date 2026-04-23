package queue

import (
	"encoding/json"
	"fmt"
	"sync"

	"charm.land/log/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
)

var (
	conn      *amqp.Connection
	channel   *amqp.Channel
	requests  amqp.Queue
	responses amqp.Queue
)

func getRabbitMQUrl() string {
	url := viper.GetString("services.connector.rabbitmq-url")
	if url == "" {
		log.Error("RabbitMQ URL is not set", "key", "services.connector.rabbitmq-url")
	}

	return url
}

type MessageHandlerFunc func(*amqp.Delivery) error

var (
	messageHandlers = []MessageHandlerFunc{}
)

func AddHandler(h MessageHandlerFunc) {
	messageHandlers = append(messageHandlers, h)
}

func Serve() {
	url := getRabbitMQUrl()
	if connection, err := amqp.Dial(url); err != nil {
		log.Fatal("Failed to connect to RabbitMQ", "error", err)
	} else {
		conn = connection
	}
	log.Info("Connected to RabbitMQ")
	defer conn.Close()

	if ch, err := conn.Channel(); err != nil {
		log.Fatal("Failed to open a channel", "error", err)
	} else {
		channel = ch
	}

	if req, res, err := setupQueues(channel); err != nil {
		log.Fatal("Failed to setup queues", "error", err)
	} else {
		requests = req
		responses = res
	}

	messages, err := channel.Consume(
		requests.Name, // queue
		"",            // consumer
		true,          // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)

	if err != nil {
		log.Fatal("Failed to register a consumer", "error", err)
	}

	for m := range messages {
		go handleMessage(&m)
	}
}

func handleMessage(msg *amqp.Delivery) error {

	wg := sync.WaitGroup{}

	for _, handler := range messageHandlers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handler(msg); err != nil {
				log.Error("Error while handling message, sending internal error back", "correlationId", msg.CorrelationId, "error", err)
				sendError(msg.CorrelationId, err)
			}
		}()
	}

	return nil
}

func BadRequestBodyErr(err error) error {
	return fmt.Errorf("bad request body: %s", err)
}

func sendError(requestId string, err error) error {
	log.Error("Sending error response", "error", err, "requestId", requestId)
	response := &connections.ToCloudResponse{
		RequestError: err.Error(),
	}

	return SendResponse(requestId, response)
}

func setupQueues(channel *amqp.Channel) (requests amqp.Queue, responses amqp.Queue, err error) {

	requests, err = channel.QueueDeclare(
		"requests", // name
		false,      // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare a queue", "error", err)
		return requests, responses, err
	}

	responses, err = channel.QueueDeclare(
		"responses", // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		log.Fatal("Failed to declare a queue", "error", err)
		return requests, responses, err
	}

	return requests, responses, nil
}

func SendResponse(requestId string, response *connections.ToCloudResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return channel.Publish(
		"",             // exchange
		responses.Name, // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: requestId,
			Body:          body,
		},
	)
}

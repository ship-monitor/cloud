package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

const (
	requestsQueueName  = "requests-1"
	responsesQueueName = "responses-1"
)

var (
	conn           *amqp.Connection
	channel        *amqp.Channel
	requestsQueue  amqp.Queue
	responsesQueue amqp.Queue
)

type Command struct {
	NodeID  string         `json:"nodeId"`
	Command string         `json:"command"`
	Args    map[string]any `json:"args"`
}

func NewCommand(nodeID, command string, args map[string]any) Command {
	return Command{
		NodeID:  nodeID,
		Command: command,
		Args:    args,
	}
}

func (cmd Command) valid() (bool, error) {
	if cmd.Command == "" {
		return false, fmt.Errorf("command not specified")
	}
	if cmd.NodeID == "" {
		return false, fmt.Errorf("node ID not specified")
	}

	return true, nil
}

// Connect establishes the RabbitMQ connection and declares the queue.
// Call once at service startup.
func Connect() error {
	url := viper.GetString("services.connector.rabbitmq-url")
	if url == "" {
		log.Error("RabbitMQ URL is not set", "key", "services.connector.rabbitmq-url")
	}

	var err error
	conn, err = amqp.Dial(url)
	if err != nil {
		return err
	}

	channel, err = conn.Channel()
	if err != nil {
		return err
	}

	requestsQueue, err = channel.QueueDeclare(
		requestsQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}
	responsesQueue, err = channel.QueueDeclare(
		responsesQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func Close() {
	if conn == nil {
		return
	}
	if err := conn.Close(); err != nil {
		log.Error("Something went wrong while closing amqp connection", "error", err)
	}
}

type CommandResponse struct {
	RequestError string         `json:"requestError"` // Connector error
	CommandError string         `json:"commandError"` // Node error
	Data         map[string]any `json:"data"`
}

func badResponse(format string, args ...any) CommandResponse {
	return CommandResponse{
		RequestError: fmt.Sprintf(format, args...),
	}
}

// Publish content to channel as JSON and returns correlation ID.
func publishJSON(
	ctx context.Context,
	ch *amqp.Channel,
	queue amqp.Queue,
	message any,
) (string, error) {
	body, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("publish JSON: %w", err)
	}

	correlationID := uuid.New().String()

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
		return "", fmt.Errorf("publish JSON %w", err)
	}

	return correlationID, nil
}

// SendCommand sends an event to the org-events queue.
func SendCommand(ctx context.Context, cmd Command) CommandResponse {
	if valid, err := cmd.valid(); !valid {
		return badResponse("invalid command: %s", err)
	}

	requestID, err := publishJSON(context.TODO(), channel, requestsQueue, cmd)
	if err != nil {
		return badResponse("failed publish command: %s", err)
	}

	messages, cancel, err := consumeMessages(channel, responsesQueue)
	if err != nil {
		return badResponse("failed to consume response: %s", err)
	}
	defer cancel()

	ctx, cancelCtx := context.WithTimeout(
		ctx,
		viper.GetDuration("services.organizations.response-command-timeout"),
	)
	defer cancelCtx()

	for {
		select {
		case response := <-messages:
			if response.CorrelationId != requestID {
				continue
			}
			var commandResponse CommandResponse
			if err := json.Unmarshal(response.Body, &commandResponse); err != nil {
				log.Error("Failed to unmarshal response", "error", err)

				return badResponse("failed to unmarshal response: %s", err)
			}
			defer acknowledge(response)
			log.Info("Received response", "requestID", requestID, "response", response)

			return commandResponse
		case <-ctx.Done():
			return badResponse("context done: %s", ctx.Err())
		}
	}
}

type CancelConsumerFunc = func()

func consumeMessages(
	ch *amqp.Channel,
	queue amqp.Queue,
) (<-chan amqp.Delivery, CancelConsumerFunc, error) {
	consumerID := uuid.New().String()

	messages, err := ch.Consume(
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
		if err := ch.Cancel(consumerID, false); err != nil {
			log.Error("Failed to cancel consumer", "consumerID", consumerID, "error", err)
		}
	}

	return messages, cancel, nil
}

func acknowledge(d amqp.Delivery) {
	if err := d.Ack(false); err != nil {
		log.Error("Failed to acknowledge message", "requestID", d.CorrelationId, "error", err)
	}
}

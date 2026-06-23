package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"charm.land/log/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/amqputils"
)

const (
	requestsQueueName  = "requests-1"
	responsesQueueName = "responses-1"
)

var (
	//nolint:gochecknoglobals
	conn *amqp.Connection
	//nolint:gochecknoglobals
	channel *amqp.Channel
	//nolint:gochecknoglobals
	requestsQueue amqp.Queue
	//nolint:gochecknoglobals
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
		return false, errors.New("command not specified")
	}

	if cmd.NodeID == "" {
		return false, errors.New("node ID not specified")
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
		return fmt.Errorf("amqp dial: %w", err)
	}

	channel, err = conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
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
		return fmt.Errorf("declare queue: %w", err)
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
		return fmt.Errorf("declare queue: %w", err)
	}

	return nil
}

func Close() {
	if conn == nil {
		return
	}

	err := conn.Close()
	if err != nil {
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
		CommandError: "",
		Data:         nil,
	}
}

// SendCommand sends an event to the org-events queue.
func SendCommand(ctx context.Context, cmd Command) CommandResponse {
	if valid, err := cmd.valid(); !valid {
		return badResponse("invalid command: %s", err)
	}

	requestID, err := amqputils.PublishJSON(ctx, channel, requestsQueue, cmd)
	if err != nil {
		return badResponse("failed publish command: %s", err)
	}

	messages, cancel, err := amqputils.ConsumeMessages(ctx, channel, responsesQueue)
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

			err := json.Unmarshal(response.Body, &commandResponse)
			if err != nil {
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

func acknowledge(d amqp.Delivery) {
	err := d.Ack(false)
	if err != nil {
		log.Error("Failed to acknowledge message", "requestID", d.CorrelationId, "error", err)
	}
}

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

const requestsQueueName = "requests-1"
const responsesQueueName = "responses-1"

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
	if conn != nil {
		_ = conn.Close()
	}
}

type CommandResponse struct {
	RequestError string         `json:"requestError"` // Connector error
	CommandError string         `json:"commandError"` // Node error
	Data         map[string]any `json:"data"`
}

// SendCommand sends an event to the org-events queue.
func SendCommand(deviceID uuid.UUID, command string, args map[string]any) CommandResponse {

	requestID := uuid.New().String()
	log.Info("Sending command", "requestID", requestID, "deviceID", deviceID, "command", command, "args", args)

	body, err := json.Marshal(Command{
		NodeID:  deviceID.String(),
		Command: command,
		Args:    args,
	})
	if err != nil {
		return CommandResponse{
			RequestError: fmt.Errorf("bad payload: %s", err).Error(),
		}
	}

	if err := channel.PublishWithContext(
		context.Background(),
		"",                 // default exchange
		requestsQueue.Name, // routing key
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			CorrelationId: requestID,
		},
	); err != nil {
		return CommandResponse{
			RequestError: fmt.Errorf("failed to publish command: %s", err).Error(),
		}
	}

	messages, err := channel.Consume(responsesQueue.Name, requestID, false, false, false, false, nil)
	if err != nil {
		return CommandResponse{
			RequestError: fmt.Errorf("failed to consume response: %s", err).Error(),
		}
	}
	defer channel.Cancel(requestID, false)
	for {

		select {
		case response := <-messages:
			if response.CorrelationId != requestID {
				log.Info("Received response with wrong requestID", "requestID", requestID, "response", response)
				continue
			}
			var commandResponse CommandResponse
			if err := json.Unmarshal(response.Body, &commandResponse); err != nil {
				log.Error("Failed to unmarshal response", "error", err)

				return CommandResponse{
					RequestError: fmt.Errorf("failed to unmarshal response: %s", err).Error(),
				}
			}
			defer response.Ack(false)
			log.Info("Received response", "requestID", requestID, "response", response)
			return commandResponse
		case <-time.After(viper.GetDuration("services.organizations.response-command-timeout")):
			log.Error("Timeout reached", "requestID", requestID)
			return CommandResponse{
				RequestError: fmt.Sprintf("timeout %v reached", viper.GetDuration("services.organizations.response-command-timeout")),
			}
		}
	}
}

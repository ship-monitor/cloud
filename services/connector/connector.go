package connector

import (
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/queue"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/repository"
)

func Setup(r gin.IRouter) {
	repository.Migrate()
	r.GET("/nodes/:id", handlers.GetSingleClientHandler())

	go func() {
		connections.Serve()
	}()

	go func() {
		queue.Serve()
	}()

	queue.AddHandler(queueHandler)

	connections.AddHandler(websocketHandler)
}

var (
	qlog  = log.WithPrefix("Queue")
	wsLog = log.WithPrefix("WebSocket")
)

func queueHandler(m *amqp.Delivery) error {
	requestId := m.CorrelationId

	qlog.Info("New message", "requestId", requestId, "body", string(m.Body))

	var cloudRequest connections.FromCloudRequest
	if err := json.Unmarshal(m.Body, &cloudRequest); err != nil {
		qlog.Error("Failed to unmarshal message", "error", err)
		return err
	}

	if err := cloudRequest.Validate(); err != nil {
		qlog.Error("Failed validate request from cloud", "error", err, "requestId", requestId)
		return fmt.Errorf("failed validate request from cloud: %s", err)
	}

	if err := connections.SendRequest(cloudRequest.NodeID, cloudRequest.ToNode(requestId)); err != nil {
		qlog.Error("Failed to send request", "error", err)
		return err
	}
	qlog.Info("Message handled", "requestId", requestId)
	return nil
}
func websocketHandler(body []byte) error {
	var response connections.FromNodeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		wsLog.Error("Failed to unmarshal message", "error", err)
		return err
	}

	wsLog.Info("New message", "requestId", response.RequestID)

	if err := queue.SendResponse(response.RequestID, response.ToCloud()); err != nil {
		wsLog.Error("Failed to send response", "error", err)
		return err
	}
	wsLog.Info("Message handled", "requestId", response.RequestID)
	return nil
}

package connector

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	repository "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repositories"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/queue"
)

func Setup(ctx context.Context, r gin.IRouter) {
	repo := repository.NewNodes(db.DB)
	if err := repo.Migrate(ctx); err != nil {
		panic(err)
	}

	h := handlers.NewHandlers(repo)

	r.GET("/nodes/:id", h.GetSingleClientHandler())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	q, err := queue.NewQueue()
	if err != nil {
		panic(err)
	}

	connServer := connections.NewServer(repo)

	q.AddHandler(queueHandler(connServer))

	connServer.AddHandler(websocketHandler(q))

	go func(ctx context.Context) {
		go func(ctx context.Context) {
			connServer.Serve(ctx)
		}(ctx)

		go func(ctx context.Context) {
			if err := q.Serve(ctx); err != nil {
				log.Fatal("failed serve queue", "error", err)
			}
		}(ctx)

		<-ctx.Done()
	}(ctx)
}

var (
	//nolint:gochecknoglobals
	qlog = log.WithPrefix("Queue")
	//nolint:gochecknoglobals
	wsLog = log.WithPrefix("WebSocket")
)

func queueHandler(c *connections.Server) queue.MessageHandlerFunc {
	return func(m *amqp.Delivery) error {
		requestId := m.CorrelationId

		qlog.Info("New message", "requestId", requestId, "body", string(m.Body))

		var cloudRequest connections.FromCloudRequest

		err := json.Unmarshal(m.Body, &cloudRequest)
		if err != nil {
			qlog.Error("Failed to unmarshal message", "error", err)

			return fmt.Errorf("unmarshal message: %w", err)
		}

		err = cloudRequest.Validate()
		if err != nil {
			qlog.Error("Failed validate request from cloud", "error", err, "requestId", requestId)

			return fmt.Errorf("failed validate request from cloud: %w", err)
		}

		err = c.SendRequest(
			cloudRequest.NodeID,
			cloudRequest.ToNode(requestId),
		)
		if err != nil {
			qlog.Error("Failed to send request", "error", err)

			return fmt.Errorf("send request: %w", err)
		}

		qlog.Info("Message handled", "requestId", requestId)

		return nil
	}
}

func websocketHandler(q *queue.Queue) connections.MessageHandlerFunc {
	return func(ctx context.Context, body []byte) error {
		var response connections.FromNodeResponse

		err := json.Unmarshal(body, &response)
		if err != nil {
			wsLog.Error("Failed to unmarshal message", "error", err)

			return fmt.Errorf("unmarshal message: %w", err)
		}

		wsLog.Info("New message", "requestId", response.RequestID, "body", string(body))

		err = q.SendResponse(ctx, response.RequestID, response.ToCloud())
		if err != nil {
			wsLog.Error("Failed to send response", "error", err)

			return fmt.Errorf("send response to queue: %w", err)
		}

		wsLog.Info("Message handled", "requestId", response.RequestID)

		return nil
	}
}

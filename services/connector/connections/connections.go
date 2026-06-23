package connections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/repository"
)

const (
	NodeIDHeader = "X-Node-Id"

	readBufferSize  = 1024
	writeBufferSize = 1024

	readHeaderTimeout = time.Second * 2
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  readBufferSize,
		WriteBufferSize: writeBufferSize,
	}
	connections = map[UUID]*websocket.Conn{}
	connMu      sync.RWMutex
)

func AppendConnection(ctx context.Context, node *AuthData, conn *websocket.Conn) {
	connMu.Lock()
	defer connMu.Unlock()

	if existingNode, _ := repository.GetNode(ctx, node.NodeID); existingNode != nil {
		log.Warn("Connection with that node already exists, closing it", "nodeId", node.NodeID)

		if _, err := repository.ReconnectNode(ctx, node.NodeID); err != nil {
			log.Error("Failed to reconnect node", "error", err)
		}
	} else {
		if _, err := repository.NewNode(ctx, node.NodeID, "DUMMY NAME"); err != nil {
			log.Error("Failed to create node", "error", err)
		}
	}

	connections[node.NodeID] = conn
}

func closeConn(ctx context.Context, nodeID UUID, conn *websocket.Conn) {
	connMu.Lock()
	defer connMu.Unlock()

	addr := fmt.Sprintf("%s %s", conn.RemoteAddr().Network(), conn.RemoteAddr().String())

	delete(connections, nodeID)

	if _, err := repository.UpdateLastConnection(ctx, nodeID); err != nil {
		log.Error("Failed to update last connection", "error", err)
	}

	log.Warn("Connection closed", "address", addr)
}

func Serve(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ipc/connect", func(w http.ResponseWriter, r *http.Request) {
		log.Info("New connection", "address", r.RemoteAddr)

		auth, err := checkAuth(r)
		if err != nil {
			log.Error("Failed authenticate connection", "error", err)

			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Failed to upgrade connection", "error", err)

			return
		}

		AppendConnection(ctx, auth, conn)

		go serveConnection(ctx, auth.NodeID, conn)

		log.Info("Connection established", "nodeId", auth.NodeID)
	})

	log.Info("Starting connections server", "address", getAddress())
	go func() {
		server := http.Server{
			Addr:              getAddress(),
			Handler:           mux,
			ReadHeaderTimeout: readHeaderTimeout,
		}

		err := server.ListenAndServe()
		if err != nil {
			log.Error("Failed to start connections server", "error", err)
			panic(err)
		}
	}()

	<-ctx.Done()
}

type AuthData struct {
	NodeID UUID
}
type MessageHandlerFunc func(ctx context.Context, message []byte) error

var handlers = []MessageHandlerFunc{}

func AddHandler(handler MessageHandlerFunc) {
	handlers = append(handlers, handler)
}

func serveConnection(ctx context.Context, nodeId UUID, conn *websocket.Conn) {
	defer func() {
		if recover() != nil {
			log.Error("Connection closed with panic", "nodeId", nodeId, "panic", recover())
		}

		closeConn(ctx, nodeId, conn)
	}()

	go func() {
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Error("Failed to read message", "error", err)

				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Error("Unexpected closing connection", "error", err)

					return
				}

				continue
			}

			if messageType == websocket.CloseMessage {
				return
			}

			for _, handler := range handlers {
				ctx, cancel := context.WithTimeout(ctx, handlerTimeout)
				defer cancel()

				go func(ctx context.Context, handler MessageHandlerFunc) {
					err := handler(ctx, message)
					if err != nil {
						log.Error("Failed to handle message", "error", err)
					}
				}(ctx, handler)
			}
		}
	}()

	<-ctx.Done()
}

const handlerTimeout = time.Second * 10

func checkAuth(r *http.Request) (*AuthData, error) {
	nodeID, err := uuid.Parse(r.Header.Get(NodeIDHeader))
	if err != nil {
		log.Error(
			"Bad UUID specifications for node id header",
			"header",
			NodeIDHeader,
			"error",
			err,
		)

		return nil, fmt.Errorf("bad node id header %q specified: %w", NodeIDHeader, err)
	}

	return &AuthData{
		NodeID: nodeID,
	}, nil
}

func getAddress() string {
	port := viper.GetInt("services.connector.ws-port")
	if port <= 0 {
		log.Error(
			"Invalid or not set websocket port",
			"port",
			port,
			"key",
			"services.connector.ws-port",
		)
	}

	return fmt.Sprintf(":%d", port)
}

func SendRequest(nodeId string, r *ToNodeRequest) error {
	id, err := uuid.Parse(nodeId)
	if err != nil {
		return fmt.Errorf("bad node id %q: %w", nodeId, err)
	}

	conn, ok := connections[id]
	if !ok {
		return fmt.Errorf("node with id %q not connected", nodeId)
	}

	message, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func IsConnected(nodeId UUID) bool {
	connMu.RLock()
	defer connMu.RUnlock()

	_, ok := connections[nodeId]

	return ok
}

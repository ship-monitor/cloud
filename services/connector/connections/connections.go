package connections

import (
	"encoding/json"
	"fmt"
	"net/http"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/repository"
)

const (
	NodeIDHeader = "X-Node-ID"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	connections = map[UUID]*websocket.Conn{}
)

func AppendConnection(node *AuthData, conn *websocket.Conn) {

	if existingNode, _ := repository.GetNode(node.NodeID); existingNode != nil {
		log.Warn("Connection with that node already exists, closing it", "nodeId", node.NodeID)
		if _, err := repository.ReconnectNode(node.NodeID); err != nil {
			log.Error("Failed to reconnect node", "error", err)
		}
	} else {
		if _, err := repository.NewNode(node.NodeID, "DUMMY NAME"); err != nil {
			log.Error("Failed to create node", "error", err)
		}
	}

	connections[node.NodeID] = conn
}

func closeConn(nodeID UUID, conn *websocket.Conn) {
	addr := fmt.Sprintf("%s %s", conn.RemoteAddr().Network(), conn.RemoteAddr().String())

	if _, err := repository.UpdateLastConnection(nodeID); err != nil {
		log.Error("Failed to update last connection", "error", err)
	}

	log.Warn("Connection closed", "address", addr)
}

func Serve() {
	srv := http.NewServeMux()
	srv.HandleFunc("/api/ipc/connect", func(w http.ResponseWriter, r *http.Request) {
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

		AppendConnection(auth, conn)

		go serveConnection(auth.NodeID, conn)

		log.Info("Connection established", "nodeId", auth.NodeID)
	})

	log.Info("Starting connections server", "address", getAddress())
	if err := http.ListenAndServe(getAddress(), srv); err != nil {
		log.Error("Failed to start connections server", "error", err)
		panic(err)
	}
}

type AuthData struct {
	NodeID UUID
}
type MessageHandlerFunc func(message []byte) error

var handlers = []MessageHandlerFunc{}

func AddHandler(handler MessageHandlerFunc) {
	handlers = append(handlers, handler)
}

func serveConnection(nodeId UUID, conn *websocket.Conn) {
	defer func() {
		if recover() != nil {
			log.Error("Connection closed with panic", "nodeId", nodeId, "panic", recover())
		}
		closeConn(nodeId, conn)
	}()
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
			go func() {
				if err := handler(message); err != nil {
					log.Error("Failed to handle message", "error", err)
				}
			}()
		}
	}
}

func checkAuth(r *http.Request) (*AuthData, error) {

	nodeID, err := uuid.Parse(r.Header.Get(NodeIDHeader))

	if err != nil {
		log.Error("Bad UUID specifications for node id header", "header", NodeIDHeader, "error", err)
		return nil, fmt.Errorf("bad node id header %q specified: %s", NodeIDHeader, err)
	}

	return &AuthData{
		NodeID: nodeID,
	}, nil
}

func getAddress() string {
	port := viper.GetInt("services.connector.ws-port")
	if port <= 0 {
		log.Error("Invalid or not set websocket port", "port", port, "key", "services.connector.ws-port")
	}
	return fmt.Sprintf(":%d", port)
}

func SendRequest(nodeId string, r *ToNodeRequest) error {
	id, err := uuid.Parse(nodeId)
	if err != nil {
		return fmt.Errorf("bad node id %q: %s", nodeId, err)
	}
	conn, ok := connections[id]
	if !ok {
		return fmt.Errorf("node with id %q not connected", nodeId)
	}

	message, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, message)
}

func IsConnected(nodeId UUID) bool {
	_, ok := connections[nodeId]
	return ok
}

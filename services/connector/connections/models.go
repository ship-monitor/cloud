package connections

import (
	"errors"

	"github.com/google/uuid"
)

type JsonMap = map[string]any

type UUID = uuid.UUID

type FromCloudRequest struct {
	NodeID  string  `json:"nodeId"`
	Command string  `json:"command"`
	Args    JsonMap `json:"args"`
}
type ToCloudResponse struct {
	RequestError string  `json:"requestError"` // Connector error
	CommandError string  `json:"commandError"` // Node error
	Data         JsonMap `json:"data"`
}

type ToNodeRequest struct {
	RequestID string  `json:"requestId"`
	Command   string  `json:"command"`
	Args      JsonMap `json:"args"`
}
type FromNodeResponse struct {
	RequestID string  `json:"requestId"`
	Data      JsonMap `json:"data"`
	Error     string  `json:"error"`
}

func (r *FromCloudRequest) Validate() error {
	if r.NodeID == "" {
		return errors.New("empty node id specified")
	}

	if r.Command == "" {
		return errors.New("command not specified")
	}

	return nil
}

func (r *FromCloudRequest) ToNode(requestId string) *ToNodeRequest {
	return &ToNodeRequest{
		RequestID: requestId,
		Command:   r.Command,
		Args:      r.Args,
	}
}

func (r *FromNodeResponse) ToCloud() *ToCloudResponse {
	return &ToCloudResponse{
		CommandError: r.Error,
		Data:         r.Data,
	}
}

package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"userId"`

	Version    int64     `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	ExpiresAt  time.Time `json:"expiresAt"`

	// Необязательно, для отображения списка устройств.
	UserAgent string `json:"userAgent,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

func (s Session) JSON() ([]byte, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	return data, nil
}

type Principal struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

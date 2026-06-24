package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Node struct {
	*bun.BaseModel `bun:"table:nodes"`

	ID              uuid.UUID  `bun:",pk,type:varchar"      json:"id"`
	Name            string     `bun:",notnull"              json:"name"`
	FirstConnection time.Time  `bun:",notnull,type:varchar" json:"firstConnection"`
	LastConnection  *time.Time `bun:",notnull,type:varchar" json:"lastConnection"`
}

type Repository interface {
	GetNode(ctx context.Context, nodeID uuid.UUID) (*Node, error)
	NewNode(ctx context.Context, nodeID uuid.UUID, name string) (*Node, error)
	ReconnectNode(ctx context.Context, nodeID uuid.UUID) (*Node, error)
	UpdateLastConnection(ctx context.Context, nodeID uuid.UUID) (*Node, error)
}

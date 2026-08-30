package device

import (
	"context"

	"github.com/google/uuid"
)

// Decision represents response for RabbitMQ http auth, that queue service will use to decide on user auth.
type Decision = string

const (
	Allow      = Decision("allow")
	Deny       = Decision("deny")
	AllowAdmin = Decision("allow administrator")
)

func (s *Service) AuthUserPath(ctx context.Context, deviceID uuid.UUID, password string) (Decision, error)

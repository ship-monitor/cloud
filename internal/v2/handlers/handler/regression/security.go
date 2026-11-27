package regression

import (
	"context"

	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

type Security struct {
	actor auth.Actor
}

func NewSecurity(actor auth.Actor) *Security {
	return &Security{actor: actor}
}

func (s Security) HandleBearerToken(
	ctx context.Context,
	_ api.OperationName,
	_ api.BearerToken,
) (context.Context, error) {
	return auth.WithActor(ctx, s.actor), nil
}

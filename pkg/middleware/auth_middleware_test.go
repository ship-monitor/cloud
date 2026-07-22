package middleware_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
)

func TestPrincipalInCtx(t *testing.T) {
	t.Parallel()

	ctx := &gin.Context{}

	principal := domain.Principal{
		UserID:    uuid.Max,
		SessionID: uuid.Nil,
	}

	middleware.AddToContext(ctx, &principal)

	if _, err := middleware.PrincipalFromContext(ctx); err != nil {
		t.Errorf("must be principal, but returned error: %v", err)
	}
}

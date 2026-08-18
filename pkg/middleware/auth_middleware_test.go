package middleware_test

import (
	"net/http"
	"testing"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/pkg/middleware"
)

func TestPrincipalInCtx(t *testing.T) {
	t.Parallel()

	request, _ := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/example",
		nil,
	)
	ctx := echo.NewContext(request, nil)

	principal := domain.Principal{
		UserID:    uuid.Max,
		SessionID: uuid.Nil,
	}

	middleware.AddToContext(ctx, &principal)

	m := middleware.NewAuthMiddleware(nil, nil, log.Default())

	if _, err := m.PrincipalFromContext(ctx); err != nil {
		t.Errorf("must be principal, but returned error: %v", err)
	}
}

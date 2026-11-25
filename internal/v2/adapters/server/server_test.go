package server_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/ogen-go/ogen/middleware"
	"github.com/ship-monitor/cloud/internal/v2/adapters/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoggingPropagatesHandlerError(t *testing.T) {
	t.Parallel()

	want := io.EOF
	request := middleware.Request{ //nolint:exhaustruct_v5
		Raw: httptest.NewRequestWithContext(t.Context(), "GET", "/", nil),
	}
	_, err := server.Logging(zap.NewNop())(
		request,
		func(middleware.Request) (middleware.Response, error) {
			return middleware.Response{}, want
		},
	)
	require.ErrorIs(t, err, want)
}

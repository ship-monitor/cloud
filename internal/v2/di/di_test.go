package di

import (
	"context"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/ship-monitor/cloud/internal/v2/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestHTTPServerStartFailsWhenPortIsInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	app := fx.New(
		fx.NopLogger,
		fx.Invoke(func(lc fx.Lifecycle) {
			newHTTPServer(lc, &config.Config{Host: host, Port: port}, nil,
				zap.NewNop())
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = app.Start(ctx)
	if err == nil {
		t.Cleanup(func() {
			stopCtx, stop := context.WithTimeout(
				context.Background(),
				time.Second,
			)
			defer stop()

			_ = app.Stop(stopCtx)
		})
	}

	require.Error(t, err, "startup must fail when the HTTP port is occupied")
	require.ErrorIs(t, err, syscall.EADDRINUSE, "%v", err)
}

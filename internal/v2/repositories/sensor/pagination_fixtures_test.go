//go:build integration

package sensor_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := t.Context()

	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if container != nil {
		testcontainers.CleanupContainer(t, container)
	}

	require.NoError(t, err)
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	sqlDB, err := db.New(&db.Config{DSN: connStr})
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})

	require.NoError(t, db.Migrate(ctx, sqlDB))

	return sqlDB
}

func checkPages[T any, C comparable](
	t *testing.T,
	list func(
		context.Context,
		pagination.Request[C],
	) (pagination.Page[T, C], error),
	want []T,
	key func(T) C,
) {
	t.Helper()

	req := pagination.Request[C]{After: nil, Limit: 1}
	for index, item := range want {
		page, err := list(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []T{item}, page.Items)

		cursor := key(item)
		if index+1 < len(want) {
			require.Equal(t, &cursor, page.Next)
			req.After = page.Next
		} else {
			require.Nil(t, page.Next)

			req.After = &cursor
		}
	}

	page, err := list(t.Context(), req)
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Nil(t, page.Next)
}

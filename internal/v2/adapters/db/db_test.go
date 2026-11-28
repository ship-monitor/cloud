//go:build integration

package db_test

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// NewDB instantiates new [sqlx.DB] connection by creating test container.
func NewDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := t.Context()
	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	sqlDB, err := db.New(&db.Config{DSN: connStr})
	require.NoError(t, err)

	return sqlDB
}

func TestMigrate(t *testing.T) {
	t.Parallel()

	sqlDB := NewDB(t)

	err := db.Migrate(t.Context(), sqlDB)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, sqlDB.Close())
	})
}

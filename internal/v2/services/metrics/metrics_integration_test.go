//go:build integration

package metrics_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/adapters/clickhousedb"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	connectrepo "github.com/ship-monitor/cloud/internal/v2/repositories/connect"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/ship-monitor/cloud/internal/v2/repositories/devicemetrics"
	"github.com/ship-monitor/cloud/internal/v2/repositories/sensormetrics"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	metricsservice "github.com/ship-monitor/cloud/internal/v2/services/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

func setupMetricsPostgres(t *testing.T) *sqlx.DB {
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

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	sqlDB, err := db.New(&db.Config{DSN: dsn})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.Migrate(ctx, sqlDB))

	return sqlDB
}

func setupMetricsClickHouse(t *testing.T) *clickhousedb.Clickhouse {
	t.Helper()

	ctx := t.Context()

	request := testcontainers.ContainerRequest{
		Image: "clickhouse/clickhouse-server:25.8",
		Env: map[string]string{
			"CLICKHOUSE_DB":       "test",
			"CLICKHOUSE_USER":     "test",
			"CLICKHOUSE_PASSWORD": "test",
		},
		ExposedPorts: []string{"9000/tcp", "8123/tcp"},
		WaitingFor: wait.ForHTTP("/ping").
			WithPort("8123/tcp").WithStartupTimeout(2 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			Started:          true,
			ContainerRequest: request,
		},
	)
	if container != nil {
		testcontainers.CleanupContainer(t, container)
	}

	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("clickhouse://test:test@%s/test?dial_timeout=5s",
		net.JoinHostPort(host, port.Port()))
	click, err := clickhousedb.New(&clickhousedb.Config{DSN: dsn}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, click.Conn().Close()) })
	require.NoError(t, click.Migrate(ctx))

	return click
}

func createMetricsTestUser(t *testing.T, sqlDB *sqlx.DB) auth.Actor {
	t.Helper()

	id := uuid.New()
	_, err := sqlDB.ExecContext(t.Context(), `
		INSERT INTO users (id, name, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		id, "test user", id.String()+"@example.test", "test hash", time.Now(),
	)
	require.NoError(t, err)

	return auth.Actor{UserID: id}
}

func TestMetricsIsolationAfterDeviceReconnectionIntegration(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	sqlDB := setupMetricsPostgres(t)
	click := setupMetricsClickHouse(t)
	devices := devicerepo.New(sqlDB)
	requests := connectrepo.New(sqlDB)
	deviceService := deviceservice.New(devices, nil, nil, requests)
	service := metricsservice.New(deviceService,
		devicemetrics.New(click.Conn()), sensormetrics.New(click.Conn()),
		zap.NewNop(),
	)
	oldOwner := createMetricsTestUser(t, sqlDB)
	newOwner := createMetricsTestUser(t, sqlDB)

	for _, kind := range []string{"device", "sensor"} {
		t.Run(kind, func(t *testing.T) {
			deviceID, sensorID := uuid.New(), uuid.New()
			connect := func(owner auth.Actor) {
				t.Helper()

				request := domain.ConnectRequest{
					ID: uuid.New(), DeviceID: deviceID,
					UserID: owner.UserID, CreatedAt: time.Now(),
				}
				require.NoError(t, requests.Create(ctx, request))
				require.NoError(
					t,
					deviceService.ApproveConnection(ctx, deviceID, request.ID),
				)
			}
			query := metrics.Query{Name: "temperature", Limit: 10}
			oldSample := metrics.Metric{
				Name:      "temperature",
				Type:      metrics.TypeGauge,
				Unit:      "C",
				Value:     11,
				Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}
			newSample := metrics.Metric{
				Name:      "temperature",
				Type:      metrics.TypeGauge,
				Unit:      "C",
				Value:     22,
				Timestamp: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			}
			ingest := func(sample metrics.Metric) error {
				if kind == "device" {
					return service.IngestDeviceMetrics(
						ctx,
						deviceID,
						[]metrics.Metric{sample},
					)
				}

				return service.IngestSensorMetrics(
					ctx,
					deviceID,
					sensorID,
					[]metrics.Metric{sample},
				)
			}
			get := func(owner auth.Actor) ([]metrics.Metric, error) {
				if kind == "device" {
					return service.GetDeviceMetrics(ctx, owner, deviceID, query)
				}

				return service.GetSensorMetrics(
					ctx,
					owner,
					deviceID,
					sensorID,
					query,
				)
			}

			connect(oldOwner)
			require.NoError(t, ingest(oldSample))

			got, err := get(oldOwner)
			require.NoError(t, err)
			require.Equal(t, []metrics.Metric{oldSample}, got)

			_, err = get(newOwner)
			require.ErrorIs(t, err, domain.ErrNotFound)

			require.NoError(t, deviceService.Remove(ctx, oldOwner, deviceID))
			_, err = get(oldOwner)
			require.ErrorIs(t, err, domain.ErrNotFound)
			_, err = get(newOwner)
			require.ErrorIs(t, err, domain.ErrNotFound)

			connect(newOwner)

			_, err = get(oldOwner)
			require.ErrorIs(t, err, domain.ErrNotFound)
			got, err = get(newOwner)
			require.NoError(t, err)
			assert.Empty(
				t,
				got,
				"new owner must not see previous owner's metrics",
			)

			require.NoError(t, ingest(newSample))

			got, err = get(newOwner)
			require.NoError(t, err)
			assert.Equal(t, []metrics.Metric{newSample}, got,
				"new owner must see only metrics from the new ownership")

			_, err = get(oldOwner)
			assert.ErrorIs(t, err, domain.ErrNotFound)
		})
	}
}

//go:build integration

package sensor_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ship-monitor/cloud/internal/v2/adapters/db"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	sensorrepo "github.com/ship-monitor/cloud/internal/v2/repositories/sensor"
	"github.com/ship-monitor/cloud/internal/v2/services/sensor"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.uber.org/zap"
)

func sensorTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	container, err := postgres.Run(t.Context(),
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

	dsn, err := container.ConnectionString(t.Context(), "sslmode=disable")
	require.NoError(t, err)
	database, err := db.New(&db.Config{DSN: dsn})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := database.Close()
		if err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	require.NoError(t, db.Migrate(t.Context(), database))

	return database
}

//nolint:exhaustruct_v5,goconst
func TestDeclareSensors_RollsBackFailedDeclaration(t *testing.T) {
	t.Parallel()

	database := sensorTestDB(t)
	repo := sensorrepo.New(database)
	service := sensor.New(repo, nil, zap.NewNop())
	deviceID := uuid.New()
	original := domain.Sensor{
		ID: uuid.New(), DeviceID: deviceID, Name: "old", DisplayName: "old",
		Manufacturer: "maker", Model: "model", Revision: "rev", Software: "v1",
	}
	require.NoError(t, repo.InsertSensor(t.Context(), &original))
	stored, err := repo.GetSensors(t.Context(), deviceID)
	require.NoError(t, err)
	require.Equal(t, []domain.Sensor{original}, stored)

	// A test-only constraint makes the database reject one otherwise valid
	// item.
	_, err = database.ExecContext(t.Context(), `ALTER TABLE sensors
		ADD CONSTRAINT test_reject_sensor CHECK (name <> 'blocked-by-test')`)
	require.NoError(t, err)

	first := domain.SensorDecl{ID: uuid.New(), Name: "valid"}
	invalid := domain.SensorDecl{ID: uuid.New(), Name: "blocked-by-test"}
	err = service.DeclareSensors(
		t.Context(), deviceID, []domain.SensorDecl{first, invalid},
	)

	var pgErr *pq.Error
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, pq.ErrorCode("23514"), pgErr.Code)
	require.Equal(t, "test_reject_sensor", pgErr.Constraint)
	stored, err = repo.GetSensors(t.Context(), deviceID)
	require.NoError(t, err)
	require.Equal(t, []domain.Sensor{original}, stored,
		"an error must leave the entire previous declaration intact")
}

//nolint:exhaustruct_v5
func TestDeclareSensors_EmptyAndDuplicate(t *testing.T) {
	t.Parallel()

	database := sensorTestDB(t)
	repo := sensorrepo.New(database)
	service := sensor.New(repo, nil, zap.NewNop())
	deviceID := uuid.New()
	otherID := uuid.New()
	original := domain.Sensor{
		ID: uuid.New(), DeviceID: deviceID, Name: "old", DisplayName: "old",
	}
	other := domain.Sensor{
		ID: uuid.New(), DeviceID: otherID, Name: "other", DisplayName: "other",
	}

	require.NoError(t, repo.InsertSensor(t.Context(), &original))
	require.NoError(t, repo.InsertSensor(t.Context(), &other))

	duplicate := domain.SensorDecl{ID: uuid.New(), Name: "duplicate"}
	err := service.DeclareSensors(
		t.Context(), deviceID, []domain.SensorDecl{duplicate, duplicate},
	)
	require.Error(t, err)
	stored, err := repo.GetSensors(t.Context(), deviceID)
	require.NoError(t, err)
	require.Equal(t, []domain.Sensor{original}, stored)

	require.NoError(t, service.DeclareSensors(
		t.Context(), deviceID, []domain.SensorDecl{},
	))
	stored, err = repo.GetSensors(t.Context(), deviceID)
	require.NoError(t, err)
	require.Empty(t, stored)
	stored, err = repo.GetSensors(t.Context(), otherID)
	require.NoError(t, err)
	require.Equal(t, []domain.Sensor{other}, stored)
}

// snapshotBarrier forces both reads to overlap when there is no serialization.
// A timeout permits a correctly serialized implementation to make progress.
type snapshotBarrier struct {
	sensor.Repo

	mu            sync.Mutex
	reads         int
	firstReleased bool
	overlapped    bool
	release       chan struct{}
}

//nolint:wrapcheck
func (r *snapshotBarrier) GetSensors(
	ctx context.Context,
	deviceID uuid.UUID,
) ([]domain.Sensor, error) {
	items, err := r.Repo.GetSensors(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()

	r.reads++

	read := r.reads
	if read == 2 {
		r.overlapped = !r.firstReleased
		close(r.release)
	}
	r.mu.Unlock()

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-r.release:
	case <-timer.C:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if read == 1 {
		r.mu.Lock()
		r.firstReleased = true
		r.mu.Unlock()
	}

	if err != nil {
		return nil, err
	}

	return items, nil
}

//nolint:exhaustruct_v5,funlen
func TestDeclareSensors_ConcurrentDeclarationsAreAtomic(t *testing.T) {
	t.Parallel()

	database := sensorTestDB(t)
	repo := sensorrepo.New(database)
	barrier := &snapshotBarrier{Repo: repo, release: make(chan struct{})}
	service := sensor.New(barrier, nil, zap.NewNop())
	deviceID := uuid.New()
	declarations := []domain.SensorDecl{
		{ID: uuid.New(), Name: "first"},
		{ID: uuid.New(), Name: "second"},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	start := make(chan struct{})

	type result struct {
		index int
		err   error
	}

	results := make(chan result, len(declarations))
	for index, declaration := range declarations {
		go func(index int, item domain.SensorDecl) {
			<-start

			err := service.DeclareSensors(
				ctx,
				deviceID,
				[]domain.SensorDecl{item},
			)
			results <- result{index: index, err: err}
		}(index, declaration)
	}

	close(start)

	errs := make([]error, len(declarations))
	for range declarations {
		outcome := <-results
		errs[outcome.index] = outcome.err
	}

	barrier.mu.Lock()
	overlapped := barrier.overlapped
	barrier.mu.Unlock()

	if !overlapped {
		t.Log("reads did not overlap; declarations may have been serialized")
	}

	stored, err := repo.GetSensors(t.Context(), deviceID)
	require.NoError(t, err)
	require.Len(t, stored, 1,
		"concurrent full declarations must not produce a mixed sensor set")

	matchedSuccess := false

	for i, declaration := range declarations {
		if stored[0].ID == declaration.ID && errs[i] == nil {
			matchedSuccess = true
		}
	}

	require.True(t, matchedSuccess,
		"stored sensor must be from a successful declaration: %v", errs)
}

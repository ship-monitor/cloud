package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	metricsservice "github.com/ship-monitor/cloud/internal/v2/services/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type ownershipFixture struct {
	deviceID uuid.UUID
	ownerID  uuid.UUID
	present  bool
}

func (f *ownershipFixture) Get(
	_ context.Context, actor auth.Actor, id uuid.UUID,
) (device.Device, error) {
	if !f.present || id != f.deviceID || actor.UserID != f.ownerID {
		return device.Device{}, deviceservice.ErrNotFound
	}

	return device.Device{ID: id, OwnerID: f.ownerID}, nil
}

type deviceMetricFixture struct {
	rows map[uuid.UUID][]metrics.Metric
}

func (f *deviceMetricFixture) InsertBatch(
	_ context.Context, id uuid.UUID, samples []metrics.Metric,
) error {
	f.rows[id] = append(f.rows[id], samples...)

	return nil
}

func (f *deviceMetricFixture) List(
	_ context.Context, id uuid.UUID, _ metrics.Query,
) ([]metrics.Metric, error) {
	return f.rows[id], nil
}

type sensorMetricKey struct {
	deviceID uuid.UUID
	sensorID uuid.UUID
}

type sensorMetricFixture struct {
	rows map[sensorMetricKey][]metrics.Metric
}

func (f *sensorMetricFixture) InsertBatch(
	_ context.Context, deviceID, sensorID uuid.UUID, samples []metrics.Metric,
) error {
	key := sensorMetricKey{deviceID, sensorID}
	f.rows[key] = append(f.rows[key], samples...)

	return nil
}

func (f *sensorMetricFixture) List(
	_ context.Context, deviceID uuid.UUID, sensorID sensor.ID, _ metrics.Query,
) ([]metrics.Metric, error) {
	return f.rows[sensorMetricKey{deviceID, sensorID}], nil
}

func TestMetricsAuthorizationAfterDeviceReconnection(t *testing.T) {
	for _, kind := range []string{"device", "sensor"} {
		t.Run(kind, func(t *testing.T) {
			ctx := t.Context()
			deviceID, sensorID := uuid.New(), uuid.New()
			oldOwner := auth.Actor{UserID: uuid.New()}
			newOwner := auth.Actor{UserID: uuid.New()}
			owners := &ownershipFixture{
				deviceID: deviceID, ownerID: oldOwner.UserID, present: true,
			}
			deviceRows := &deviceMetricFixture{
				rows: make(map[uuid.UUID][]metrics.Metric),
			}
			sensorRows := &sensorMetricFixture{
				rows: make(map[sensorMetricKey][]metrics.Metric),
			}
			service := metricsservice.New(
				owners,
				deviceRows,
				sensorRows,
				zap.NewNop(),
			)
			oldSample := metrics.Metric{
				Name:      "temperature",
				Value:     1,
				Timestamp: time.Unix(1, 0),
			}
			newSample := metrics.Metric{
				Name:      "temperature",
				Value:     2,
				Timestamp: time.Unix(2, 0),
			}
			query := metrics.Query{Name: "temperature", Limit: 10}

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
			get := func(actor auth.Actor) ([]metrics.Metric, error) {
				if kind == "device" {
					return service.GetDeviceMetrics(ctx, actor, deviceID, query)
				}

				return service.GetSensorMetrics(
					ctx,
					actor,
					deviceID,
					sensorID,
					query,
				)
			}

			require.NoError(t, ingest(oldSample))

			got, err := get(oldOwner)
			require.NoError(t, err)
			require.Equal(t, []metrics.Metric{oldSample}, got)

			// Simulate deletion and reconnecting the same hardware ID to a
			// different owner while the metrics store retains its history.
			owners.present = false
			_, err = get(oldOwner)
			require.ErrorIs(t, err, deviceservice.ErrNotFound)

			owners.ownerID, owners.present = newOwner.UserID, true

			require.NoError(t, ingest(newSample))

			_, err = get(oldOwner)
			require.ErrorIs(t, err, deviceservice.ErrNotFound)
			got, err = get(newOwner)
			require.NoError(t, err)
			assert.Contains(t, got, newSample)
			// The fake deliberately retains old samples; it cannot model a
			// real deletion that purges history from the metrics store.
		})
	}
}

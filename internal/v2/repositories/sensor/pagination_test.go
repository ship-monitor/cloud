//go:build integration

package sensor_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	domainsensor "github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/internal/v2/repositories/sensor"
	"github.com/stretchr/testify/require"
)

func TestRepository_CursorPages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	sensors := sensor.New(db)
	firstID, foreignID := uuid.UUID{1}, uuid.UUID{2}
	firstSensor := domainsensor.Sensor{
		ID: uuid.UUID{1}, DeviceID: firstID, Name: "", DisplayName: "",
		Manufacturer: "", Model: "", Revision: "", Software: "",
	}
	lastSensor := firstSensor
	lastSensor.ID = uuid.UUID{3}
	other := firstSensor
	other.ID = uuid.UUID{2}
	other.DeviceID = foreignID

	for _, item := range []domainsensor.Sensor{
		lastSensor, other, firstSensor,
	} {
		require.NoError(t, sensors.InsertSensor(t.Context(), &item))
	}

	list := func(ctx context.Context, req pagination.Request[domainsensor.ID]) (
		pagination.Page[domainsensor.Sensor, domainsensor.ID], error,
	) {
		return sensors.GetSensorsPage(ctx, firstID, req)
	}
	checkPages(
		t,
		list,
		[]domainsensor.Sensor{firstSensor, lastSensor},
		func(s domainsensor.Sensor) domainsensor.ID { return s.ID },
	)
}

package sensor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
)

// ErrNotFound is returned when no sensor matches the given ID and device.
var ErrNotFound = errors.New("sensor not found")

type Repo struct {
	db *sqlx.DB
}

type row struct {
	ID           string     `db:"id"`
	Name         string     `db:"name"`
	DeviceID     uuid.UUID  `db:"device_id"`
	DisplayName  string     `db:"display_name"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
	Manufacturer string     `db:"manufacturer"`
	Model        string     `db:"model"`
	Revision     string     `db:"revision"`
	Software     string     `db:"software"`
}

func (r row) toDomain() sensor.Sensor {
	return sensor.Sensor{
		ID:           r.ID,
		Name:         r.Name,
		DeviceID:     r.DeviceID,
		DisplayName:  r.DisplayName,
		Manufacturer: r.Manufacturer,
		Model:        r.Model,
		Revision:     r.Revision,
		Software:     r.Software,
	}
}

const columns = "id, name, device_id, display_name, created_at, updated_at, manufacturer, model, revision, software"

func New(
	db *sqlx.DB,
) *Repo {
	return &Repo{
		db: db,
	}
}

func (s *Repo) InsertSensor(
	ctx context.Context,
	item *sensor.Sensor,
) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sensors (`+columns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		item.ID,
		item.Name,
		item.DeviceID,
		item.DisplayName,
		time.Now().UTC(),
		time.Now().UTC(),
		item.Manufacturer,
		item.Model,
		item.Revision,
		item.Software,
	)
	if err != nil {
		return fmt.Errorf("execute insert: %w", err)
	}

	return nil
}

func (s *Repo) GetSensors(
	ctx context.Context,
	deviceID uuid.UUID,
) ([]sensor.Sensor, error) {
	var rows []row

	err := s.db.SelectContext(
		ctx,
		&rows,
		`SELECT `+columns+` FROM sensors WHERE device_id = $1 ORDER BY id`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("execute select: %w", err)
	}

	sensors := make([]sensor.Sensor, len(rows))
	for i, r := range rows {
		sensors[i] = r.toDomain()
	}

	return sensors, nil
}

func (s *Repo) GetSensor(
	ctx context.Context,
	sensorID string,
	deviceID uuid.UUID,
) (*sensor.Sensor, error) {
	var record row

	err := s.db.GetContext(
		ctx,
		&record,
		`SELECT `+columns+` FROM sensors WHERE id = $1 AND device_id = $2`,
		sensorID,
		deviceID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("select sensor: %w", err)
	}

	result := record.toDomain()

	return &result, nil
}

// RenameSensor updates the display name of a sensor belonging to deviceID.
// It returns ErrNotFound if the sensor does not exist for that device.
func (s *Repo) RenameSensor(
	ctx context.Context,
	sensorID string,
	deviceID uuid.UUID,
	displayName string,
) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE sensors SET display_name = $1 WHERE id = $2 AND device_id = $3`,
		displayName, sensorID, deviceID,
	)
	if err != nil {
		return fmt.Errorf("rename sensor: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rename sensor: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteSensor removes a sensor belonging to deviceID.
// It returns ErrNotFound if the sensor does not exist for that device.
func (s *Repo) DeleteSensor(
	ctx context.Context,
	sensorID string,
	deviceID uuid.UUID,
) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM sensors WHERE id = $1 AND device_id = $2`,
		sensorID, deviceID,
	)
	if err != nil {
		return fmt.Errorf("delete sensor: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete sensor: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

// SensorExists reports whether a sensor with the given ID exists on the given
// device.
func (s *Repo) SensorExists(
	ctx context.Context,
	sensorID string,
	deviceID uuid.UUID,
) (bool, error) {
	var exists bool

	err := s.db.
		QueryRowxContext(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM sensors WHERE id = $1 AND device_id = $2)`,
			sensorID,
			deviceID,
		).
		Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check sensor exists: %w", err)
	}

	return exists, nil
}

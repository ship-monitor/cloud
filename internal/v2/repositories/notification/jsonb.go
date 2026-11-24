package notification

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

var errTypeAssert = errors.New("type assertion to []byte failed")

// JSONB is a wrapper for dynamic data structures.
// Currently not supported arrays and integer values.
//
//nolint:recvcheck
type JSONB map[string]any

// Scan implements the [sql.Scanner] interface to decode JSONB into
// map[string]any.
func (m *JSONB) Scan(value any) error {
	if value == nil {
		*m = nil

		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errTypeAssert
	}

	err := json.Unmarshal(bytes, m)
	if err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	return nil
}

// Value implements the [driver.Valuer] interface to encode map[string]any into
// JSONB.
func (m JSONB) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil //nolint:nilnil
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	return data, nil
}

package domain

import "time"

type StateRecord struct {
	DeviceID  string    `json:"deviceId" validate:"required"`
	State     string    `json:"state" validate:"required"`
	Timestamp time.Time `json:"timestamp" validate:"required"`
	Value     any       `json:"value" validate:"required"`
}

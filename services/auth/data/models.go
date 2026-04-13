package data

import "time"

type TimestampsModel struct {
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

func (m *TimestampsModel) UpdateTimestamp() {
	m.UpdatedAt = time.Now()
}

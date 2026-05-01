package data

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type OrganizationDevice struct {
	*bun.BaseModel `bun:"table:organization_devices"`

	ID             uuid.UUID `bun:",pk,type:varchar" json:"id"`
	OrganizationID uuid.UUID `bun:",notnull,type:varchar" json:"organizationId"`
	CreatedAt      time.Time `bun:",nullzero,notnull" json:"createdAt"`
}

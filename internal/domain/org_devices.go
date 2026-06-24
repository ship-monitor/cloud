package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const DefaultOrganizationDeviceName = "Unknown Device"

type OrganizationDevice struct {
	*bun.BaseModel `bun:"table:organization_devices"`

	ID             uuid.UUID `bun:",pk,type:varchar"      json:"id"`
	OrganizationID uuid.UUID `bun:",notnull,type:varchar" json:"organizationId"`
	CreatedAt      time.Time `bun:",nullzero,notnull"     json:"createdAt"`
	Name           string    `bun:",notnull,default"      json:"name"`
}

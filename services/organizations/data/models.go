package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

type Organization struct {
	*bun.BaseModel `bun:"table:organization"`

	ID        uuid.UUID `bun:",pk,type:uuid,default:gen_random_uuid()" json:"id"`
	Name      string    `bun:",notnull" json:"name"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	CreatorID uuid.UUID `bun:",notnull,type:uuid" json:"creatorId"`
}
type OrganizationMember struct {
	*bun.BaseModel `bun:"table:organization_member"`
	MemberID       uuid.UUID `bun:",notnull,type:uuid" json:"memberId"`
	OrganizationID uuid.UUID `bun:",notnull,type:uuid" json:"organizationId"`
}

func GetOrganization(id uuid.UUID) (*Organization, error) {
	var org Organization
	err := db.DB.NewSelect().Model(&org).Where("id = ?", id).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &org, nil
}

type CreateOrganizationInput struct {
	Name      string    `json:"name"`
	CreatorID uuid.UUID `json:"creatorId"`
}

func CreateOrganization(in CreateOrganizationInput) (Organization, error) {
	org := Organization{
		CreatedAt: time.Now(),
		Name:      in.Name,
		CreatorID: in.CreatorID,
	}
	_, err := db.DB.NewInsert().Model(&org).Exec(context.Background())
	return org, err
}

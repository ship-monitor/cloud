package data

import (
	"context"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

type Organization struct {
	*bun.BaseModel `bun:"table:organizations"`

	ID                  uuid.UUID            `bun:",pk,type:varchar"      json:"id"`
	Name                string               `bun:",notnull"              json:"name"`
	CreatedAt           time.Time            `bun:",nullzero,notnull"     json:"createdAt"`
	UpdatedAt           time.Time            `bun:",nullzero"     	 json:"updatedAt"`
	CreatorID           uuid.UUID            `bun:",notnull,type:varchar" json:"creatorId"`
	OrganizationMembers []OrganizationMember `bun:"rel:has-many,join:id=organization_id"`
}

// Role участника в организации.
type Role string

const (
	RoleOwner         Role = "owner"
	RoleAdministrator Role = "administrator"
	RoleMember        Role = "member"
)

// OrganizationMember — связь пользователя с организацией.
type OrganizationMember struct {
	*bun.BaseModel `bun:"table:organization_members"`

	MemberID       uuid.UUID `bun:",notnull,type:varchar" json:"memberId"`
	OrganizationID uuid.UUID `bun:",notnull,type:varchar" json:"organizationId"`
	Role           Role      `bun:",notnull"              json:"role"`
	JoinedAt       time.Time `bun:",nullzero,notnull"     json:"joinedAt"`
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
	Name      string    `json:"name"      validate:"required"`
	CreatorID uuid.UUID `json:"creatorId" validate:"required"`
}

func CreateOrganization(in CreateOrganizationInput) (Organization, error) {
	log.Debug("Creating organization", "name", in.Name, "creatorId", in.CreatorID)
	err := validator.New().Struct(in)
	if err != nil {
		return Organization{}, fmt.Errorf("failed validate input: %s", err)
	}

	org := Organization{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      in.Name,
		CreatorID: in.CreatorID,
	}
	_, err = db.DB.NewInsert().Model(&org).Exec(context.Background())

	return org, err
}

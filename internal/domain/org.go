package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Organization struct {
	*bun.BaseModel `bun:"table:organizations"`

	ID                  uuid.UUID            `bun:",pk,type:varchar" json:"id"`
	Name                string               `bun:",notnull" json:"name"`
	CreatedAt           time.Time            `bun:",nullzero,notnull" json:"createdAt"`
	UpdatedAt           time.Time            `bun:",nullzero" json:"updatedAt"`
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

// OrganizationMemberWithUser is an organization member enriched with user
// profile data.
type OrganizationMemberWithUser struct {
	MemberID       uuid.UUID `bun:"member_id"       json:"memberId"`
	OrganizationID uuid.UUID `bun:"organization_id" json:"organizationId"`
	Role           Role      `bun:"role"            json:"role"`
	JoinedAt       time.Time `bun:"joined_at"       json:"joinedAt"`
	Name           string    `bun:"name"            json:"name"`
	Email          string    `bun:"email"           json:"email"`
}

package dto

import (
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data" 
)

// --- Invitation DTOs -----------------------------------------------------

// CreateInvitationRequest is the payload for creating an invitation.
type CreateInvitationRequest struct {
	InviteeEmail string `json:"inviteeEmail" binding:"required"`
}

// InvitationResponse represents an invitation in API responses.
type InvitationResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	InviteeEmail   string    `json:"inviteeEmail"`
	Token          string    `json:"token"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

// ListInvitationsResponse wraps a list of invitations.
type ListInvitationsResponse struct {
	Invitations []InvitationResponse `json:"invitations"`
}

type OrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateOrganizationResponse struct {
	Success      bool                  `json:"success"`
	Error        string                `json:"error,omitempty"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
}

type GetOrganizationResponse struct {
	*OrganizationResponse
}

type UpdateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateOrganizationResponse struct {
	Success      bool                  `json:"success"`
	Error        string                `json:"error,omitempty"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
}
// --- Member management DTOs -----------------------------------------------

// AddMemberRequest — запрос на добавление участника
type AddMemberRequest struct {
	UserID uuid.UUID `json:"userId" binding:"required"`
	Role   data.Role `json:"role" binding:"required"`
}

// UpdateMemberRoleRequest — запрос на изменение роли
type UpdateMemberRoleRequest struct {
	Role data.Role `json:"role" binding:"required"`
}
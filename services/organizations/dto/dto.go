package dto

import (
	"time"

	"github.com/google/uuid"
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

package dto

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateOrganizationRequest struct {
	Name string `json:"name"`
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
	Name string `json:"name"`
}

type UpdateOrganizationResponse struct {
	Success      bool                  `json:"success"`
	Error        string                `json:"error,omitempty"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
}

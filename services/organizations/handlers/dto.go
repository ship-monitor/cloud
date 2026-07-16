package handlers

import (
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type ErrorResponse = requests.BadResponse

type ValidationError struct {
	ActualTag   string `json:"actualTag"`
	Error       string `json:"error"`
	Field       string `json:"field"`
	StructField string `json:"structField"`
	Tag         string `json:"tag"`
}

func Error(err error) requests.BadResponse {
	return requests.ResponseErr(err)
}

// --- Organization DTOs ---

type CreateOrganizationResponse struct {
	OrganizationID uuid.UUID `json:"organizationId"`
}

type OrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateOrganizationRequest struct {
	Name string `binding:"required" json:"name"`
}

type UpdateOrganizationRequest struct {
	Name string `binding:"required" json:"name"`
}

// --- Member DTOs ---

type AddMemberRequest struct {
	UserID uuid.UUID `binding:"required" json:"userId"`
	Role   string    `binding:"required" json:"role"`
}

type UpdateMemberRoleRequest struct {
	Role string `binding:"required" json:"role"`
}

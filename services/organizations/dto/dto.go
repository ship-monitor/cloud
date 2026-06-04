package dto

import (
	"time"

	"github.com/google/uuid"
)

type ErrorResponse struct {
	Details          string            `json:"details"`
	ValidationErrors []ValidationError `json:"validationErrors,omitempty"`
}

type ValidationError struct {
	ActualTag   string `json:"actualTag"`
	Error       string `json:"error"`
	Field       string `json:"field"`
	StructField string `json:"structField"`
	Tag         string `json:"tag"`
}

func Error(err error) ErrorResponse {
	return ErrorResponse{Details: err.Error()}
}

// --- Organization DTOs ---

type OrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

// --- Member DTOs ---

type AddMemberRequest struct {
	UserID uuid.UUID `json:"userId" binding:"required"`
	Role   string    `json:"role" binding:"required"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// --- Invitation DTOs ---

type CreateInvitationRequest struct {
	InviteeEmail string `json:"inviteeEmail" binding:"required"`
}

type CreateInvitationBulkRequest struct {
	InviteeEmails []string `json:"inviteeEmails" binding:"required"`
}

type InvitationResponse struct {
	ID               uuid.UUID `json:"id"`
	OrganizationID   uuid.UUID `json:"organizationId"`
	OrganizationName string    `json:"organizationName,omitempty"`
	InviteeEmail     string    `json:"inviteeEmail"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type ListInvitationsResponse struct {
	Invitations []InvitationResponse `json:"invitations"`
}

// --- Device DTOs ---

type ConnectDeviceRequest struct {
	DeviceID uuid.UUID `json:"deviceId" binding:"required"`
	Name     string    `json:"name"`
}

type SendCommandRequest struct {
	Command string                 `json:"command" binding:"required"`
	Args    map[string]interface{} `json:"args,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type DeviceResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	CreatedAt      time.Time `json:"createdAt"`
	Name           string    `json:"name"`
	IsConnected    bool      `json:"isConnected"`
}

type SendCommandResponse struct {
	RequestError string         `json:"requestError,omitempty"`
	CommandError string         `json:"commandError,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
}

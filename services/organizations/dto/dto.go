package dto

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

// --- Invitation DTOs ---

type CreateInvitationRequest struct {
	InviteeEmail string `binding:"required" json:"inviteeEmail"`
}

type CreateInvitationBulkRequest struct {
	InviteeEmails []string `binding:"required" json:"inviteeEmails"`
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

type UpdateDeviceRequest struct {
	Name string `binding:"required" json:"name"`
}

type ConnectDeviceRequest struct {
	DeviceID uuid.UUID `binding:"required" json:"deviceId"`
	Name     string    `json:"name"`
}

type SendCommandRequest struct {
	Command string         `binding:"required" json:"command"`
	Args    map[string]any `json:"args,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
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

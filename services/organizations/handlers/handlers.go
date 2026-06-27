package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

type OrganizationID = uuid.UUID

type OrganizationService interface {
	CreateOrganization(
		ctx context.Context,
		name string,
		creatorID uuid.UUID,
	) (OrganizationID, error)
	GetOrganization(
		ctx context.Context,
		id OrganizationID,
		userID uuid.UUID,
	) (*domain.Organization, error)
	GetUsersOrganizations(
		ctx context.Context,
		userID uuid.UUID,
		page int,
	) ([]*domain.Organization, error)
	RenameOrganization(
		ctx context.Context,
		organizationID uuid.UUID,
		name string,
		userID uuid.UUID,
	) error
	DeleteOrganization(ctx context.Context, organizationID, userID uuid.UUID) error
	GetMembers(
		ctx context.Context,
		organizationID uuid.UUID,
		userID uuid.UUID,
	) ([]domain.OrganizationMemberWithUser, error)
	AddMember(
		ctx context.Context,
		organizationID uuid.UUID,
		actorID uuid.UUID,
		userID uuid.UUID,
		role domain.Role,
	) (*domain.OrganizationMember, error)
	UpdateMemberRole(
		ctx context.Context,
		organizationID uuid.UUID,
		actorID uuid.UUID,
		userID uuid.UUID,
		role domain.Role,
	) error
	RemoveMember(
		ctx context.Context,
		organizationID uuid.UUID,
		actorID uuid.UUID,
		userID uuid.UUID,
	) error
	CreateInvitation(
		ctx context.Context,
		organizationID uuid.UUID,
		inviterID uuid.UUID,
		inviteeEmail string,
	) (*services.InvitationDetails, error)
	ListMyInvitations(ctx context.Context, email string) ([]services.InvitationDetails, error)
	ListOrgInvitations(
		ctx context.Context,
		organizationID uuid.UUID,
		userID uuid.UUID,
	) ([]services.InvitationDetails, error)
	AcceptInvitation(
		ctx context.Context,
		invitationID uuid.UUID,
		userID uuid.UUID,
		userEmail string,
	) error
	DeclineInvitation(ctx context.Context, invitationID uuid.UUID, userEmail string) error
}

type OrgDevicesService interface {
	ConnectDevice(
		ctx context.Context,
		deviceID, organizationID, userID uuid.UUID,
		name string,
	) error
	DisconnectDevice(ctx context.Context, deviceID, userID uuid.UUID) error
	RenameDevice(ctx context.Context, deviceID, userID uuid.UUID, name string) error
	GetDevice(ctx context.Context, deviceID, userID uuid.UUID) (*domain.OrganizationDevice, error)
	GetDevices(
		ctx context.Context,
		organizationID, userID uuid.UUID,
	) ([]domain.OrganizationDevice, error)
}

type HTTPHandler struct {
	orgs    OrganizationService
	devices OrgDevicesService
}

func New(orgs OrganizationService, devices OrgDevicesService) *HTTPHandler {
	return &HTTPHandler{
		orgs:    orgs,
		devices: devices,
	}
}

func (h *HTTPHandler) HandleCreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid create organization request", "error", err)
		c.JSON(http.StatusBadRequest, dto.Error(fmt.Errorf("invalid request: %w", err)))

		return
	}

	session := auth.GetSession(c)

	id, err := h.orgs.CreateOrganization(c.Request.Context(), req.Name, session.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusCreated, dto.CreateOrganizationResponse{
		OrganizationID: id,
	})
}

func (h *HTTPHandler) HandleGetMyOrganizations(c *gin.Context) {
	session := auth.GetSession(c)

	orgs, err := h.orgs.GetUsersOrganizations(c.Request.Context(), session.UserID, 0)
	if err != nil {
		log.Error("Failed to get organizations for member", "error", err, "user", session.UserID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	resp := make([]dto.OrganizationResponse, 0, len(orgs))
	for _, org := range orgs {
		resp = append(resp, organizationToDTO(org))
	}

	c.JSON(http.StatusOK, gin.H{"organizations": resp})
}

func (h *HTTPHandler) HandleGetOrganization(c *gin.Context) {
	organizationID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	org, err := h.orgs.GetOrganization(
		c.Request.Context(),
		organizationID,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, dto.Error(err))

		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, organizationToDTO(org))
}

func (h *HTTPHandler) HandleUpdateOrganization(c *gin.Context) {
	id := requests.MustGetParamUUID(c, "id")

	var req dto.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))

		return
	}

	session := auth.GetSession(c)

	err := h.orgs.RenameOrganization(c.Request.Context(), id, req.Name, session.UserID)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		c.Status(http.StatusOK)
	}
}

func (h *HTTPHandler) HandleDeleteOrganization(c *gin.Context) {
	id := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	err := h.orgs.DeleteOrganization(c.Request.Context(), id, session.UserID)
	switch {
	case errors.Is(err, services.ErrOrganizationNotFound):
		c.JSON(http.StatusNotFound, dto.Error(err))
	case errors.Is(err, services.ErrOnlyOwnerCanDelete):
		c.JSON(http.StatusForbidden, dto.Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.Status(http.StatusOK)
	}
}

func (h *HTTPHandler) HandleGetMembers(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	members, err := h.orgs.GetMembers(c.Request.Context(), orgID, session.UserID)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.JSON(http.StatusOK, gin.H{"members": members})
	}
}

func (h *HTTPHandler) HandleAddMember(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))

		return
	}

	member, err := h.orgs.AddMember(
		c.Request.Context(),
		orgID,
		session.UserID,
		req.UserID,
		domain.Role(req.Role),
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
	case errors.Is(err, services.ErrCannotAssignOwnerRole),
		errors.Is(err, services.ErrInvalidMemberRole):
		c.JSON(http.StatusBadRequest, dto.Error(err))
	case errors.Is(err, services.ErrMemberAlreadyExists):
		c.JSON(http.StatusConflict, dto.Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.JSON(http.StatusCreated, gin.H{
			"memberId":       member.MemberID,
			"organizationId": member.OrganizationID,
			"role":           member.Role,
			"joinedAt":       member.JoinedAt,
		})
	}
}

func (h *HTTPHandler) HandleUpdateMemberRole(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	userID := requests.MustGetParamUUID(c, "userId")

	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(fmt.Errorf("invalid request: %w", err)))

		return
	}

	session := auth.GetSession(c)

	err := h.orgs.UpdateMemberRole(
		c.Request.Context(),
		orgID,
		session.UserID,
		userID,
		domain.Role(req.Role),
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
	case errors.Is(err, services.ErrMemberNotFound):
		c.JSON(http.StatusNotFound, dto.Error(err))
	case errors.Is(err, services.ErrCannotChangeOwnerRole):
		c.JSON(http.StatusForbidden, dto.Error(err))
	case errors.Is(err, services.ErrCannotAssignOwnerRole),
		errors.Is(err, services.ErrInvalidMemberRole):
		c.JSON(http.StatusBadRequest, dto.Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.Status(http.StatusOK)
	}
}

func (h *HTTPHandler) HandleRemoveMember(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	userID := requests.MustGetParamUUID(c, "userId")
	session := auth.GetSession(c)

	err := h.orgs.RemoveMember(c.Request.Context(), orgID, session.UserID, userID)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
	case errors.Is(err, services.ErrMemberNotFound):
		c.JSON(http.StatusNotFound, dto.Error(err))
	case errors.Is(err, services.ErrNotAllowed),
		errors.Is(err, services.ErrRemovingOrganizationOwner):
		c.JSON(http.StatusForbidden, dto.Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.Status(http.StatusOK)
	}
}

func organizationToDTO(org *domain.Organization) dto.OrganizationResponse {
	return dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
}

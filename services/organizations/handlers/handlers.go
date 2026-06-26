package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
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
}
type OrgDevicesService interface {
	ConnectDevice(ctx context.Context, deviceID, organizationID, userID uuid.UUID) error
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

func (h *HTTPHandler) HandleGetOrganization(c *gin.Context) {
	organizationID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	org, err := h.orgs.GetOrganization(
		c.Request.Context(),
		organizationID,
		session.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	})
}

func HandleGetMyOrganizations(c *gin.Context) {
	session := auth.GetSession(c)

	orgs, err := data.GetOrganizationsByMember(session.UserID)
	if err != nil {
		log.Error("Failed to get organizations for member", "error", err, "user", session.UserID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	resp := make([]dto.OrganizationResponse, 0, len(orgs))
	for _, org := range orgs {
		resp = append(resp, dto.OrganizationResponse{
			ID:        org.ID,
			Name:      org.Name,
			CreatedAt: org.CreatedAt,
			UpdatedAt: org.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"organizations": resp})
}

func HandleGetOrganization(c *gin.Context) {
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("Invalid organization id in get request", "id", idStr)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, id)
	if err != nil {
		log.Error(
			"Failed to check membership",
			"error",
			err,
			"organization",
			id,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	if !isMember {
		log.Warn("Access denied for organization", "organization", id, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	org, err := data.GetOrganization(id)
	if err != nil {
		log.Warn("Organization not found", "organization", id, "user", session.UserID)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("organization not found")))

		return
	}

	c.JSON(http.StatusOK, dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	})
}

func HandleUpdateOrganization(c *gin.Context) {
	id := requests.MustGetParamUUID(c, "id")

	var req dto.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))

		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(id, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdministrator {
		c.JSON(
			http.StatusForbidden,
			dto.Error(errors.New("only owner or administrator can update organization")),
		)

		return
	}

	org, err := data.GetOrganization(id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("organization not found")))

		return
	}

	org, err = data.UpdateOrganization(org, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	})
}

func HandleDeleteOrganization(c *gin.Context) {
	id := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	org, err := data.GetOrganization(id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(err))

		return
	}

	if org.CreatorID != session.UserID {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner can delete organization")))

		return
	}

	if err := data.DeleteOrganization(id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.Status(http.StatusOK)
}

func HandleGetMembers(c *gin.Context) {
	idStr := c.Param("id")

	orgID, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("Invalid organization id in get members request", "id", idStr)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	if !isMember {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	members, err := data.GetMembersWithUserInfo(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

func HandleAddMember(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id ")
	session := auth.GetSession(c)

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))

		return
	}

	if _, err := data.GetMember(orgID, session.UserID); err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	if domain.Role(req.Role) == domain.RoleOwner {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("cannot assign owner role")))

		return
	}

	if domain.Role(req.Role) != domain.RoleAdministrator &&
		domain.Role(req.Role) != domain.RoleMember {
		c.JSON(
			http.StatusBadRequest,
			dto.Error(errors.New("invalid role, allowed: administrator, member")),
		)

		return
	}

	isMember, err := data.IsMember(req.UserID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	} else if isMember {
		c.JSON(http.StatusConflict, dto.Error(errors.New("user is already a member")))

		return
	}

	member := domain.OrganizationMember{
		MemberID:       req.UserID,
		OrganizationID: orgID,
		Role:           domain.Role(req.Role),
		JoinedAt:       time.Now(),
	}

	if err := data.AddMember(member); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"memberId":       member.MemberID,
		"organizationId": member.OrganizationID,
		"role":           member.Role,
		"joinedAt":       member.JoinedAt,
	})
}

func HandleUpdateMemberRole(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	userID := requests.MustGetParamUUID(c, "userId")

	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(fmt.Errorf("invalid request: %w", err)))

		return
	}

	session := auth.GetSession(c)

	_, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("member not found")))

		return
	}

	if targetMember.Role == domain.RoleOwner {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("cannot change owner role")))

		return
	}

	if domain.Role(req.Role) == domain.RoleOwner {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("cannot assign owner role")))

		return
	}

	if domain.Role(req.Role) != domain.RoleAdministrator &&
		domain.Role(req.Role) != domain.RoleMember {
		c.JSON(
			http.StatusBadRequest,
			dto.Error(errors.New("invalid role, allowed: administrator, member")),
		)

		return
	}

	if err := data.UpdateMemberRole(orgID, userID, domain.Role(req.Role)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.Status(http.StatusOK)
}

func HandleRemoveMember(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	userID := requests.MustGetParamUUID(c, "userId")
	session := auth.GetSession(c)

	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	if currentMember.Role != domain.RoleOwner && currentMember.Role != domain.RoleAdministrator {
		c.JSON(
			http.StatusForbidden,
			dto.Error(
				fmt.Errorf("only owner or administrator can remove members: %w", ErrNotAllowed),
			),
		)

		return
	}

	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(ErrMemberNotFound))

		return
	}

	if targetMember.Role == domain.RoleOwner {
		c.JSON(http.StatusForbidden, dto.Error(ErrRemovingOwner))

		return
	}

	if err := data.RemoveMember(orgID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.Status(http.StatusOK)
}

var (
	ErrNotAllowed     = errors.New("not allowed")
	ErrMemberNotFound = errors.New("member not found")
	ErrRemovingOwner  = errors.New("removing owner of organization")
)

package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
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
	) (*data.Organization, error)
	GetUsersOrganizations(
		ctx context.Context,
		userID uuid.UUID,
		page int,
	) ([]data.Organization, error)
}

type HTTPHandler struct {
	orgs OrganizationService
}

func New(orgs OrganizationService) *HTTPHandler {
	return &HTTPHandler{
		orgs: orgs,
	}
}

func (h *HTTPHandler) HandleCreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid create organization request", "error", err)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))

		return
	}

	session := auth.GetSession(c)

	id, err := h.orgs.CreateOrganization(c.Request.Context(), req.Name, session.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	} else {
		c.JSON(http.StatusCreated, dto.CreateOrganizationResponse{
			OrganizationID: id,
		})
	}
}

func (h *HTTPHandler) HandleGetOrganization(c *gin.Context) {
	organizationID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	org, err := h.orgs.GetOrganization(c.Request.Context(), organizationID, session.UserID)
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
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("Invalid organization id in update request", "id", idStr)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	var req dto.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid update organization request", "error", err, "organization", id)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request")))

		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(id, session.UserID)
	if err != nil {
		log.Warn(
			"Access denied for organization update — not a member",
			"organization",
			id,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		log.Warn(
			"Insufficient role for organization update",
			"organization",
			id,
			"user",
			session.UserID,
			"role",
			member.Role,
		)
		c.JSON(
			http.StatusForbidden,
			dto.Error(errors.New("only owner or administrator can update organization")),
		)

		return
	}

	org, err := data.GetOrganization(id)
	if err != nil {
		log.Warn("Organization not found for update", "organization", id, "user", session.UserID)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("organization not found")))

		return
	}

	org, err = data.UpdateOrganization(org, req.Name)
	if err != nil {
		log.Error(
			"Failed to update organization",
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

	c.JSON(http.StatusOK, dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	})
}

func HandleDeleteOrganization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("Invalid organization id in delete request", "id", idStr)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	session := auth.GetSession(c)

	org, err := data.GetOrganization(id)
	if err != nil {
		log.Warn("Organization not found for delete", "organization", id, "user", session.UserID)
		c.JSON(http.StatusNotFound, dto.Error(err))

		return
	}
	if org.CreatorID != session.UserID {
		log.Warn(
			"Non-owner attempted to delete organization",
			"organization",
			id,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner can delete organization")))

		return
	}

	if err := data.DeleteOrganization(id); err != nil {
		log.Error(
			"Failed to delete organization",
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

	c.JSON(http.StatusOK, gin.H{"success": true})
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
		log.Error(
			"Failed to check membership for members list",
			"error",
			err,
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}
	if !isMember {
		log.Warn("Access denied for members list", "organization", orgID, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}

	members, err := data.GetMembersWithUserInfo(orgID)
	if err != nil {
		log.Error(
			"Failed to get members",
			"error",
			err,
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

func HandleAddMember(c *gin.Context) {
	idStr := c.Param("id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("Invalid organization id in add member request", "id", idStr)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid add member request", "error", err, "organization", orgID)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))

		return
	}

	session := auth.GetSession(c)

	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		log.Warn(
			"Access denied for add member — not a member",
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		log.Warn(
			"Insufficient role for add member",
			"organization",
			orgID,
			"user",
			session.UserID,
			"role",
			currentMember.Role,
		)
		c.JSON(
			http.StatusForbidden,
			dto.Error(errors.New("only owner or administrator can add members")),
		)

		return
	}

	if data.Role(req.Role) == data.RoleOwner {
		log.Warn(
			"Attempt to assign owner role",
			"organization",
			orgID,
			"user",
			session.UserID,
			"target",
			req.UserID,
		)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("cannot assign owner role")))

		return
	}

	if data.Role(req.Role) != data.RoleAdministrator && data.Role(req.Role) != data.RoleMember {
		log.Warn(
			"Invalid role in add member request",
			"organization",
			orgID,
			"user",
			session.UserID,
			"role",
			req.Role,
		)
		c.JSON(
			http.StatusBadRequest,
			dto.Error(errors.New("invalid role, allowed: administrator, member")),
		)

		return
	}

	isMember, err := data.IsMember(req.UserID, orgID)
	if err != nil {
		log.Error(
			"Failed to check membership for add member",
			"error",
			err,
			"organization",
			orgID,
			"target",
			req.UserID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}
	if isMember {
		log.Warn("Target user is already a member", "organization", orgID, "target", req.UserID)
		c.JSON(http.StatusConflict, dto.Error(errors.New("user is already a member")))

		return
	}

	member := data.OrganizationMember{
		MemberID:       req.UserID,
		OrganizationID: orgID,
		Role:           data.Role(req.Role),
		JoinedAt:       time.Now(),
	}

	if err := data.AddMember(member); err != nil {
		log.Error("Failed to add member", "error", err, "organization", orgID, "target", req.UserID)
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
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))

		return
	}

	session := auth.GetSession(c)

	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		log.Warn(
			"Access denied for update role — not a member",
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		log.Warn(
			"Insufficient role for update role",
			"organization",
			orgID,
			"user",
			session.UserID,
			"role",
			currentMember.Role,
		)
		c.JSON(
			http.StatusForbidden,
			dto.Error(errors.New("only owner or administrator can change roles")),
		)

		return
	}

	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		log.Warn("Target member not found for role update", "organization", orgID, "target", userID)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("member not found")))

		return
	}
	if targetMember.Role == data.RoleOwner {
		log.Warn(
			"Attempt to change owner role",
			"organization",
			orgID,
			"user",
			session.UserID,
			"target",
			userID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("cannot change owner role")))

		return
	}

	if data.Role(req.Role) == data.RoleOwner {
		log.Warn(
			"Attempt to assign owner role via update",
			"organization",
			orgID,
			"user",
			session.UserID,
			"target",
			userID,
		)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("cannot assign owner role")))

		return
	}

	if data.Role(req.Role) != data.RoleAdministrator && data.Role(req.Role) != data.RoleMember {
		log.Warn(
			"Invalid role in update role request",
			"organization",
			orgID,
			"user",
			session.UserID,
			"role",
			req.Role,
		)
		c.JSON(
			http.StatusBadRequest,
			dto.Error(errors.New("invalid role, allowed: administrator, member")),
		)

		return
	}

	if err := data.UpdateMemberRole(orgID, userID, data.Role(req.Role)); err != nil {
		log.Error(
			"Failed to update member role",
			"error",
			err,
			"organization",
			orgID,
			"target",
			userID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandleRemoveMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("Invalid organization id in remove member request", "id", c.Param("id"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))

		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		log.Warn("Invalid user id in remove member request", "userId", c.Param("userId"))
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid user id")))

		return
	}

	session := auth.GetSession(c)

	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		log.Warn(
			"Access denied for remove member — not a member",
			"organization",
			orgID,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))

		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		log.Warn(
			"Insufficient role for remove member",
			"organization",
			orgID,
			"user",
			session.UserID,
			"role",
			currentMember.Role,
		)
		c.JSON(
			http.StatusForbidden,
			dto.Error(errors.New("only owner or administrator can remove members")),
		)

		return
	}

	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		log.Warn("Target member not found for removal", "organization", orgID, "target", userID)
		c.JSON(http.StatusNotFound, dto.Error(errors.New("member not found")))

		return
	}
	if targetMember.Role == data.RoleOwner {
		log.Warn(
			"Attempt to remove owner",
			"organization",
			orgID,
			"user",
			session.UserID,
			"target",
			userID,
		)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("cannot remove owner")))

		return
	}

	if err := data.RemoveMember(orgID, userID); err != nil {
		log.Error("Failed to remove member", "error", err, "organization", orgID, "target", userID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

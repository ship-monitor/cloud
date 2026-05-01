package organizations

import (
	"errors"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

func HandleCreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))
		return
	}

	session := auth.GetSession(c)

	org, err := data.CreateOrganization(data.CreateOrganizationInput{
		Name:      req.Name,
		CreatorID: session.UserID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	err = data.AddMember(data.OrganizationMember{
		MemberID:       session.UserID,
		OrganizationID: org.ID,
		Role:           data.RoleOwner,
		JoinedAt:       time.Now(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusCreated, dto.OrganizationResponse{
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
		log.Error("Invalid organization id", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, id)
	if err != nil {
		log.Error("Failed to check membership", "error", err, "organization", id, "user", session.UserID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}
	if !isMember {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}

	org, err := data.GetOrganization(id)
	if err != nil {
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
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

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
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can update organization")))
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
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Error("Bad UUID specified", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	org, err := data.GetOrganization(id)
	if err != nil {
		log.Error("Failed to get organization", "error", err, "id", idStr, "user", session.UserID)
		c.JSON(http.StatusNotFound, dto.Error(err))
		return
	}
	if org.CreatorID != session.UserID {
		log.Error("Only owner can delete organization", "id", idStr, "user", session.UserID)
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner can delete organization")))
		return
	}

	if err := data.DeleteOrganization(id); err != nil {
		log.Error("Failed to delete organization", "error", err, "id", idStr, "user", session.UserID)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandleGetMembers(c *gin.Context) {
	idStr := c.Param("id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil || !isMember {
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
	idStr := c.Param("id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))
		return
	}

	session := auth.GetSession(c)

	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can add members")))
		return
	}

	if data.Role(req.Role) == data.RoleOwner {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("cannot assign owner role")))
		return
	}

	if data.Role(req.Role) != data.RoleAdministrator && data.Role(req.Role) != data.RoleMember {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid role, allowed: administrator, member")))
		return
	}

	isMember, err := data.IsMember(req.UserID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}
	if isMember {
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
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid user id")))
		return
	}

	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: "+err.Error())))
		return
	}

	session := auth.GetSession(c)

	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can change roles")))
		return
	}

	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("member not found")))
		return
	}
	if targetMember.Role == data.RoleOwner {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("cannot change owner role")))
		return
	}

	if data.Role(req.Role) == data.RoleOwner {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("cannot assign owner role")))
		return
	}

	if data.Role(req.Role) != data.RoleAdministrator && data.Role(req.Role) != data.RoleMember {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid role, allowed: administrator, member")))
		return
	}

	if err := data.UpdateMemberRole(orgID, userID, data.Role(req.Role)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandleRemoveMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid user id")))
		return
	}

	session := auth.GetSession(c)

	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can remove members")))
		return
	}

	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("member not found")))
		return
	}
	if targetMember.Role == data.RoleOwner {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("cannot remove owner")))
		return
	}

	if err := data.RemoveMember(orgID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

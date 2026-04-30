package organizations

import (
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

func HandleCreateInvitation(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner or administrator can invite members"})
		return
	}

	var req dto.CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	exists, err := data.HasPendingInvitation(orgID, req.InviteeEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "pending invitation already exists for this email"})
		return
	}

	inv, err := data.CreateInvitation(data.OrgInvitationInput{
		OrganizationID: orgID,
		InviteeEmail:   req.InviteeEmail,
		ExpiresAt:      time.Now().Add(48 * time.Hour),
	})
	if err != nil {
		log.Error("Failed to create invitation", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invitationToDTO(inv))
}

func HandleListMyInvitations(c *gin.Context) {
	session := auth.GetSession(c)

	invs, err := data.ListInvitationsForUser(session.Email)
	if err != nil {
		log.Error("Failed to list invitations", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]dto.InvitationResponse, 0, len(invs))
	for i := range invs {
		resp = append(resp, invitationToDTO(&invs[i]))
	}

	c.JSON(http.StatusOK, dto.ListInvitationsResponse{Invitations: resp})
}

func HandleListOrgInvitations(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner or administrator can view invitations"})
		return
	}

	invs, err := data.ListInvitationsForOrg(orgID)
	if err != nil {
		log.Error("Failed to list org invitations", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]dto.InvitationResponse, 0, len(invs))
	for i := range invs {
		resp = append(resp, invitationToDTO(&invs[i]))
	}

	c.JSON(http.StatusOK, dto.ListInvitationsResponse{Invitations: resp})
}

func HandleAcceptInvitation(c *gin.Context) {
	invID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
		return
	}

	inv, err := data.GetInvitationByID(invID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		return
	}

	if inv.Status != data.StatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation already processed"})
		return
	}

	if time.Now().After(inv.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "invitation expired"})
		return
	}

	session := auth.GetSession(c)
	if session.Email != inv.InviteeEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the invitee"})
		return
	}

	alreadyMember, err := data.IsMember(session.UserID, inv.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if alreadyMember {
		c.JSON(http.StatusConflict, gin.H{"error": "you are already a member of this organization"})
		return
	}

	// Mark accepted first — if AddMember fails the invitation can be retried;
	// the reverse order would leave a dangling member with a still-pending invitation.
	if err := data.UpdateInvitationStatus(invID, data.StatusAccepted); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := data.AddMember(data.OrganizationMember{
		MemberID:       session.UserID,
		OrganizationID: inv.OrganizationID,
		Role:           data.RoleMember,
		JoinedAt:       time.Now(),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandleDeclineInvitation(c *gin.Context) {
	invID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
		return
	}

	inv, err := data.GetInvitationByID(invID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		return
	}

	if inv.Status != data.StatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation already processed"})
		return
	}

	session := auth.GetSession(c)
	if session.Email != inv.InviteeEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the invitee"})
		return
	}

	if err := data.UpdateInvitationStatus(invID, data.StatusDeclined); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func invitationToDTO(inv *data.OrganizationInvitation) dto.InvitationResponse {
	return dto.InvitationResponse{
		ID:             inv.ID,
		OrganizationID: inv.OrganizationID,
		InviteeEmail:   inv.InviteeEmail,
		Status:         string(inv.Status),
		CreatedAt:      inv.CreatedAt,
		ExpiresAt:      inv.ExpiresAt,
	}
}

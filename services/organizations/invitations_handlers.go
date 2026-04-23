package organizations

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

// --- Invitation handlers -------------------------------------------------

// HandleCreateInvitation invites a user to an organization.
func HandleCreateInvitation(c *gin.Context) {
	orgIDStr := c.Param("id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	// Only the creator (or an admin) can send invites – for now we check creator.
	session := auth.GetSession(c)
	org, err := data.GetOrganization(orgID)
	if err != nil || org.CreatorID != session.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only creator can invite members"})
		return
	}

	var req dto.CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Generate a short random token (e.g. 32‑char hex)
	token := uuid.New().String()

	inv, err := data.CreateInvitation(data.OrgInvitationInput{
		OrganizationID: orgID,
		InviteeEmail:   req.InviteeEmail,
		Token:          token,
		ExpiresAt:      time.Now().Add(48 * time.Hour), // 48h expiry
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := dto.InvitationResponse{
		ID:             inv.ID,
		OrganizationID: inv.OrganizationID,
		InviteeEmail:   inv.InviteeEmail,
		Token:          inv.Token,
		Status:         string(inv.Status),
		CreatedAt:      inv.CreatedAt,
		ExpiresAt:      inv.ExpiresAt,
	}

	c.JSON(http.StatusCreated, resp)
}

// HandleListInvitations returns all pending invitations for an organization.
func HandleListInvitations(c *gin.Context) {
	orgIDStr := c.Param("id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	session := auth.GetSession(c)
	// Only members can view invites – check membership
	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	invs, err := data.ListPendingInvitations(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []dto.InvitationResponse
	for _, i := range invs {
		resp = append(resp, dto.InvitationResponse{
			ID:             i.ID,
			OrganizationID: i.OrganizationID,
			InviteeEmail:   i.InviteeEmail,
			Token:          i.Token,
			Status:         string(i.Status),
			CreatedAt:      i.CreatedAt,
			ExpiresAt:      i.ExpiresAt,
		})
	}

	c.JSON(http.StatusOK, dto.ListInvitationsResponse{Invitations: resp})
}

// HandleAcceptInvitation accepts an invitation by its token.
func HandleAcceptInvitation(c *gin.Context) {
	token := c.Param("token")
	inv, err := data.GetInvitationByToken(token)
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

	// The user who accepts must match the invitee email.
	session := auth.GetSession(c)
	if session.Email != inv.InviteeEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not the invitee"})
		return
	}

	// Add user as member
	if err := data.AddMember(data.OrganizationMember{
		MemberID:       session.UserID,
		OrganizationID: inv.OrganizationID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Mark invitation as accepted
	if err := data.UpdateInvitationStatus(token, data.StatusAccepted); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandleDeclineInvitation declines an invitation by its token.
func HandleDeclineInvitation(c *gin.Context) {
	token := c.Param("token")
	inv, err := data.GetInvitationByToken(token)
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

	if err := data.UpdateInvitationStatus(token, data.StatusDeclined); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

package organizations

import (
	"encoding/json"
	"errors"
	"io"
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
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can invite members")))
		return
	}

	// Читаем сырой JSON
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("failed to read request body")))
		return
	}

	// Пробуем распарсить как одиночное приглашение
	var singleReq struct {
		InviteeEmail string `json:"inviteeEmail"`
	}
	if err := json.Unmarshal(body, &singleReq); err == nil && singleReq.InviteeEmail != "" {
		inv, err := createInvitation(orgID, singleReq.InviteeEmail)
		if err != nil {
			c.JSON(http.StatusConflict, dto.Error(err))
			return
		}
		c.JSON(http.StatusCreated, invitationToDTO(inv))
		return
	}

	// Пробуем распарсить как массовое приглашение
	var bulkReq struct {
		InviteeEmails []string `json:"inviteeEmails"`
	}
	if err := json.Unmarshal(body, &bulkReq); err == nil && len(bulkReq.InviteeEmails) > 0 {
		var created []dto.InvitationResponse
		var errs []string

		for _, email := range bulkReq.InviteeEmails {
			inv, err := createInvitation(orgID, email)
			if err != nil {
				errs = append(errs, email+": "+err.Error())
			} else {
				created = append(created, invitationToDTO(inv))
			}
		}

		c.JSON(http.StatusCreated, gin.H{"invitations": created, "errors": errs})
		return
	}

	c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid request: expected inviteeEmail or inviteeEmails")))
}

func createInvitation(orgID uuid.UUID, email string) (*data.OrganizationInvitation, error) {
	exists, err := data.HasPendingInvitation(orgID, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("pending invitation already exists for " + email)
	}

	return data.CreateInvitation(data.OrgInvitationInput{
		OrganizationID: orgID,
		InviteeEmail:   email,
		ExpiresAt:      time.Now().Add(48 * time.Hour),
	})
}

func HandleListMyInvitations(c *gin.Context) {
	session := auth.GetSession(c)

	invs, err := data.ListInvitationsForUser(session.Email)
	if err != nil {
		log.Error("Failed to list invitations", "error", err)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
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
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid organization id")))
		return
	}

	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("only owner or administrator can view invitations")))
		return
	}

	invs, err := data.ListInvitationsForOrg(orgID)
	if err != nil {
		log.Error("Failed to list org invitations", "error", err)
		c.JSON(http.StatusInternalServerError, dto.Error(err))
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
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid invitation id")))
		return
	}

	inv, err := data.GetInvitationByID(invID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("invitation not found")))
		return
	}

	if inv.Status != data.StatusPending {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invitation already processed")))
		return
	}

	if time.Now().After(inv.ExpiresAt) {
		c.JSON(http.StatusGone, dto.Error(errors.New("invitation expired")))
		return
	}

	session := auth.GetSession(c)
	if session.Email != inv.InviteeEmail {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("you are not the invitee")))
		return
	}

	alreadyMember, err := data.IsMember(session.UserID, inv.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}
	if alreadyMember {
		c.JSON(http.StatusConflict, dto.Error(errors.New("you are already a member of this organization")))
		return
	}

	if err := data.UpdateInvitationStatus(invID, data.StatusAccepted); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	if err := data.AddMember(data.OrganizationMember{
		MemberID:       session.UserID,
		OrganizationID: inv.OrganizationID,
		Role:           data.RoleMember,
		JoinedAt:       time.Now(),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandleDeclineInvitation(c *gin.Context) {
	invID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invalid invitation id")))
		return
	}

	inv, err := data.GetInvitationByID(invID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(errors.New("invitation not found")))
		return
	}

	if inv.Status != data.StatusPending {
		c.JSON(http.StatusBadRequest, dto.Error(errors.New("invitation already processed")))
		return
	}

	session := auth.GetSession(c)
	if session.Email != inv.InviteeEmail {
		c.JSON(http.StatusForbidden, dto.Error(errors.New("you are not the invitee")))
		return
	}

	if err := data.UpdateInvitationStatus(invID, data.StatusDeclined); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func invitationToDTO(inv *data.OrganizationInvitation) dto.InvitationResponse {
	org, err := data.GetOrganization(inv.OrganizationID)
	orgName := ""
	if err == nil {
		orgName = org.Name
	}

	return dto.InvitationResponse{
		ID:               inv.ID,
		OrganizationID:   inv.OrganizationID,
		OrganizationName: orgName,
		InviteeEmail:     inv.InviteeEmail,
		Status:           string(inv.Status),
		CreatedAt:        inv.CreatedAt,
		ExpiresAt:        inv.ExpiresAt,
	}
}

package handlers

import (
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

var (
	ErrAlreadyProcessed         = errors.New("invitation already processed")
	ErrInvitationNotFound       = errors.New("invitation not found")
	ErrAccessDenied             = errors.New("access denied")
	ErrInvitationAlreadyPending = errors.New("invitation already exists and is pending")
	ErrInvitationExpired        = errors.New("invitation expired")
	ErrAlreadyMember            = errors.New("user is already a member")
	ErrNotInvitee               = errors.New("user is not invitee")
	ErrInvalidInviteRequest     = errors.New(
		"invalid request: expected inviteeEmail or inviteeEmails",
	)
)

func HandleCreateInvitation(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	_, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(ErrAccessDenied))

		return
	}

	var singleReq struct {
		InviteeEmail string `json:"inviteeEmail"`
	}
	if err := c.ShouldBindJSON(&singleReq); err == nil && singleReq.InviteeEmail != "" {
		inv, err := createInvitation(orgID, singleReq.InviteeEmail)

		switch {
		case errors.Is(err, ErrInvitationAlreadyPending):
			c.JSON(http.StatusConflict, dto.Error(err))
		case err != nil:
			c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Error(err))
		default:
			c.JSON(http.StatusCreated, invitationToDTO(inv))
		}

		return
	}

	var bulkReq struct {
		InviteeEmails []string `json:"inviteeEmails"`
	}
	if err := c.ShouldBindJSON(&bulkReq); err != nil || len(bulkReq.InviteeEmails) == 0 {
		c.JSON(
			http.StatusBadRequest,
			dto.Error(ErrInvalidInviteRequest),
		)

		return
	}

	var (
		created []dto.InvitationResponse
		errs    error
	)

	for _, email := range bulkReq.InviteeEmails {
		if inv, err := createInvitation(orgID, email); err != nil {
			errs = errors.Join(errs, fmt.Errorf("invite %q: %w", email, err))
		} else {
			created = append(created, invitationToDTO(inv))
		}
	}

	c.JSON(http.StatusCreated, gin.H{"invitations": created, "errors": errs.Error()})
}

func createInvitation(orgID uuid.UUID, email string) (*domain.OrganizationInvitation, error) {
	exists, err := data.HasPendingInvitation(orgID, email)
	if err != nil {
		return nil, fmt.Errorf("check pending invitations: %w", err)
	}

	if exists {
		return nil, ErrInvitationAlreadyPending
	}

	inv, err := data.CreateInvitation(data.OrgInvitationInput{
		OrganizationID: orgID,
		InviteeEmail:   email,
		ExpiresAt:      time.Now().Add(InvitationsTTL),
	})
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	return inv, nil
}

const InvitationsTTL = 48 * time.Hour

func HandleListMyInvitations(c *gin.Context) {
	session := auth.GetSession(c)

	invs, err := data.ListInvitationsForUser(session.Email)
	if err != nil {
		log.Error("Failed to list invitations for user", "error", err, "user", session.UserID)
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
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	member, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Error(ErrAccessDenied))

		return
	}

	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdministrator {
		c.JSON(
			http.StatusForbidden,
			dto.Error(ErrAccessDenied),
		)

		return
	}

	invs, err := data.ListInvitationsForOrg(orgID)
	if err != nil {
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
	invID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	inv, err := data.GetInvitationByID(c.Request.Context(), invID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error(ErrInvitationNotFound))

		return
	}

	if inv.Status != domain.InvStatusPending {
		c.JSON(http.StatusBadRequest, dto.Error(ErrAlreadyProcessed))

		return
	}

	if time.Now().After(inv.ExpiresAt) {
		c.JSON(http.StatusGone, dto.Error(ErrInvitationExpired))

		return
	}

	if session.Email != inv.InviteeEmail {
		c.JSON(http.StatusForbidden, dto.Error(ErrNotInvitee))

		return
	}

	alreadyMember, err := data.IsMember(session.UserID, inv.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	if alreadyMember {
		c.JSON(http.StatusConflict, dto.Error(ErrAlreadyMember))

		return
	}

	if err := data.UpdateInvitationStatus(invID, domain.InvStatusAccepted); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	if err := data.AddMember(domain.OrganizationMember{
		MemberID:       session.UserID,
		OrganizationID: inv.OrganizationID,
		Role:           domain.RoleMember,
		JoinedAt:       time.Now(),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.Status(http.StatusOK)
}

func HandleDeclineInvitation(c *gin.Context) {
	invID := requests.MustGetParamUUID(c, "id")

	inv, err := data.GetInvitationByID(c.Request.Context(), invID)
	if err != nil {
		log.Warn("Invitation not found for decline", "invitation", invID)
		c.JSON(http.StatusNotFound, dto.Error(ErrInvitationNotFound))

		return
	}

	if inv.Status != domain.InvStatusPending {
		log.Warn(
			"Attempt to decline already processed invitation",
			"invitation",
			invID,
			"status",
			inv.Status,
		)
		c.JSON(http.StatusBadRequest, dto.Error(ErrAlreadyProcessed))

		return
	}

	session := auth.GetSession(c)
	if session.Email != inv.InviteeEmail {
		log.Warn(
			"User is not the invitee on decline",
			"invitation",
			invID,
			"user",
			session.UserID,
			"invitee",
			inv.InviteeEmail,
		)
		c.JSON(http.StatusForbidden, dto.Error(ErrNotInvitee))

		return
	}

	if err := data.UpdateInvitationStatus(invID, domain.InvStatusDeclined); err != nil {
		log.Error(
			"Failed to update invitation status to declined",
			"error",
			err,
			"invitation",
			invID,
		)
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	c.Status(http.StatusOK)
}

func invitationToDTO(inv *domain.OrganizationInvitation) dto.InvitationResponse {
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

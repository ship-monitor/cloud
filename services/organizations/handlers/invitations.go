package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

var ErrInvalidInviteRequest = errors.New(
	"invalid request: expected inviteeEmail or inviteeEmails",
)

func (h *HTTPHandler) HandleCreateInvitation(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req struct {
		InviteeEmail  string   `json:"inviteeEmail"`
		InviteeEmails []string `json:"inviteeEmails"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(ErrInvalidInviteRequest))

		return
	}

	if req.InviteeEmail != "" {
		inv, err := h.orgs.CreateInvitation(
			c.Request.Context(),
			orgID,
			session.UserID,
			req.InviteeEmail,
		)
		writeCreateInvitationResponse(c, inv, err)

		return
	}

	if len(req.InviteeEmails) == 0 {
		c.JSON(http.StatusBadRequest, dto.Error(ErrInvalidInviteRequest))

		return
	}

	created := make([]dto.InvitationResponse, 0, len(req.InviteeEmails))

	var errs error

	for _, email := range req.InviteeEmails {
		inv, err := h.orgs.CreateInvitation(c.Request.Context(), orgID, session.UserID, email)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("invite %q: %w", email, err))
		} else {
			created = append(created, invitationToDTO(*inv))
		}
	}

	resp := gin.H{"invitations": created}
	if errs != nil {
		resp["errors"] = errs.Error()
	}

	c.JSON(http.StatusCreated, resp)
}

func writeCreateInvitationResponse(
	c *gin.Context,
	inv *services.InvitationDetails,
	err error,
) {
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
	case errors.Is(err, services.ErrInvitationAlreadyPending):
		c.JSON(http.StatusConflict, dto.Error(err))
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.JSON(http.StatusCreated, invitationToDTO(*inv))
	}
}

func (h *HTTPHandler) HandleListMyInvitations(c *gin.Context) {
	session := auth.GetSession(c)

	invs, err := h.orgs.ListMyInvitations(c.Request.Context(), session.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error(err))

		return
	}

	resp := make([]dto.InvitationResponse, 0, len(invs))
	for i := range invs {
		resp = append(resp, invitationToDTO(invs[i]))
	}

	c.JSON(http.StatusOK, dto.ListInvitationsResponse{Invitations: resp})
}

func (h *HTTPHandler) HandleListOrgInvitations(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	invs, err := h.orgs.ListOrgInvitations(c.Request.Context(), orgID, session.UserID)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember), errors.Is(err, services.ErrNotAllowed):
		c.JSON(http.StatusForbidden, dto.Error(errors.New("access denied")))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		resp := make([]dto.InvitationResponse, 0, len(invs))
		for i := range invs {
			resp = append(resp, invitationToDTO(invs[i]))
		}

		c.JSON(http.StatusOK, dto.ListInvitationsResponse{Invitations: resp})
	}
}

func (h *HTTPHandler) HandleAcceptInvitation(c *gin.Context) {
	invID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	err := h.orgs.AcceptInvitation(c.Request.Context(), invID, session.UserID, session.Email)
	switch {
	case errors.Is(err, services.ErrInvitationNotFound):
		c.JSON(http.StatusNotFound, dto.Error(err))
	case errors.Is(err, services.ErrInvitationAlreadyProcessed):
		c.JSON(http.StatusBadRequest, dto.Error(err))
	case errors.Is(err, services.ErrInvitationExpired):
		c.JSON(http.StatusGone, dto.Error(err))
	case errors.Is(err, services.ErrNotInvitee):
		c.JSON(http.StatusForbidden, dto.Error(err))
	case errors.Is(err, services.ErrMemberAlreadyExists):
		c.JSON(http.StatusConflict, dto.Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.Status(http.StatusOK)
	}
}

func (h *HTTPHandler) HandleDeclineInvitation(c *gin.Context) {
	invID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	err := h.orgs.DeclineInvitation(c.Request.Context(), invID, session.Email)
	switch {
	case errors.Is(err, services.ErrInvitationNotFound):
		c.JSON(http.StatusNotFound, dto.Error(err))
	case errors.Is(err, services.ErrInvitationAlreadyProcessed):
		c.JSON(http.StatusBadRequest, dto.Error(err))
	case errors.Is(err, services.ErrNotInvitee):
		c.JSON(http.StatusForbidden, dto.Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, dto.Error(err))
	default:
		c.Status(http.StatusOK)
	}
}

func invitationToDTO(inv services.InvitationDetails) dto.InvitationResponse {
	return dto.InvitationResponse{
		ID:               inv.Invitation.ID,
		OrganizationID:   inv.Invitation.OrganizationID,
		OrganizationName: inv.OrganizationName,
		InviteeEmail:     inv.Invitation.InviteeEmail,
		Status:           string(inv.Invitation.Status),
		CreatedAt:        inv.Invitation.CreatedAt,
		ExpiresAt:        inv.Invitation.ExpiresAt,
	}
}

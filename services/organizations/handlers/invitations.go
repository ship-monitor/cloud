package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

var ErrInvalidInviteRequest = errors.New(
	"invalid request: expected inviteeEmail or inviteeEmails",
)

func (h *OrgsHandlers) HandleCreateInvitation(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req struct {
		InviteeEmail  string   `json:"inviteeEmail"`
		InviteeEmails []string `json:"inviteeEmails"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Error(ErrInvalidInviteRequest))

		return
	}

	if req.InviteeEmail != "" {
		inv, err := h.invitations.CreateInvitation(
			c.Request.Context(),
			orgID,
			session.UserID,
			req.InviteeEmail,
		)
		writeCreateInvitationResponse(c, inv, err)

		return
	}

	if len(req.InviteeEmails) == 0 {
		c.JSON(http.StatusBadRequest, Error(ErrInvalidInviteRequest))

		return
	}

	created := make([]InvitationResponse, 0, len(req.InviteeEmails))

	var errs error

	for _, email := range req.InviteeEmails {
		inv, err := h.invitations.CreateInvitation(
			c.Request.Context(),
			orgID,
			session.UserID,
			email,
		)
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
		c.JSON(http.StatusForbidden, Error(AccesDeniedError(err)))
	case errors.Is(err, services.ErrInvitationAlreadyPending):
		c.JSON(http.StatusConflict, Error(err))
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, Error(err))
	default:
		c.JSON(http.StatusCreated, invitationToDTO(*inv))
	}
}

func (h *OrgsHandlers) HandleListMyInvitations(c *gin.Context) {
	session := auth.GetSession(c)

	invs, err := h.invitations.ListMyInvitations(
		c.Request.Context(),
		session.Email,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Error(err))

		return
	}

	resp := make([]InvitationResponse, 0, len(invs))
	for i := range invs {
		resp = append(resp, invitationToDTO(invs[i]))
	}

	c.JSON(http.StatusOK, ListInvitationsResponse{Invitations: resp})
}

func (h *OrgsHandlers) HandleListOrgInvitations(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	invs, err := h.invitations.ListOrgInvitations(
		c.Request.Context(),
		orgID,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember),
		errors.Is(err, services.ErrNotAllowed):
		c.JSON(http.StatusForbidden, Error(AccesDeniedError(err)))
	case err != nil:
		c.JSON(http.StatusInternalServerError, Error(err))
	default:
		resp := make([]InvitationResponse, 0, len(invs))
		for i := range invs {
			resp = append(resp, invitationToDTO(invs[i]))
		}

		c.JSON(http.StatusOK, ListInvitationsResponse{Invitations: resp})
	}
}

func (h *OrgsHandlers) HandleAcceptInvitation(c *gin.Context) {
	invID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	err := h.invitations.AcceptInvitation(
		c.Request.Context(),
		invID,
		session.UserID,
		session.Email,
	)
	switch {
	case errors.Is(err, services.ErrInvitationNotFound):
		c.JSON(http.StatusNotFound, Error(err))
	case errors.Is(err, services.ErrInvitationAlreadyProcessed):
		c.JSON(http.StatusBadRequest, Error(err))
	case errors.Is(err, services.ErrInvitationExpired):
		c.JSON(http.StatusGone, Error(err))
	case errors.Is(err, services.ErrNotInvitee):
		c.JSON(http.StatusForbidden, Error(err))
	case errors.Is(err, services.ErrMemberAlreadyExists):
		c.JSON(http.StatusConflict, Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, Error(err))
	default:
		c.Status(http.StatusOK)
	}
}

func (h *OrgsHandlers) HandleDeclineInvitation(c *gin.Context) {
	invID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	err := h.invitations.DeclineInvitation(
		c.Request.Context(),
		invID,
		session.Email,
	)
	switch {
	case errors.Is(err, services.ErrInvitationNotFound):
		c.JSON(http.StatusNotFound, Error(err))
	case errors.Is(err, services.ErrInvitationAlreadyProcessed):
		c.JSON(http.StatusBadRequest, Error(err))
	case errors.Is(err, services.ErrNotInvitee):
		c.JSON(http.StatusForbidden, Error(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, Error(err))
	default:
		c.Status(http.StatusOK)
	}
}

func invitationToDTO(inv services.InvitationDetails) InvitationResponse {
	return InvitationResponse{
		ID:               inv.Invitation.ID,
		OrganizationID:   inv.Invitation.OrganizationID,
		OrganizationName: inv.OrganizationName,
		InviteeEmail:     inv.Invitation.InviteeEmail,
		Status:           string(inv.Invitation.Status),
		CreatedAt:        inv.Invitation.CreatedAt,
		ExpiresAt:        inv.Invitation.ExpiresAt,
	}
}

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type CreateInvitationRequest struct {
	InviteeEmail string `binding:"required" json:"inviteeEmail"`
}

type CreateInvitationBulkRequest struct {
	InviteeEmails []string `binding:"required" json:"inviteeEmails"`
}

type InvitationResponse struct {
	ID               uuid.UUID `json:"id"`
	OrganizationID   uuid.UUID `json:"organizationId"`
	OrganizationName string    `json:"organizationName,omitempty"`
	InviteeEmail     string    `json:"inviteeEmail"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type ListInvitationsResponse struct {
	Invitations []InvitationResponse `json:"invitations"`
}

type InvitationsService interface {
	CreateInvitation(
		ctx context.Context,
		organizationID uuid.UUID,
		inviterID uuid.UUID,
		inviteeEmail string,
	) (*services.InvitationDetails, error)
	ListMyInvitations(
		ctx context.Context,
		email string,
	) ([]services.InvitationDetails, error)
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
	DeclineInvitation(
		ctx context.Context,
		invitationID uuid.UUID,
		userEmail string,
	) error
}

var ErrInvalidInviteRequest = errors.New(
	"invalid request: expected inviteeEmail or inviteeEmails",
)

var _ pkg.Handler = (*InvitationHandler)(nil)

type InvitationHandler struct {
	invitations InvitationsService
	middleware  *auth.Middleware
}

func NewInvitation(
	invitations InvitationsService,
	middleware *auth.Middleware,
) *InvitationHandler {
	return &InvitationHandler{invitations: invitations, middleware: middleware}
}

// SetupRoutes implements [pkg.Handler].
func (h *InvitationHandler) SetupRoutes(router gin.IRouter) {
	router.POST(
		"/api/organizations/:id/invitations",
		h.middleware.WithAuthenticationRequired,
		h.HandleCreateInvitation,
	)
	router.GET(
		"/api/organizations/:id/invitations",
		h.middleware.WithAuthenticationRequired,
		h.HandleListOrgInvitations,
	)
	router.GET(
		"/api/invitations",
		h.middleware.WithAuthenticationRequired,
		h.HandleListMyInvitations,
	)
	router.POST(
		"/api/invitations/:id/accept",
		h.middleware.WithAuthenticationRequired, h.HandleAcceptInvitation,
	)
	router.POST(
		"/api/invitations/:id/decline",
		h.middleware.WithAuthenticationRequired, h.HandleDeclineInvitation,
	)
}

func (h *InvitationHandler) HandleCreateInvitation(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	var req struct {
		InviteeEmail  string   `json:"inviteeEmail"`
		InviteeEmails []string `json:"inviteeEmails"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(ErrInvalidInviteRequest),
		)

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
		c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(ErrInvalidInviteRequest),
		)

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
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case errors.Is(err, services.ErrInvitationAlreadyPending):
		c.JSON(http.StatusConflict, requests.ResponseErr(err))
	case err != nil:
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			requests.ResponseErr(err),
		)
	default:
		c.JSON(http.StatusCreated, invitationToDTO(*inv))
	}
}

func (h *InvitationHandler) HandleListMyInvitations(c *gin.Context) {
	session := auth.GetSession(c)

	invs, err := h.invitations.ListMyInvitations(
		c.Request.Context(),
		session.Email,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))

		return
	}

	resp := make([]InvitationResponse, 0, len(invs))
	for i := range invs {
		resp = append(resp, invitationToDTO(invs[i]))
	}

	c.JSON(http.StatusOK, ListInvitationsResponse{Invitations: resp})
}

func (h *InvitationHandler) HandleListOrgInvitations(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		resp := make([]InvitationResponse, 0, len(invs))
		for i := range invs {
			resp = append(resp, invitationToDTO(invs[i]))
		}

		c.JSON(http.StatusOK, ListInvitationsResponse{Invitations: resp})
	}
}

func (h *InvitationHandler) HandleAcceptInvitation(c *gin.Context) {
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
		c.JSON(http.StatusNotFound, requests.ResponseErr(err))
	case errors.Is(err, services.ErrInvitationAlreadyProcessed):
		c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	case errors.Is(err, services.ErrInvitationExpired):
		c.JSON(http.StatusGone, requests.ResponseErr(err))
	case errors.Is(err, services.ErrNotInvitee):
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case errors.Is(err, services.ErrMemberAlreadyExists):
		c.JSON(http.StatusConflict, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
	default:
		c.Status(http.StatusOK)
	}
}

func (h *InvitationHandler) HandleDeclineInvitation(c *gin.Context) {
	invID := requests.MustGetParamUUID(c, "id")
	session := auth.GetSession(c)

	err := h.invitations.DeclineInvitation(
		c.Request.Context(),
		invID,
		session.Email,
	)
	switch {
	case errors.Is(err, services.ErrInvitationNotFound):
		c.JSON(http.StatusNotFound, requests.ResponseErr(err))
	case errors.Is(err, services.ErrInvitationAlreadyProcessed):
		c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	case errors.Is(err, services.ErrNotInvitee):
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))
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

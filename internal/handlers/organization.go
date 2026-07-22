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
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/middleware"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/requests"
)

type CreateOrganizationResponse struct {
	OrganizationID uuid.UUID `json:"organizationId"`
}

type OrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateOrganizationRequest struct {
	Name string `binding:"required" json:"name"`
}

type UpdateOrganizationRequest struct {
	Name string `binding:"required" json:"name"`
}

type AddMemberRequest struct {
	UserID uuid.UUID `binding:"required" json:"userId"`
	Role   string    `binding:"required" json:"role"`
}

type UpdateMemberRoleRequest struct {
	Role string `binding:"required" json:"role"`
}

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
	RenameOrganization(
		ctx context.Context,
		organizationID uuid.UUID,
		name string,
		userID uuid.UUID,
	) error
	DeleteOrganization(
		ctx context.Context,
		organizationID, userID uuid.UUID,
	) error
}

type OrganizationMembersService interface {
	GetMembers(
		ctx context.Context,
		organizationID uuid.UUID,
		userID uuid.UUID,
	) ([]domain.OrganizationMemberWithUser, error)
	AddMember(
		ctx context.Context,
		organizationID uuid.UUID,
		actorID uuid.UUID,
		userID uuid.UUID,
		role domain.Role,
	) (*domain.OrganizationMember, error)
	UpdateMemberRole(
		ctx context.Context,
		organizationID uuid.UUID,
		actorID uuid.UUID,
		userID uuid.UUID,
		role domain.Role,
	) error
	RemoveMember(
		ctx context.Context,
		organizationID uuid.UUID,
		actorID uuid.UUID,
		userID uuid.UUID,
	) error
}

var _ pkg.Handler = (*OrgsHandler)(nil)

type OrgsHandler struct {
	orgs       OrganizationService
	members    OrganizationMembersService
	middleware *middleware.AuthMiddleware
}

func NewOrgs(
	orgs OrganizationService,
	members OrganizationMembersService,
	middleware *middleware.AuthMiddleware,
) *OrgsHandler {
	return &OrgsHandler{
		orgs:       orgs,
		members:    members,
		middleware: middleware,
	}
}

// SetupRoutes implements [pkg.Handler].
func (h *OrgsHandler) SetupRoutes(router gin.IRouter) {
	orgs := router.Group(
		"/api/organizations",
		h.middleware.RequireAuth(),
	)
	orgs.POST("/", h.HandleCreateOrganization)
	orgs.GET("/my", h.HandleGetMyOrganizations)
	orgs.GET("/:id", h.HandleGetOrganization)
	orgs.PATCH("/:id", h.HandleUpdateOrganization)
	orgs.DELETE("/:id", h.HandleDeleteOrganization)

	// Роуты участников организации
	orgs.GET("/:id/members", h.HandleGetMembers)
	orgs.POST("/:id/members", h.HandleAddMember)
	orgs.PATCH("/:id/members/:userId", h.HandleUpdateMemberRole)
	orgs.DELETE("/:id/members/:userId", h.HandleRemoveMember)
}

func (h *OrgsHandler) HandleCreateOrganization(c *gin.Context) {
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid create organization request", "error", err)
		c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(fmt.Errorf("invalid request: %w", err)),
		)

		return
	}

	session := middleware.MustPrincipal(c)

	id, err := h.orgs.CreateOrganization(
		c.Request.Context(),
		req.Name,
		session.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))

		return
	}

	c.JSON(http.StatusCreated, CreateOrganizationResponse{
		OrganizationID: id,
	})
}

func (h *OrgsHandler) HandleGetMyOrganizations(c *gin.Context) {
	session := middleware.MustPrincipal(c)

	orgs, err := h.orgs.GetUsersOrganizations(
		c.Request.Context(),
		session.UserID,
		0,
	)
	if err != nil {
		log.Error(
			"Failed to get organizations for member",
			"error",
			err,
			"user",
			session.UserID,
		)
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))

		return
	}

	resp := make([]OrganizationResponse, 0, len(orgs))
	for _, org := range orgs {
		resp = append(resp, organizationToDTO(org))
	}

	c.JSON(http.StatusOK, gin.H{"organizations": resp})
}

func (h *OrgsHandler) HandleGetOrganization(c *gin.Context) {
	organizationID := requests.MustGetParamUUID(c, "id")
	session := middleware.MustPrincipal(c)

	org, err := h.orgs.GetOrganization(
		c.Request.Context(),
		organizationID,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))

		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, requests.ResponseErr(err))

		return
	}

	c.JSON(http.StatusOK, organizationToDTO(org))
}

func InvalidRequestError(err error) error {
	return fmt.Errorf("invalid request: %w", err)
}

func AccesDeniedError(err error) error {
	return fmt.Errorf("access denied: %w", err)
}

func InternalServerError(err error) error {
	return fmt.Errorf("internal server error: %w", err)
}

func (h *OrgsHandler) HandleUpdateOrganization(c *gin.Context) {
	id := requests.MustGetParamUUID(c, "id")

	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(InvalidRequestError(err)),
		)

		return
	}

	session := middleware.MustPrincipal(c)

	err := h.orgs.RenameOrganization(
		c.Request.Context(),
		id,
		req.Name,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(
			http.StatusForbidden,
			requests.ResponseErr(AccesDeniedError(err)),
		)
	case err != nil:
		c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(InternalServerError(err)),
		)
	default:
		c.Status(http.StatusOK)
	}
}

func (h *OrgsHandler) HandleDeleteOrganization(c *gin.Context) {
	id := requests.MustGetParamUUID(c, "id")
	session := middleware.MustPrincipal(c)

	err := h.orgs.DeleteOrganization(c.Request.Context(), id, session.UserID)
	switch {
	case errors.Is(err, services.ErrOrganizationNotFound):
		c.JSON(http.StatusNotFound, requests.ResponseErr(err))
	case errors.Is(err, services.ErrOnlyOwnerCanDelete):
		c.JSON(
			http.StatusForbidden,
			requests.ResponseErr(AccesDeniedError(err)),
		)
	case err != nil:
		c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(InternalServerError(err)),
		)
	default:
		c.Status(http.StatusOK)
	}
}

func (h *OrgsHandler) HandleGetMembers(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := middleware.MustPrincipal(c)

	members, err := h.members.GetMembers(
		c.Request.Context(),
		orgID,
		session.UserID,
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(
			http.StatusForbidden,
			requests.ResponseErr(AccesDeniedError(err)),
		)
	case err != nil:
		c.JSON(
			http.StatusInternalServerError,
			requests.ResponseErr(InternalServerError(err)),
		)
	default:
		c.JSON(http.StatusOK, gin.H{"members": members})
	}
}

func (h *OrgsHandler) HandleAddMember(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	session := middleware.MustPrincipal(c)

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(InvalidRequestError(err)),
		)

		return
	}

	member, err := h.members.AddMember(
		c.Request.Context(),
		orgID,
		session.UserID,
		req.UserID,
		domain.Role(req.Role),
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(
			http.StatusForbidden,
			requests.ResponseErr(AccesDeniedError(err)),
		)
	case errors.Is(err, services.ErrCannotAssignOwnerRole),
		errors.Is(err, services.ErrInvalidMemberRole):
		c.JSON(http.StatusBadRequest, AccesDeniedError(err))
	case errors.Is(err, services.ErrMemberAlreadyExists):
		c.JSON(http.StatusConflict, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, InternalServerError(err))
	default:
		c.JSON(http.StatusCreated, gin.H{
			"memberId":       member.MemberID,
			"organizationId": member.OrganizationID,
			"role":           member.Role,
			"joinedAt":       member.JoinedAt,
		})
	}
}

func (h *OrgsHandler) HandleUpdateMemberRole(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	userID := requests.MustGetParamUUID(c, "userId")

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			requests.ResponseErr(InvalidRequestError(err)),
		)

		return
	}

	session := middleware.MustPrincipal(c)

	err := h.members.UpdateMemberRole(
		c.Request.Context(),
		orgID,
		session.UserID,
		userID,
		domain.Role(req.Role),
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(
			http.StatusForbidden,
			requests.ResponseErr(AccesDeniedError(err)),
		)
	case errors.Is(err, services.ErrMemberNotFound):
		c.JSON(http.StatusNotFound, requests.ResponseErr(err))
	case errors.Is(err, services.ErrCannotChangeOwnerRole):
		c.JSON(http.StatusForbidden, requests.ResponseErr(err))
	case errors.Is(err, services.ErrCannotAssignOwnerRole) ||
		errors.Is(err, services.ErrInvalidMemberRole):
		c.JSON(http.StatusBadRequest, requests.ResponseErr(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, InternalServerError(err))
	default:
		c.Status(http.StatusOK)
	}
}

func (h *OrgsHandler) HandleRemoveMember(c *gin.Context) {
	orgID := requests.MustGetParamUUID(c, "id")
	userID := requests.MustGetParamUUID(c, "userId")
	session := middleware.MustPrincipal(c)

	err := h.members.RemoveMember(
		c.Request.Context(),
		orgID,
		session.UserID,
		userID,
	)
	switch {
	case errors.Is(err, services.ErrUserIsNotMember):
		c.JSON(
			http.StatusForbidden,
			requests.ResponseErr(AccesDeniedError(err)),
		)
	case errors.Is(err, services.ErrMemberNotFound):
		c.JSON(http.StatusNotFound, requests.ResponseErr(err))
	case errors.Is(err, services.ErrNotAllowed),
		errors.Is(err, services.ErrRemovingOrganizationOwner):
		c.JSON(http.StatusForbidden, AccesDeniedError(err))
	case err != nil:
		c.JSON(http.StatusInternalServerError, InternalServerError(err))
	default:
		c.Status(http.StatusOK)
	}
}

func organizationToDTO(org *domain.Organization) OrganizationResponse {
	return OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
}

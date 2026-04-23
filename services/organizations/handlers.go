package organizations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/auth"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

// HandleCreateOrganization создаёт новую организацию
func HandleCreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	session := auth.GetSession(c)

	// Создаём организацию
	org, err := data.CreateOrganization(data.CreateOrganizationInput{
		Name:      req.Name,
		CreatorID: session.UserID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Добавляем создателя как участника
	err = data.AddMember(data.OrganizationMember{
		MemberID:       session.UserID,
		OrganizationID: org.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}

	c.JSON(http.StatusCreated, resp)
}

// HandleGetMyOrganizations возвращает организации текущего пользователя
func HandleGetMyOrganizations(c *gin.Context) {
	session := auth.GetSession(c)

	orgs, err := data.GetOrganizationsByMember(session.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []dto.OrganizationResponse
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

// HandleGetOrganization получает организацию по ID
func HandleGetOrganization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	session := auth.GetSession(c)

	// Проверяем, что пользователь имеет доступ к организации
	isMember, err := data.IsMember(session.UserID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	org, err := data.GetOrganization(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	resp := dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}

	c.JSON(http.StatusOK, resp)
}

// HandleUpdateOrganization обновляет организацию
func HandleUpdateOrganization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var req dto.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	session := auth.GetSession(c)

	// Проверяем права (пока только creator может редактировать, потом добавим роли)
	org, err := data.GetOrganization(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	if org.CreatorID != session.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only creator can update organization"})
		return
	}

	org, err = data.UpdateOrganization(org, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := dto.OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}

	c.JSON(http.StatusOK, resp)
}

// HandleDeleteOrganization удаляет организацию
func HandleDeleteOrganization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	session := auth.GetSession(c)

	// Проверяем права (пока только creator может удалять)
	org, err := data.GetOrganization(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	if org.CreatorID != session.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only creator can delete organization"})
		return
	}

	err = data.DeleteOrganization(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

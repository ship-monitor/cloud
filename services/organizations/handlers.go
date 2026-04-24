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

// HandleCreateOrganization создаёт новую организацию
func HandleCreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	session := auth.GetSession(c)

	org, err := data.CreateOrganization(data.CreateOrganizationInput{
		Name:      req.Name,
		CreatorID: session.UserID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = data.AddMember(data.OrganizationMember{
		MemberID:       session.UserID,
		OrganizationID: org.ID,
		Role:           data.RoleOwner,
		JoinedAt:       time.Now(),
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

	// Проверяем права: owner или administrator
	member, err := data.GetMember(id, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if member.Role != data.RoleOwner && member.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner or administrator can update organization"})
		return
	}

	org, err := data.GetOrganization(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
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

	// Только owner может удалить
	member, err := data.GetMember(id, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if member.Role != data.RoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner can delete organization"})
		return
	}

	err = data.DeleteOrganization(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandleGetMembers возвращает список участников организации
func HandleGetMembers(c *gin.Context) {
	idStr := c.Param("id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	session := auth.GetSession(c)

	// Проверяем, что пользователь — участник
	isMember, err := data.IsMember(session.UserID, orgID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	members, err := data.GetMembersWithUserInfo(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// HandleAddMember добавляет участника в организацию
func HandleAddMember(c *gin.Context) {
	idStr := c.Param("id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	session := auth.GetSession(c)

	// Проверяем право приглашать (owner или administrator)
	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner or administrator can add members"})
		return
	}

	// Нельзя назначить owner
	if data.Role(req.Role) == data.RoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot assign owner role"})
		return
	}

	// Валидируем роль
	if data.Role(req.Role) != data.RoleAdministrator && data.Role(req.Role) != data.RoleMember {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role. allowed: administrator, member"})
		return
	}

	// Проверяем, не состоит ли уже в организации
	isMember, err := data.IsMember(req.UserID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isMember {
		c.JSON(http.StatusConflict, gin.H{"error": "user is already a member"})
		return
	}

	member := data.OrganizationMember{
		MemberID:       req.UserID,
		OrganizationID: orgID,
		Role:           data.Role(req.Role),
		JoinedAt:       time.Now(),
	}

	err = data.AddMember(member)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"memberId":       member.MemberID,
		"organizationId": member.OrganizationID,
		"role":           member.Role,
		"joinedAt":       member.JoinedAt,
	})
}

// HandleUpdateMemberRole обновляет роль участника
func HandleUpdateMemberRole(c *gin.Context) {
	orgIDStr := c.Param("id")
	userIDStr := c.Param("userId")

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	session := auth.GetSession(c)

	// Проверяем право (owner или administrator)
	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner or administrator can change roles"})
		return
	}

	// Нельзя менять роль owner
	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	if targetMember.Role == data.RoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot change owner role"})
		return
	}

	// Нельзя назначить owner
	if data.Role(req.Role) == data.RoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot assign owner role"})
		return
	}

	// Валидируем роль
	if data.Role(req.Role) != data.RoleAdministrator && data.Role(req.Role) != data.RoleMember {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role. allowed: administrator, member"})
		return
	}

	err = data.UpdateMemberRole(orgID, userID, data.Role(req.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandleRemoveMember удаляет участника из организации
func HandleRemoveMember(c *gin.Context) {
	orgIDStr := c.Param("id")
	userIDStr := c.Param("userId")

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	session := auth.GetSession(c)

	// Проверяем право (owner или administrator)
	currentMember, err := data.GetMember(orgID, session.UserID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if currentMember.Role != data.RoleOwner && currentMember.Role != data.RoleAdministrator {
		c.JSON(http.StatusForbidden, gin.H{"error": "only owner or administrator can remove members"})
		return
	}

	// Нельзя удалить owner
	targetMember, err := data.GetMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	if targetMember.Role == data.RoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot remove owner"})
		return
	}

	err = data.RemoveMember(orgID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
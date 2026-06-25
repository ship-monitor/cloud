package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

// AddMember добавляет пользователя в организацию с ролью и синхронизирует SpiceDB.
func AddMember(member domain.OrganizationMember) error {
	// Сохраняем в БД
	_, err := db.DB.NewInsert().Model(&member).Exec(context.Background())
	if err != nil {
		return fmt.Errorf("insert member: %w", err)
	}

	return nil
}

// UpdateMemberRole обновляет роль участника.
func UpdateMemberRole(orgID, userID uuid.UUID, role domain.Role) error {
	_, err := db.DB.NewUpdate().
		Model((*domain.OrganizationMember)(nil)).
		Set("role = ?", role).
		Where("organization_id = ? AND member_id = ?", orgID, userID).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}

	return nil
}

// RemoveMember удаляет участника из организации.
func RemoveMember(orgID, userID uuid.UUID) error {
	_, err := db.DB.NewDelete().
		Model((*domain.OrganizationMember)(nil)).
		Where("organization_id = ? AND member_id = ?", orgID, userID).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}

	return nil
}

// GetMember возвращает информацию об участнике.
func GetMember(orgID, userID uuid.UUID) (*domain.OrganizationMember, error) {
	var member domain.OrganizationMember

	err := db.DB.NewSelect().
		Model(&member).
		Where("organization_id = ? AND member_id = ?", orgID, userID).
		Scan(context.Background())
	if err != nil {
		return nil, errors.New("member not found")
	}

	return &member, nil
}

// GetMembersWithUserInfo возвращает участников организации с информацией о пользователях.
func GetMembersWithUserInfo(orgID uuid.UUID) ([]MemberWithUser, error) {
	var members []MemberWithUser

	err := db.DB.NewSelect().
		TableExpr("organization_members AS om").
		ColumnExpr("om.member_id, om.organization_id, om.role, om.joined_at, u.name, u.email").
		Join("JOIN users AS u ON u.id = om.member_id").
		Where("om.organization_id = ?", orgID).
		Order("om.joined_at ASC").
		Scan(context.Background(), &members)
	if err != nil {
		return nil, fmt.Errorf("select members: %w", err)
	}

	return members, nil
}

// MemberWithUser — участник с данными пользователя.
type MemberWithUser struct {
	MemberID       uuid.UUID   `json:"memberId"`
	OrganizationID uuid.UUID   `json:"organizationId"`
	Role           domain.Role `json:"role"`
	JoinedAt       time.Time   `json:"joinedAt"`
	Name           string      `json:"name"`
	Email          string      `json:"email"`
}

// UpdateOrganization обновляет название организации.
func UpdateOrganization(org *domain.Organization, name string) (*domain.Organization, error) {
	org.Name = name
	org.UpdatedAt = time.Now()

	_, err := db.DB.NewUpdate().
		Model(org).
		Column("name", "updated_at").
		Where("id = ?", org.ID).
		Exec(context.Background())
	if err != nil {
		return nil, fmt.Errorf("update organization: %w", err)
	}

	return GetOrganization(org.ID)
}

// DeleteOrganization удаляет организацию.
func DeleteOrganization(id uuid.UUID) error {
	_, err := db.DB.NewDelete().
		Model((*domain.OrganizationMember)(nil)).
		Where("organization_id = ?", id).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("delete organization members: %w", err)
	}

	_, err = db.DB.NewDelete().
		Model((*domain.Organization)(nil)).
		Where("id = ?", id).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	return nil
}

// GetOrganizationsByMember возвращает все организации пользователя.
func GetOrganizationsByMember(userID uuid.UUID) ([]domain.Organization, error) {
	var orgs []domain.Organization

	err := db.DB.NewSelect().
		Model(&orgs).
		Join("JOIN organization_members AS om ON om.organization_id = organization.id").
		Where("om.member_id = ?", userID).
		Order("organization.created_at DESC").
		Scan(context.Background())
	if err != nil {
		return nil, fmt.Errorf("select organizations: %w", err)
	}

	return orgs, nil
}

// IsMember проверяет, является ли пользователь участником организации.
func IsMember(userID, orgID uuid.UUID) (bool, error) {
	exists, err := db.DB.NewSelect().
		Model((*domain.OrganizationMember)(nil)).
		Where("member_id = ? AND organization_id = ?", userID, orgID).
		Exists(context.Background())
	if err != nil {
		return false, fmt.Errorf("select member: %w", err)
	}

	return exists, nil
}

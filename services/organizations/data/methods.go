package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

// AddMember добавляет пользователя в организацию
func AddMember(member OrganizationMember) error {
	_, err := db.DB.NewInsert().Model(&member).Exec(context.Background())
	return err
}

// UpdateOrganization обновляет название организации
func UpdateOrganization(id uuid.UUID, name string) (*Organization, error) {
	org := &Organization{ID: id, Name: name, UpdatedAt: time.Now()}
	_, err := db.DB.NewUpdate().
		Model(org).
		Column("name", "updated_at").
		Where("id = ?", id).
		Exec(context.Background())
	if err != nil {
		return nil, err
	}
	return GetOrganization(id)
}

// DeleteOrganization удаляет организацию
func DeleteOrganization(id uuid.UUID) error {
	// Сначала удаляем всех участников
	_, err := db.DB.NewDelete().
		Model((*OrganizationMember)(nil)).
		Where("organization_id = ?", id).
		Exec(context.Background())
	if err != nil {
		return err
	}
	
	// Потом удаляем организацию
	_, err = db.DB.NewDelete().
		Model((*Organization)(nil)).
		Where("id = ?", id).
		Exec(context.Background())
	return err
}

// GetOrganizationsByMember возвращает все организации пользователя
func GetOrganizationsByMember(userID uuid.UUID) ([]Organization, error) {
	var orgs []Organization
	err := db.DB.NewSelect().
		Model(&orgs).
		Join("JOIN organization_member AS om ON om.organization_id = organization.id").
		Where("om.member_id = ?", userID).
		Order("organization.created_at DESC").
		Scan(context.Background())
	return orgs, err
}

// IsMember проверяет, является ли пользователь участником организации
func IsMember(userID, orgID uuid.UUID) (bool, error) {
	exists, err := db.DB.NewSelect().
		Model((*OrganizationMember)(nil)).
		Where("member_id = ? AND organization_id = ?", userID, orgID).
		Exists(context.Background())
	return exists, err
}

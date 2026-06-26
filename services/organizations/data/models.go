package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

func GetOrganization(id uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization

	err := db.DB.NewSelect().Model(&org).Where("id = ?", id).Scan(context.Background())
	if err != nil {
		return nil, fmt.Errorf("select organization: %w", err)
	}

	return &org, nil
}

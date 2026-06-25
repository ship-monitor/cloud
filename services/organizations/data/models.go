package data

import (
	"context"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/go-playground/validator/v10"
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

type CreateOrganizationInput struct {
	Name      string    `json:"name"      validate:"required"`
	CreatorID uuid.UUID `json:"creatorId" validate:"required"`
}

func CreateOrganization(in CreateOrganizationInput) (domain.Organization, error) {
	log.Debug("Creating organization", "name", in.Name, "creatorId", in.CreatorID)

	err := validator.New().Struct(in)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("failed validate input: %w", err)
	}

	org := domain.Organization{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      in.Name,
		CreatorID: in.CreatorID,
	}

	_, err = db.DB.NewInsert().Model(&org).Exec(context.Background())
	if err != nil {
		return domain.Organization{}, fmt.Errorf("insert organization: %w", err)
	}

	return org, nil
}

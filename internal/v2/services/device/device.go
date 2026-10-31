// Package device contains the device business logic.
package device

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
)

var (
	// ErrNotFound is returned when the device does not exist,
	// or exists but is owned by another user.
	ErrNotFound = errors.New("device not found")

	// ErrAlreadyExists is returned when a device with the given id
	// was already added.
	ErrAlreadyExists = errors.New("device already exists")
)

// Repository is the persistence interface of the Service.
type Repository interface {
	Create(ctx context.Context, d domain.Device) error
	Get(ctx context.Context, id uuid.UUID) (domain.Device, error)
	ListByOwner(
		ctx context.Context,
		ownerID uuid.UUID,
		paging domain.Paging,
	) ([]domain.Device, error)
	Rename(ctx context.Context, id uuid.UUID, name string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service implements device use cases.
type Service struct {
	repo Repository
}

// New returns a Service backed by the given repository.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the device with the given id, if it is owned by ownerID.
//
//nolint:varnamelen
func (s *Service) Get(
	ctx context.Context,
	ownerID, id uuid.UUID,
) (domain.Device, error) {
	d, err := s.repo.Get(ctx, id)
	if errors.Is(err, devicerepo.ErrNotFound) {
		return domain.Device{}, ErrNotFound
	}

	if err != nil {
		return domain.Device{}, fmt.Errorf("get device: %w", err)
	}

	// Do not leak other users' devices.
	if d.OwnerID != ownerID {
		return domain.Device{}, ErrNotFound
	}

	return d, nil
}

// ListByOwner returns a page of devices owned by the given user.
func (s *Service) ListByOwner(
	ctx context.Context,
	ownerID uuid.UUID,
	paging domain.Paging,
) ([]domain.Device, error) {
	if paging.Size == 0 {
		paging.Size = domain.DefaultPageSize
	}

	devices, err := s.repo.ListByOwner(ctx, ownerID, paging)
	if err != nil {
		return nil, fmt.Errorf("select devices by owner: %w", err)
	}

	return devices, nil
}

// Add connects a new device to the given user.
//
//nolint:varnamelen
func (s *Service) Add(
	ctx context.Context,
	ownerID, id uuid.UUID,
	securityKey string,
) error {
	_, err := s.repo.Get(ctx, id)
	if err == nil {
		return ErrAlreadyExists
	}

	if !errors.Is(err, devicerepo.ErrNotFound) {
		return fmt.Errorf("get device: %w", err)
	}

	d := domain.Device{
		ID:          id,
		OwnerID:     ownerID,
		Name:        "",
		SecurityKey: securityKey,
	}

	if err := s.repo.Create(ctx, d); err != nil {
		return fmt.Errorf("create device: %w", err)
	}

	return nil
}

// Rename updates the name of the device with the given id,
// if it is owned by ownerID.
func (s *Service) Rename(
	ctx context.Context,
	ownerID, id uuid.UUID,
	name string,
) error {
	if _, err := s.Get(ctx, ownerID, id); err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	if err := s.repo.Rename(ctx, id, name); err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}

func (s *Service) Remove(
	ctx context.Context,
	deviceID uuid.UUID,
	userID uuid.UUID,
) error {
	if _, err := s.Get(ctx, userID, deviceID); err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	err := s.repo.Delete(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	return nil
}

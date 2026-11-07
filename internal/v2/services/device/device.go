// Package device contains the device business logic.
package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/ship-monitor/cloud/pkg/connector/schemas/messages"
	"github.com/ship-monitor/cloud/pkg/hub"
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
//
// All methods that take a device id are scoped by ownerID,
// so devices of other users are reported as not found.
type Repository interface {
	Create(ctx context.Context, d domain.Device) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	Get(ctx context.Context, ownerID, id uuid.UUID) (domain.Device, error)
	ListByOwner(
		ctx context.Context,
		ownerID uuid.UUID,
		paging domain.Paging,
	) ([]domain.Device, error)
	Rename(ctx context.Context, ownerID, id uuid.UUID, name string) error
	Delete(ctx context.Context, ownerID, id uuid.UUID) error
}

type AliveRepository interface {
	GetLastHeartbeat(
		ctx context.Context,
		deviceID uuid.UUID,
	) (*hub.Heartbeat, error)
}
type ConnRequestRepository interface {
	Create(
		ctx context.Context,
		request domain.ConnectRequest,
	) error
	Get(
		ctx context.Context,
		id uuid.UUID,
	) (domain.ConnectRequest, error)
	SetApproved(
		ctx context.Context,
		id uuid.UUID,
		approved bool,
	) error
}
type Publisher interface {
	PublishConnectRequest(
		ctx context.Context,
		r *messages.ConnectRequest,
	) error
}

// Service implements device use cases.
type Service struct {
	repo  Repository
	alive AliveRepository
	pub   Publisher
	conn  ConnRequestRepository
}

// New returns a Service backed by the given repository.
func New(
	repo Repository,
	alive AliveRepository,
	pub Publisher,
	conn ConnRequestRepository,
) *Service {
	return &Service{
		repo:  repo,
		alive: alive,
		pub:   pub,
		conn:  conn,
	}
}

// Get returns the device with the given id, if it is owned by the actor.
func (s *Service) Get(
	ctx context.Context,
	actor auth.Actor,
	id uuid.UUID,
) (domain.Device, error) {
	device, err := s.repo.Get(ctx, actor.UserID, id)
	if errors.Is(err, devicerepo.ErrNotFound) {
		return domain.Device{}, ErrNotFound
	}

	if err != nil {
		return domain.Device{}, fmt.Errorf("get device: %w", err)
	}

	return device, nil
}

// ListByOwner returns a page of devices owned by the actor.
func (s *Service) ListByOwner(
	ctx context.Context,
	actor auth.Actor,
	paging domain.Paging,
) ([]domain.Device, error) {
	if paging.Size == 0 {
		paging.Size = domain.DefaultPageSize
	}

	devices, err := s.repo.ListByOwner(ctx, actor.UserID, paging)
	if err != nil {
		return nil, fmt.Errorf("select devices by owner: %w", err)
	}

	return devices, nil
}

// Connect creates new connect request.
func (s *Service) Connect(
	ctx context.Context,
	actor auth.Actor,
	id uuid.UUID,
	securityKey string,
) error {
	exists, err := s.repo.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("check device exists: %w", err)
	}

	if exists {
		return ErrAlreadyExists
	}

	request := messages.ConnectRequest{
		RequestID:   uuid.New(),
		SecurityKey: securityKey,
		DeviceID:    id,
	}

	err = s.conn.Create(ctx, domain.ConnectRequest{
		ID:        request.RequestID,
		DeviceID:  id,
		CreatedAt: time.Now(),
		Approved:  false,
		UserID:    actor.UserID,
	})
	if err != nil {
		return fmt.Errorf("create connect request: %w", err)
	}

	err = s.pub.PublishConnectRequest(ctx, &request)
	if err != nil {
		return fmt.Errorf("publish connect request: %w", err)
	}

	return nil
}

// Rename updates the name of the device with the given id,
// if it is owned by the actor.
func (s *Service) Rename(
	ctx context.Context,
	actor auth.Actor,
	id uuid.UUID,
	name string,
) error {
	err := s.repo.Rename(ctx, actor.UserID, id, name)
	if errors.Is(err, devicerepo.ErrNotFound) {
		return ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}

// Remove deletes the device with the given id, if it is owned by the actor.
func (s *Service) Remove(
	ctx context.Context,
	actor auth.Actor,
	id uuid.UUID,
) error {
	err := s.repo.Delete(ctx, actor.UserID, id)
	if errors.Is(err, devicerepo.ErrNotFound) {
		return ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	return nil
}

// PingTimeout used to detect online status.
// It is timeout between hub sending heartbeats.
const PingTimeout = time.Minute * 2

// IsDeviceOnline reports whether the device owned by the actor
// has sent a heartbeat recently.
func (s *Service) IsDeviceOnline(
	ctx context.Context,
	actor auth.Actor,
	id uuid.UUID,
) (bool, time.Time, error) {
	if _, err := s.Get(ctx, actor, id); err != nil {
		return false, time.Time{}, fmt.Errorf("get device: %w", err)
	}

	heartbeat, err := s.alive.GetLastHeartbeat(ctx, id)
	if err != nil {
		return false, time.Time{}, fmt.Errorf("get last heartbeat: %w", err)
	}

	if heartbeat.Timestamp.Add(PingTimeout).After(time.Now()) {
		return true, heartbeat.Timestamp, nil
	}

	return false, heartbeat.Timestamp, nil
}

// ApproveDuration represents time in which request can be approved.
const ApproveDuration = time.Minute

var errRequestExpired = errors.New("connect request expired")

// ApproveConnection approves connection by connect request ID.
func (s *Service) ApproveConnection(
	ctx context.Context,
	requestID uuid.UUID,
) error {
	request, err := s.conn.Get(ctx, requestID)
	if err != nil {
		return fmt.Errorf("get connect request: %w", err)
	}

	if request.Approved {
		return nil
	}

	if request.CreatedAt.Add(ApproveDuration).After(time.Now()) {
		return errRequestExpired
	}

	err = s.repo.Create(ctx, domain.Device{
		ID:          request.DeviceID,
		OwnerID:     request.UserID,
		Name:        "NEW DEVICE " + request.DeviceID.String(),
		SecurityKey: "",
	})
	if err != nil {
		return fmt.Errorf("create new device: %w", err)
	}

	err = s.conn.SetApproved(ctx, requestID, true)
	if err != nil {
		return fmt.Errorf("approve connect request: %w", err)
	}

	return nil
}

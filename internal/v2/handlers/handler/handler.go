// Package handler implements the generated [api.Handler].
//
//nolint:varnamelen
package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	userservice "github.com/ship-monitor/cloud/internal/v2/services/user"
)

var _ api.Handler = (*Handler)(nil)

type Handler struct {
	tokenService TokenService
	users        UserService
	devices      DeviceService
}

func New(
	tokenService TokenService,
	users UserService,
	devices DeviceService,
) *Handler {
	return &Handler{
		tokenService: tokenService,
		users:        users,
		devices:      devices,
	}
}

var errNoPrincipalInCtx = errors.New("no authenticated principal in context")

// currentUserID returns the id of the authenticated user from ctx.
func currentUserID(ctx context.Context) (uuid.UUID, error) {
	principal := auth.GetPrincipal(ctx)
	if principal == nil {
		return uuid.UUID{}, errNoPrincipalInCtx
	}

	return principal.UserID(), nil
}

// RegisterUser implements [api.Handler].
func (h *Handler) RegisterUser(
	ctx context.Context,
	req *api.RegisterUserReq,
) error {
	_, err := h.users.Register(ctx, req.Email, req.Password, req.Name)
	if errors.Is(err, userservice.ErrEmailTaken) {
		return errEmailTaken
	}

	if err != nil {
		return errRegisterUser.WithDetail(err.Error())
	}

	return nil
}

// LoginUser implements [api.Handler].
func (h *Handler) LoginUser(
	ctx context.Context,
	req *api.LoginUserReq,
) (api.LoginUserRes, error) {
	u, err := h.users.Login(ctx, req.Email, req.Password)
	if errors.Is(err, userservice.ErrInvalidCredentials) {
		return nil, errInvalidCredentials
	}

	if err != nil {
		return nil, errLogin.WithDetail(err.Error())
	}

	token, _, err := h.tokenService.CreateTokens(ctx, &u)
	if err != nil {
		return nil, errLogin.WithDetail(
			fmt.Errorf("create tokens: %w", err).Error(),
		)
	}

	return &api.LoginUserOK{
		Success: true,
		Token:   api.NewOptString(token),
	}, nil
}

// GetDevices implements [api.Handler].
func (h *Handler) GetDevices(ctx context.Context) (*api.GetDevicesOK, error) {
	ownerID, err := currentUserID(ctx)
	if err != nil {
		return nil, errUnauthenticated
	}

	devices, err := h.devices.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}

	return &api.GetDevicesOK{
		Devices: toAPIDevices(devices),
	}, nil
}

// AddDevice implements [api.Handler].
func (h *Handler) AddDevice(
	ctx context.Context,
	req *api.AddDeviceReq,
) (api.AddDeviceRes, error) {
	ownerID, err := currentUserID(ctx)
	if err != nil {
		return nil, errUnauthenticated
	}

	deviceID, err := uuid.Parse(req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	err = h.devices.Add(ctx, ownerID, deviceID, req.SecurityKey)
	if errors.Is(err, deviceservice.ErrAlreadyExists) {
		return &api.AddDeviceConflict{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("add device: %w", err)
	}

	return &api.AddDeviceCreated{}, nil
}

// GetDevice implements [api.Handler].
func (h *Handler) GetDevice(
	ctx context.Context,
	params api.GetDeviceParams,
) (*api.Device, error) {
	ownerID, err := currentUserID(ctx)
	if err != nil {
		return nil, errUnauthenticated
	}

	deviceID, err := uuid.Parse(params.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("bad reqest: %w", err)
	}

	d, err := h.devices.Get(ctx, ownerID, deviceID)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return nil, fmt.Errorf("not found: %w", err)
	}

	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	return toAPIDevice(d), nil
}

// EditDevice implements [api.Handler].
func (h *Handler) EditDevice(
	ctx context.Context,
	req *api.EditDeviceReq,
	params api.EditDeviceParams,
) error {
	ownerID, err := currentUserID(ctx)
	if err != nil {
		return errUnauthenticated
	}

	deviceID, err := uuid.Parse(params.DeviceId)
	if err != nil {
		return fmt.Errorf("bad request: %w", err)
	}

	err = h.devices.Rename(ctx, ownerID, deviceID, req.Name)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return fmt.Errorf("not found: %w", err)
	}

	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}

// GetDeviceMetrics implements [api.Handler].
func (h *Handler) GetDeviceMetrics(
	ctx context.Context,
	params api.GetDeviceMetricsParams,
) (api.GetDeviceMetricsRes, error) {
	return nil, errors.New("not implemented")
}

func toAPIDevice(d devicedomain.Device) *api.Device {
	return &api.Device{
		ID:      d.ID.String(),
		Name:    d.Name,
		OwnerID: api.NewOptString(d.OwnerID.String()),
	}
}

func toAPIDevices(devices []devicedomain.Device) []api.Device {
	out := make([]api.Device, len(devices))
	for i, d := range devices {
		out[i] = *toAPIDevice(d)
	}

	return out
}

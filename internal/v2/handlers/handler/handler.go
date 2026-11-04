// Package handler implements the generated [api.Handler].
//
//nolint:varnamelen
package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	userservice "github.com/ship-monitor/cloud/internal/v2/services/user"
	"go.uber.org/zap"
)

var _ api.Handler = (*Handler)(nil)

type Handler struct {
	tokenService TokenService
	users        UserService
	devices      DeviceService
	logger       *zap.Logger
}

// RefreshToken implements [api.Handler].
func (h *Handler) RefreshToken(
	ctx context.Context,
	req *api.RefreshTokenReq,
) (*api.RefreshResponseOK, error) {
	token, refresh, err := h.tokenService.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, errRefreshToken.WithDetail(err.Error())
	}

	return &api.RefreshResponseOK{
		Success:      true,
		Token:        token,
		RefreshToken: refresh,
	}, nil
}

func New(
	tokenService TokenService,
	users UserService,
	devices DeviceService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		tokenService: tokenService,
		users:        users,
		devices:      devices,
		logger:       logger,
	}
}

// currentActor returns the authenticated user from ctx,
// or the unauthenticated problem when the request is anonymous.
func currentActor(ctx context.Context) (auth.Actor, error) {
	actor, ok := auth.ActorFromContext(ctx)
	if !ok {
		return auth.Actor{}, errUnauthenticated
	}

	return actor, nil
}

// RemoveDevice implements [api.Handler].
func (h *Handler) RemoveDevice(
	ctx context.Context,
	params api.RemoveDeviceParams,
) error {
	actor, err := currentActor(ctx)
	if err != nil {
		return fmt.Errorf("get actor: %w", err)
	}

	err = h.devices.Remove(ctx, actor, params.DeviceID)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return errDeviceNotFound
	}

	if err != nil {
		return fmt.Errorf("remove device: %w", err)
	}

	return nil
}

// RegisterUser implements [api.Handler].
func (h *Handler) RegisterUser(
	ctx context.Context,
	req *api.RegisterForm,
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
) (*api.LoginResponseOK, error) {
	u, err := h.users.Login(ctx, req.Email, req.Password)
	if errors.Is(err, userservice.ErrInvalidCredentials) {
		return nil, errInvalidCredentials
	}

	if err != nil {
		return nil, errLogin.WithDetail(err.Error())
	}

	token, refresh, err := h.tokenService.CreateTokens(ctx, &u)
	if err != nil {
		return nil, errLogin.WithDetail(
			fmt.Errorf("create tokens: %w", err).Error(),
		)
	}

	return &api.LoginResponseOK{
		Success:      true,
		Token:        token,
		RefreshToken: refresh,
	}, nil
}

// GetDevices implements [api.Handler].
func (h *Handler) GetDevices(
	ctx context.Context,
	params api.GetDevicesParams,
) (*api.GetDevicesOK, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	paging, err := pagingFromParams(params)
	if err != nil {
		return nil, fmt.Errorf("paging: %w", err)
	}

	devices, err := h.devices.ListByOwner(ctx, actor, paging)
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
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	err = h.devices.Add(ctx, actor, req.DeviceID, req.SecurityKey)
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
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	d, err := h.devices.Get(ctx, actor, params.DeviceID)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return nil, errDeviceNotFound
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
	actor, err := currentActor(ctx)
	if err != nil {
		return fmt.Errorf("get actor: %w", err)
	}

	err = h.devices.Rename(ctx, actor, params.DeviceID, req.Name)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return errDeviceNotFound
	}

	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}

// GetDeviceOnlineStatus implements [api.Handler].
func (h *Handler) GetDeviceOnlineStatus(
	ctx context.Context,
	params api.GetDeviceOnlineStatusParams,
) (*api.GetOnlineResponseOK, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	online, lastSeen, err := h.devices.IsDeviceOnline(
		ctx,
		actor,
		params.DeviceID,
	)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return nil, errDeviceNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get device online status: %w", err)
	}

	return &api.GetOnlineResponseOK{
		Online:   online,
		LastSeen: lastSeen.Unix(),
	}, nil
}

// GetDeviceMetrics implements [api.Handler].
func (h *Handler) GetDeviceMetrics(
	ctx context.Context,
	params api.GetDeviceMetricsParams,
) (api.GetDeviceMetricsRes, error) {
	return nil, errUnknownError.WithDetail("Currently not implemented")
}

// pagingFromParams converts API pagination parameters to domain paging.
//
// The generated code enforces page >= 0 and pageSize >= 1,
// but not the maximum page size or integer values, so both are checked here.
func pagingFromParams(
	params api.GetDevicesParams,
) (devicedomain.Paging, error) {
	paging := devicedomain.Paging{
		Page: 0,
		Size: devicedomain.DefaultPageSize,
	}

	if v, ok := params.Page.Get(); ok {
		paging.Page = v
	}

	if v, ok := params.PageSize.Get(); ok {
		if v > devicedomain.MaxPageSize {
			return paging, errInvalidPagination
		}

		paging.Size = v
	}

	return paging, nil
}

func toAPIDevice(d devicedomain.Device) *api.Device {
	return &api.Device{
		ID:      d.ID,
		Name:    d.Name,
		OwnerID: d.OwnerID,
	}
}

func toAPIDevices(devices []devicedomain.Device) []api.Device {
	out := make([]api.Device, len(devices))
	for i, d := range devices {
		out[i] = *toAPIDevice(d)
	}

	return out
}

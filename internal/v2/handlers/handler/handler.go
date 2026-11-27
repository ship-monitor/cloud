// Package handler implements the generated [api.Handler].
//
//nolint:varnamelen
package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"go.uber.org/zap"
)

var _ api.Handler = (*Handler)(nil)

type Handler struct {
	tokenService  TokenService
	users         UserService
	devices       DeviceService
	sensors       SensorService
	metrics       MetricsService
	notifications NotificationService
	logger        *zap.Logger
}

// RefreshToken implements [api.Handler].
func (h *Handler) RefreshToken(
	ctx context.Context,
	req *api.RefreshTokenReq,
) (*api.RefreshResponseOK, error) {
	token, refresh, err := h.tokenService.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, errRefreshToken
	}

	return &api.RefreshResponseOK{
		Token:        token,
		RefreshToken: refresh,
	}, nil
}

func New(
	tokenService TokenService,
	users UserService,
	devices DeviceService,
	sensors SensorService,
	metrics MetricsService,
	notifications NotificationService,
	logger *zap.Logger,
) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Handler{
		tokenService:  tokenService,
		users:         users,
		devices:       devices,
		sensors:       sensors,
		metrics:       metrics,
		notifications: notifications,
		logger:        logger,
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
	if errors.Is(err, device.ErrNotFound) {
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
	if errors.Is(err, user.ErrEmailTaken) {
		return errEmailTaken
	}

	if err != nil {
		h.logger.Error("Register user", zap.Error(err))

		return errRegisterUser
	}

	return nil
}

// LoginUser implements [api.Handler].
func (h *Handler) LoginUser(
	ctx context.Context,
	req *api.LoginUserReq,
) (*api.LoginResponseOK, error) {
	u, err := h.users.Login(ctx, req.Email, req.Password)
	if errors.Is(err, user.ErrInvalidCredentials) {
		return nil, errInvalidCredentials
	}

	if err != nil {
		return nil, errLogin
	}

	token, refresh, err := h.tokenService.CreateTokens(ctx, &u)
	if err != nil {
		return nil, errLogin
	}

	return &api.LoginResponseOK{
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

// ConnectDevice implements [api.Handler].
func (h *Handler) ConnectDevice(
	ctx context.Context,
	req *api.ConnectDeviceReq,
) (api.ConnectDeviceRes, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	err = h.devices.Connect(ctx, actor, req.DeviceID, req.SecurityKey)
	if errors.Is(err, device.ErrAlreadyExists) {
		return new(
			toApiProblem(errAlreadyConnected),
		), nil
	}

	if err != nil {
		return nil, fmt.Errorf("add device: %w", err)
	}

	return &api.ConnectDeviceAccepted{}, nil
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
	if errors.Is(err, device.ErrNotFound) {
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
	if errors.Is(err, device.ErrNotFound) {
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
	if errors.Is(err, device.ErrNotFound) {
		return nil, errDeviceNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get device online status: %w", err)
	}

	return &api.GetOnlineResponseOK{
		Online:   online,
		LastSeen: lastSeen,
	}, nil
}

// GetHealth implements [api.Handler].
func (h *Handler) GetHealth(ctx context.Context) (*api.HealthResponse, error) {
	return &api.HealthResponse{
		Status: api.ServiceStatusOk,
	}, nil
}

// GetSensors implements [api.Handler].
func (h *Handler) GetSensors(
	ctx context.Context,
	params api.GetSensorsParams,
) (*api.GetSensorsOK, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	sensors, err := h.sensors.GetSensors(ctx, actor, params.DeviceID)
	if err != nil {
		if errors.Is(err, sensor.ErrNotFound) {
			return nil, errSensorsNotFound
		}

		return nil, fmt.Errorf("get sensors: %w", err)
	}

	response := api.GetSensorsOK{
		Sensors: make([]api.Sensor, 0, len(sensors)),
	}

	for _, s := range sensors {
		response.Sensors = append(response.Sensors, *toApiSensor(&s))
	}

	return &response, nil
}

// EditSensor implements [api.Handler].
func (h *Handler) EditSensor(
	ctx context.Context,
	req *api.EditSensorReq,
	params api.EditSensorParams,
) (*api.EditSensorOK, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	err = h.sensors.Rename(
		ctx,
		actor,
		params.SensorID,
		params.DeviceID,
		req.DisplayName,
	)
	if errors.Is(err, sensor.ErrNotFound) ||
		errors.Is(err, device.ErrNotFound) {
		return nil, errSensorsNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("rename: %w", err)
	}

	s, err := h.sensors.GetSensor(
		ctx,
		actor,
		params.SensorID,
		params.DeviceID,
	)
	if errors.Is(err, sensor.ErrNotFound) ||
		errors.Is(err, device.ErrNotFound) {
		return nil, errSensorsNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get updated sensor: %w", err)
	}

	return &api.EditSensorOK{
		Sensor: *toApiSensor(s),
	}, nil
}

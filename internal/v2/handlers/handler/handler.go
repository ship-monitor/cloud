// Package handler implements the generated [api.Handler].
//
//nolint:varnamelen
package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	deviceService "github.com/ship-monitor/cloud/internal/v2/services/device"
	userservice "github.com/ship-monitor/cloud/internal/v2/services/user"
	"go.uber.org/zap"
)

var _ api.Handler = (*Handler)(nil)

type Handler struct {
	tokenService TokenService
	users        UserService
	devices      DeviceService
	sensors      SensorService
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
	sensors SensorService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		tokenService: tokenService,
		users:        users,
		devices:      devices,
		sensors:      sensors,
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
	if errors.Is(err, deviceService.ErrNotFound) {
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
		h.logger.Error("Register user", zap.Error(err))

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
	if errors.Is(err, deviceService.ErrAlreadyExists) {
		return new(
			toApiProblem(errAlreadyConnected.WithDetail(err.Error())),
		), nil
	}

	if err != nil {
		return nil, fmt.Errorf("add device: %w", err)
	}

	return &api.ConnectDeviceCreated{}, nil
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
	if errors.Is(err, deviceService.ErrNotFound) {
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
	if errors.Is(err, deviceService.ErrNotFound) {
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
	if errors.Is(err, deviceService.ErrNotFound) {
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
) (*api.Metrics, error) {
	return nil, errUnknownError.WithDetail("Currently not implemented")
}

// GetSensorMetrics implements [api.Handler].
func (h *Handler) GetSensorMetrics(
	ctx context.Context,
	params api.GetSensorMetricsParams,
) (*api.Metrics, error) {
	return nil, errUnknownError.WithDetail("Currently not implemented")
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
	if err != nil {
		return nil, fmt.Errorf("rename: %w", err)
	}

	sensor, err := h.sensors.GetSensor(
		ctx,
		actor,
		params.SensorID,
		params.DeviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("get updated sensor: %w", err)
	}

	return &api.EditSensorOK{
		Sensor: *toApiSensor(sensor),
	}, nil
}

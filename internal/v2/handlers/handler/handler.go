package handler

import (
	"context"

	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

var _ api.Handler = (*Handler)(nil)

type Handler struct {
	tokenService TokenService
}

// AddDevice implements [api.Handler].
func (h *Handler) AddDevice(
	ctx context.Context,
	req *api.AddDeviceReq,
) (api.AddDeviceRes, error) {
	panic("unimplemented")
}

// EditDevice implements [api.Handler].
func (h *Handler) EditDevice(
	ctx context.Context,
	params api.EditDeviceParams,
) (*api.Problem, error) {
	panic("unimplemented")
}

// GetDevice implements [api.Handler].
func (h *Handler) GetDevice(
	ctx context.Context,
	params api.GetDeviceParams,
) (api.GetDeviceRes, error) {
	panic("unimplemented")
}

// GetDeviceMetrics implements [api.Handler].
func (h *Handler) GetDeviceMetrics(
	ctx context.Context,
	params api.GetDeviceMetricsParams,
) (*api.Problem, error) {
	panic("unimplemented")
}

// GetDevices implements [api.Handler].
func (h *Handler) GetDevices(ctx context.Context) (api.GetDevicesRes, error) {
	panic("unimplemented")
}

// LoginUser implements [api.Handler].
func (h *Handler) LoginUser(
	ctx context.Context,
	req *api.LoginUserReq,
) (api.LoginUserRes, error) {
	panic("unimplemented")
}

// RegisterUser implements [api.Handler].
func (h *Handler) RegisterUser(
	ctx context.Context,
	req *api.RegisterUserReq,
) (api.RegisterUserRes, error) {
	panic("unimplemented")
}

func New(tokenService TokenService) *Handler {
	return &Handler{
		tokenService: tokenService,
	}
}

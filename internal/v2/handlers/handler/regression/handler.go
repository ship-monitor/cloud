package regression

import (
	"github.com/ship-monitor/cloud/internal/v2/handlers/handler"
	"go.uber.org/zap"
)

type handlerConf struct {
	tokenService  handler.TokenService
	users         handler.UserService
	devices       handler.DeviceService
	sensors       handler.SensorService
	metrics       handler.MetricsService
	notifications handler.NotificationService
	logger        *zap.Logger
}

type HandlerOption func(*handlerConf)

func NewHandler(options ...HandlerOption) *handler.Handler {
	var conf handlerConf

	for _, opt := range options {
		opt(&conf)
	}

	return handler.New(
		conf.tokenService,
		conf.users,
		conf.devices,
		conf.sensors,
		conf.metrics,
		conf.notifications,
		conf.logger,
	)
}

func WithToken(token handler.TokenService) HandlerOption {
	return func(hc *handlerConf) {
		hc.tokenService = token
	}
}

func WithUsers(users handler.UserService) HandlerOption {
	return func(hc *handlerConf) {
		hc.users = users
	}
}

func WithSensors(sensors handler.SensorService) HandlerOption {
	return func(hc *handlerConf) {
		hc.sensors = sensors
	}
}

func WithMetrics(metrics handler.MetricsService) HandlerOption {
	return func(hc *handlerConf) {
		hc.metrics = metrics
	}
}

func WithDevices(devices handler.DeviceService) HandlerOption {
	return func(hc *handlerConf) {
		hc.devices = devices
	}
}

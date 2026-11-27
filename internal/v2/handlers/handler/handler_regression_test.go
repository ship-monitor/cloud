package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"github.com/ship-monitor/cloud/internal/v2/handlers/handler"
	"github.com/ship-monitor/cloud/internal/v2/handlers/handler/regression"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	userservice "github.com/ship-monitor/cloud/internal/v2/services/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func regressionRequest(
	t *testing.T,
	h *handler.Handler,
	actor auth.Actor,
	method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	server, err := api.NewServer(h, regression.NewSecurity(actor))
	require.NoError(t, err)

	request := httptest.NewRequestWithContext(
		t.Context(), method, path, strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer test-token")

	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	return response
}

type regressionTokenService struct {
	handler.TokenService

	refreshErr error
	createErr  error
}

func (s *regressionTokenService) RefreshTokens(
	_ context.Context,
	_ string,
) (string, string, error) {
	return "", "", s.refreshErr
}

func (s *regressionTokenService) CreateTokens(
	_ context.Context,
	_ *user.User,
) (string, string, error) {
	return "", "", s.createErr
}

type regressionDeviceGetService struct {
	handler.DeviceService

	err   error
	calls int
}

func (s *regressionDeviceGetService) Get(
	_ context.Context,
	_ auth.Actor,
	_ uuid.UUID,
) (devicedomain.Device, error) {
	s.calls++

	return devicedomain.Device{}, s.err
}

func TestRegression_GenericErrorResponseDoesNotExposeDetails(t *testing.T) {
	t.Parallel()

	const secret = "internal-database-secret-123"

	service := &regressionDeviceGetService{err: errors.New(secret)}
	response := regressionRequest(
		t,
		regression.NewHandler(regression.WithDevices(service)),
		auth.Actor{UserID: uuid.New()},
		http.MethodGet,
		"/api/v2/devices/"+uuid.New().String(),
		"",
	)
	require.Equal(t, 1, service.calls)
	require.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Contains(
		t,
		response.Header().Get("Content-Type"),
		"application/problem+json",
	)
	assert.NotContains(t, response.Body.String(), secret)

	var body struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Type)
	assert.Equal(t, http.StatusInternalServerError, body.Status)
	assert.NotContains(t, body.Detail, secret)
}

func TestRegression_UnexpectedErrorsAreNotExposed(t *testing.T) {
	t.Parallel()

	const secret = "database-password-secret-123"

	failure := errors.New(secret)

	cases := []struct {
		name   string
		path   string
		body   string
		h      *handler.Handler
		status int
	}{
		{
			name: "registration",
			path: "/api/v2/auth/register",
			body: `{"email":"user@example.com","password":"password","name":"User"}`,
			h: regression.NewHandler(
				regression.WithUsers(
					regression.NewUserServiceWithRegisterErr(
						failure,
					)),
			),
			status: http.StatusInternalServerError,
		},
		{
			name: "login",
			path: "/api/v2/auth/login",
			body: `{"email":"user@example.com","password":"password"}`,
			h: regression.NewHandler(
				regression.WithUsers(

					regression.NewUserServiceWithLoginErr(failure),
				),
			),
			status: http.StatusInternalServerError,
		},
		{
			name: "login token creation",
			path: "/api/v2/auth/login",
			body: `{"email":"user@example.com","password":"password"}`,
			h: regression.NewHandler(
				regression.WithToken(

					&regressionTokenService{createErr: failure},
				),
				regression.WithUsers(
					&regression.UserService{},
				),
			),
			status: http.StatusInternalServerError,
		},
		{
			name: "refresh",
			path: "/api/v2/auth/refresh",
			body: `{"refreshToken":"token"}`,
			h: regression.NewHandler(
				regression.WithToken(&regressionTokenService{
					refreshErr: failure,
				}),
			),
			status: http.StatusUnauthorized,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			response := regressionRequest(
				t, tc.h, auth.Actor{}, http.MethodPost, tc.path, tc.body,
			)
			assert.Equal(t, tc.status, response.Code)
			assert.NotContains(t, response.Body.String(), secret)

			var body struct {
				Type   string `json:"type"`
				Status int    `json:"status"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			assert.NotEmpty(t, body.Type)
			assert.Equal(t, tc.status, body.Status)
		})
	}
}

func TestRegression_RegisterValidationAndDuplicate(t *testing.T) {
	t.Parallel()

	t.Run("invalid email rejected before service", func(t *testing.T) {
		t.Parallel()

		form := api.RegisterForm{
			Email:    "not an email@@example.com",
			Password: "password",
			Name:     "User",
		}
		require.Error(t, form.Validate())

		service := &regression.UserService{}
		response := regressionRequest(
			t,
			regression.NewHandler(regression.WithUsers(service)),
			auth.Actor{},
			http.MethodPost,
			"/api/v2/auth/register",
			`{"email":"not an email@@example.com","password":"password","name":"User"}`,
		)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Zero(t, service.Calls())
	})

	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: "empty password",
			body: `{"email":"user@example.com","password":"","name":"User"}`,
		},
		{
			name: "empty name",
			body: `{"email":"user@example.com","password":"password","name":""}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &regression.UserRepository{}
			response := regressionRequest(
				t,
				regression.NewHandler(
					regression.WithUsers(userservice.New(repo)),
				),
				auth.Actor{},
				http.MethodPost,
				"/api/v2/auth/register",
				tc.body,
			)
			logResponse(t, response)
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Zero(t, repo.Checks())
			assert.Zero(t, repo.Creates())
		})
	}

	t.Run("password longer than bcrypt limit", func(t *testing.T) {
		t.Parallel()

		password := strings.Repeat("x", 73)
		form := api.RegisterForm{
			Email: "user@example.com", Password: password, Name: "User",
		}
		require.Error(t, form.Validate())

		repo := &regression.UserRepository{}
		response := regressionRequest(
			t,
			regression.NewHandler(regression.WithUsers(userservice.New(repo))),
			auth.Actor{},
			http.MethodPost, "/api/v2/auth/register",
			fmt.Sprintf(
				`{"email":"user@example.com","password":%q,"name":"User"}`,
				password,
			),
		)
		assert.Zero(t, repo.Creates())
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("duplicate email is conflict", func(t *testing.T) {
		t.Parallel()

		service := regression.NewUserServiceWithRegisterErr(
			fmt.Errorf("register: %w", user.ErrEmailTaken),
		)
		response := regressionRequest(
			t,
			regression.NewHandler(regression.WithUsers(service)),
			auth.Actor{}, http.MethodPost,
			"/api/v2/auth/register",
			`{"email":"user@example.com","password":"password","name":"User"}`,
		)
		logResponse(t, response)
		assert.Equal(t, http.StatusConflict, response.Code)
		assert.Equal(t, 1, service.Calls())
	})

	t.Run("duplicate at insert is conflict", func(t *testing.T) {
		t.Parallel()

		response := regressionRequest(
			t, regression.NewHandler(
				regression.WithUsers(
					regression.NewUserServiceWithRegisterErr(
						user.ErrEmailTaken,
					),
				),
			),
			auth.Actor{},
			http.MethodPost, "/api/v2/auth/register",
			`{"email":"user@example.com","password":"password","name":"User"}`,
		)
		logResponse(t, response)
		assert.Equal(t, http.StatusConflict, response.Code)
	})
}

type regressionSensorService struct {
	handler.SensorService

	renameErr   error
	getErr      error
	renameCalls int
	getCalls    int
}

func (s *regressionSensorService) Rename(
	_ context.Context,
	_ auth.Actor,
	_ sensor.ID,
	_ uuid.UUID,
	_ string,
) error {
	s.renameCalls++

	return s.renameErr
}

func (s *regressionSensorService) GetSensor(
	_ context.Context,
	_ auth.Actor,
	_ sensor.ID,
	_ uuid.UUID,
) (*sensor.Sensor, error) {
	s.getCalls++

	return nil, s.getErr
}

func TestRegression_EditMissingSensorReturnsNotFound(t *testing.T) {
	t.Parallel()

	missing := fmt.Errorf("sensor repository: %w", sensor.ErrNotFound)
	for _, tc := range []struct {
		name      string
		renameErr error
		getErr    error
		getCalls  int
	}{
		{name: "rename", renameErr: missing},
		{name: "lookup after rename", getErr: missing, getCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &regressionSensorService{
				renameErr: tc.renameErr,
				getErr:    tc.getErr,
			}
			path := fmt.Sprintf(
				"/api/v2/devices/%s/sensors/%s",
				uuid.New(),
				uuid.New(),
			)
			response := regressionRequest(
				t,
				regression.NewHandler(regression.WithSensors(service)),
				auth.Actor{UserID: uuid.New()},
				http.MethodPatch, path, `{"displayName":"new name"}`,
			)
			logResponse(t, response)
			assert.Equal(t, 1, service.renameCalls)
			assert.Equal(t, http.StatusNotFound, response.Code)
			assert.Equal(t, tc.getCalls, service.getCalls)
		})
	}
}

type regressionMetricsService struct {
	handler.MetricsService
}

func (regressionMetricsService) GetDeviceMetrics(
	_ context.Context,
	_ auth.Actor,
	_ uuid.UUID,
	_ metrics.Query,
) ([]metrics.Metric, error) {
	return nil, nil
}

func (regressionMetricsService) GetSensorMetrics(
	_ context.Context,
	_ auth.Actor,
	_ uuid.UUID,
	_ sensor.ID,
	_ metrics.Query,
) ([]metrics.Metric, error) {
	return nil, nil
}

func TestRegression_EmptyMetricsMatchContract(t *testing.T) {
	t.Parallel()

	handle := regression.NewHandler(
		regression.WithMetrics(&regressionMetricsService{}),
	)
	ctx := auth.WithActor(t.Context(), auth.Actor{UserID: uuid.New()})
	deviceID := uuid.New()

	for _, tc := range []struct {
		name string
		get  func() (*api.Metrics, error)
	}{
		{
			name: "device",
			get: func() (*api.Metrics, error) {
				return handle.GetDeviceMetricsByName(ctx, api.GetDeviceMetricsByNameParams{
					DeviceID: deviceID, Metric: "temperature",
				})
			},
		},
		{
			name: "sensor",
			get: func() (*api.Metrics, error) {
				return handle.GetSensorMetricsByName(ctx, api.GetSensorMetricsByNameParams{
					DeviceID: deviceID, SensorID: uuid.New(), Metric: "temperature",
				})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, err := tc.get()
			require.NoError(t, err)
			require.NotNil(t, response)
			assert.NotNil(t, response.Values)
			assert.Empty(t, response.Values)
			assert.NoError(t, response.Validate())
		})
	}
}

func TestRegression_ConnectIsAcceptedPendingApproval(t *testing.T) {
	t.Parallel()

	requests := regression.NewConnectRequestsRepo()
	publisher := regression.NewPublisher()
	service := deviceservice.New(
		regression.NewConnectRepo(),
		nil,
		publisher,
		requests,
	)
	actor := auth.Actor{UserID: uuid.New()}
	deviceID := uuid.New()
	response := regressionRequest(
		t,
		regression.NewHandler(regression.WithDevices(service)),
		actor, http.MethodPost,
		"/api/v2/devices",
		fmt.Sprintf(`{"deviceId":%q,"securityKey":"key"}`, deviceID),
	)
	require.True(t, publisher.Called())
	require.NotNil(t, publisher.Message())
	assert.Equal(t, deviceID, publisher.DeviceID())
	assert.Equal(
		t,
		requests.Request().ID,
		publisher.Message().Payload.RequestID,
	)
	assert.Equal(t, "key", publisher.Message().Payload.SecretKey)
	assert.Equal(t, actor.UserID, requests.Request().UserID)
	assert.Equal(t, deviceID, requests.Request().DeviceID)
	assert.False(t, requests.Request().Approved)
	// The current spec declares 201; this asserts the desired asynchronous
	// contract and requires a spec and generated API change to pass.
	assert.Equal(t, http.StatusAccepted, response.Code)
}

func logResponse(t *testing.T, r *httptest.ResponseRecorder) {
	t.Helper()
	t.Logf(
		"\nResponse:\n\tStatus: %d\n\tHeader: %v\n\tBody: %s",
		r.Code,
		r.Header(),
		r.Body.String(),
	)
}

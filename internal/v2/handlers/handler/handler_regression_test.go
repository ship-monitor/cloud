package handler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	devicerepo "github.com/ship-monitor/cloud/internal/v2/repositories/device"
	sensorrepo "github.com/ship-monitor/cloud/internal/v2/repositories/sensor"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
	userservice "github.com/ship-monitor/cloud/internal/v2/services/user"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type regressionSecurity struct {
	actor auth.Actor
}

func (s regressionSecurity) HandleBearerToken(
	ctx context.Context,
	_ api.OperationName,
	_ api.BearerToken,
) (context.Context, error) {
	return auth.WithActor(ctx, s.actor), nil
}

func regressionRequest(
	t *testing.T,
	h *Handler,
	actor auth.Actor,
	method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	h.logger = zap.NewNop()
	server, err := api.NewServer(h, regressionSecurity{actor: actor})
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

type regressionUserService struct {
	UserService

	registerErr error
	loginErr    error
	calls       int
}

func (s *regressionUserService) Register(
	_ context.Context,
	_, _, _ string,
) (user.User, error) {
	s.calls++

	return user.User{}, s.registerErr
}

func (s *regressionUserService) Login(
	_ context.Context,
	_, _ string,
) (user.User, error) {
	return user.User{}, s.loginErr
}

type regressionTokenService struct {
	TokenService

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
	DeviceService

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
		t, &Handler{devices: service}, auth.Actor{UserID: uuid.New()},
		http.MethodGet, "/api/v2/devices/"+uuid.New().String(), "",
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
		h      *Handler
		status int
	}{
		{
			name: "registration",
			path: "/api/v2/auth/register",
			body: `{"email":"user@example.com","password":"password","name":"User"}`,
			h: &Handler{
				users: &regressionUserService{registerErr: failure},
			},
			status: http.StatusInternalServerError,
		},
		{
			name:   "login",
			path:   "/api/v2/auth/login",
			body:   `{"email":"user@example.com","password":"password"}`,
			h:      &Handler{users: &regressionUserService{loginErr: failure}},
			status: http.StatusInternalServerError,
		},
		{
			name: "login token creation",
			path: "/api/v2/auth/login",
			body: `{"email":"user@example.com","password":"password"}`,
			h: &Handler{
				users:        &regressionUserService{},
				tokenService: &regressionTokenService{createErr: failure},
			},
			status: http.StatusInternalServerError,
		},
		{
			name: "refresh",
			path: "/api/v2/auth/refresh",
			body: `{"refreshToken":"token"}`,
			h: &Handler{
				tokenService: &regressionTokenService{refreshErr: failure},
			},
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

type regressionUserRepository struct {
	userservice.Repository

	checks  int
	creates int
}

func (r *regressionUserRepository) EmailTaken(
	_ context.Context,
	_ string,
) (bool, error) {
	r.checks++

	return false, nil
}

func (r *regressionUserRepository) Create(
	_ context.Context,
	_ user.User,
) error {
	r.creates++

	return &pq.Error{Code: "23505", Constraint: "users_email_key"}
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

		service := &regressionUserService{}
		response := regressionRequest(
			t,
			&Handler{users: service},
			auth.Actor{},
			http.MethodPost,
			"/api/v2/auth/register",
			`{"email":"not an email@@example.com","password":"password","name":"User"}`,
		)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Zero(t, service.calls)
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

			repo := &regressionUserRepository{}
			response := regressionRequest(
				t, &Handler{users: userservice.New(repo)}, auth.Actor{},
				http.MethodPost, "/api/v2/auth/register", tc.body,
			)
			t.Logf("Response: %d\n%s", response.Code, response.Body.String())
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Zero(t, repo.checks)
			assert.Zero(t, repo.creates)
		})
	}

	t.Run("password longer than bcrypt limit", func(t *testing.T) {
		t.Parallel()

		password := strings.Repeat("x", 73)
		form := api.RegisterForm{
			Email: "user@example.com", Password: password, Name: "User",
		}
		require.Error(t, form.Validate())

		repo := &regressionUserRepository{}
		response := regressionRequest(
			t, &Handler{users: userservice.New(repo)}, auth.Actor{},
			http.MethodPost, "/api/v2/auth/register",
			fmt.Sprintf(
				`{"email":"user@example.com","password":%q,"name":"User"}`,
				password,
			),
		)
		assert.Zero(t, repo.creates)
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("duplicate email is conflict", func(t *testing.T) {
		t.Parallel()

		service := &regressionUserService{
			registerErr: fmt.Errorf("register: %w", user.ErrEmailTaken),
		}
		response := regressionRequest(
			t, &Handler{users: service}, auth.Actor{}, http.MethodPost,
			"/api/v2/auth/register",
			`{"email":"user@example.com","password":"password","name":"User"}`,
		)
		t.Logf(
			"Response:\nCode: %d\nBody: %s",
			response.Code,
			response.Body.String(),
		)
		assert.Equal(t, http.StatusConflict, response.Code)
		assert.Equal(t, 1, service.calls)
	})

	t.Run("duplicate at insert is conflict", func(t *testing.T) {
		t.Parallel()

		response := regressionRequest(
			t, &Handler{users: &regressionUserService{
				registerErr: user.ErrEmailTaken,
			}}, auth.Actor{},
			http.MethodPost, "/api/v2/auth/register",
			`{"email":"user@example.com","password":"password","name":"User"}`,
		)
		t.Logf(
			"Response:\nCode: %d\nBody: %s",
			response.Code,
			response.Body.String(),
		)
		assert.Equal(t, http.StatusConflict, response.Code)
	})
}

type regressionSensorService struct {
	SensorService

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

	missing := fmt.Errorf("sensor repository: %w", sensorrepo.ErrNotFound)
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
				t, &Handler{sensors: service}, auth.Actor{UserID: uuid.New()},
				http.MethodPatch, path, `{"displayName":"new name"}`,
			)
			assert.Equal(t, 1, service.renameCalls)
			assert.Equal(t, http.StatusNotFound, response.Code)
			assert.Equal(t, tc.getCalls, service.getCalls)
		})
	}
}

type regressionMetricsService struct {
	MetricsService
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

	h := &Handler{metrics: regressionMetricsService{}}
	ctx := auth.WithActor(t.Context(), auth.Actor{UserID: uuid.New()})
	deviceID := uuid.New()

	for _, tc := range []struct {
		name string
		get  func() (*api.Metrics, error)
	}{
		{
			name: "device",
			get: func() (*api.Metrics, error) {
				return h.GetDeviceMetricsByName(ctx, api.GetDeviceMetricsByNameParams{
					DeviceID: deviceID, Metric: "temperature",
				})
			},
		},
		{
			name: "sensor",
			get: func() (*api.Metrics, error) {
				return h.GetSensorMetricsByName(ctx, api.GetSensorMetricsByNameParams{
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

type regressionConnectRepo struct {
	deviceservice.Repository
}

func (regressionConnectRepo) Exists(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

type regressionConnectRequests struct {
	deviceservice.ConnRequestRepository

	request devicedomain.ConnectRequest
}

func (r *regressionConnectRequests) Create(
	_ context.Context,
	request devicedomain.ConnectRequest,
) error {
	r.request = request

	return nil
}

type regressionPublisher struct {
	deviceservice.Publisher

	called   bool
	deviceID uuid.UUID
	message  *asyncapi.ConnectRequestMessage
}

func (p *regressionPublisher) PublishConnectRequest(
	_ context.Context,
	deviceID uuid.UUID,
	message *asyncapi.ConnectRequestMessage,
) error {
	p.called = true
	p.deviceID = deviceID
	p.message = message

	return nil
}

func TestRegression_ConnectIsAcceptedPendingApproval(t *testing.T) {
	t.Parallel()

	requests := &regressionConnectRequests{}
	publisher := &regressionPublisher{}
	service := deviceservice.New(
		regressionConnectRepo{},
		nil,
		publisher,
		requests,
	)
	actor := auth.Actor{UserID: uuid.New()}
	deviceID := uuid.New()
	response := regressionRequest(
		t, &Handler{devices: service}, actor, http.MethodPost,
		"/api/v2/devices",
		fmt.Sprintf(`{"deviceId":%q,"securityKey":"key"}`, deviceID),
	)
	require.True(t, publisher.called)
	require.NotNil(t, publisher.message)
	assert.Equal(t, deviceID, publisher.deviceID)
	assert.Equal(t, requests.request.ID, publisher.message.Payload.RequestID)
	assert.Equal(t, "key", publisher.message.Payload.SecretKey)
	assert.Equal(t, actor.UserID, requests.request.UserID)
	assert.Equal(t, deviceID, requests.request.DeviceID)
	assert.False(t, requests.request.Approved)
	// The current spec declares 201; this asserts the desired asynchronous
	// contract and requires a spec and generated API change to pass.
	assert.Equal(t, http.StatusAccepted, response.Code)
}

// The fake database reports that the device exists, but the owner-scoped
// update affects no rows, as it would for a device owned by another user.
type regressionSQLConnector struct {
	updateArgs []driver.NamedValue
}

func (c *regressionSQLConnector) Connect(context.Context) (driver.Conn, error) {
	return regressionSQLConn{connector: c}, nil
}

func (c *regressionSQLConnector) Driver() driver.Driver {
	return regressionSQLDriver{connector: c}
}

type regressionSQLDriver struct {
	connector *regressionSQLConnector
}

func (d regressionSQLDriver) Open(string) (driver.Conn, error) {
	return regressionSQLConn(d), nil
}

type regressionSQLConn struct {
	connector *regressionSQLConnector
}

func (regressionSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}

func (regressionSQLConn) Close() error { return nil }

func (regressionSQLConn) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected transaction")
}

func (regressionSQLConn) QueryContext(
	_ context.Context,
	_ string,
	_ []driver.NamedValue,
) (driver.Rows, error) {
	return &regressionExistsRows{}, nil
}

func (c regressionSQLConn) ExecContext(
	_ context.Context,
	_ string,
	args []driver.NamedValue,
) (driver.Result, error) {
	c.connector.updateArgs = append([]driver.NamedValue(nil), args...)

	return driver.RowsAffected(0), nil
}

type regressionExistsRows struct {
	done bool
}

func (*regressionExistsRows) Columns() []string { return []string{"exists"} }
func (*regressionExistsRows) Close() error      { return nil }

func (r *regressionExistsRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}

	r.done = true
	dest[0] = true

	return nil
}

func TestRegression_ForeignDeviceRenameDoesNotSucceed(t *testing.T) {
	t.Parallel()

	connector := &regressionSQLConnector{}
	db := sqlx.NewDb(sql.OpenDB(connector), "regression")

	t.Cleanup(func() { _ = db.Close() })

	service := deviceservice.New(devicerepo.New(db), nil, nil, nil)
	actor := auth.Actor{UserID: uuid.New()}
	deviceID := uuid.New()
	response := regressionRequest(
		t, &Handler{devices: service}, actor, http.MethodPatch,
		"/api/v2/devices/"+deviceID.String(), `{"name":"new name"}`,
	)
	require.Len(t, connector.updateArgs, 3)
	assert.Equal(t, actor.UserID.String(), connector.updateArgs[2].Value)
	assert.Equal(t, http.StatusNotFound, response.Code)
}

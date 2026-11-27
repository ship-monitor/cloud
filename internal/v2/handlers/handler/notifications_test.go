package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	domainnotification "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"github.com/ship-monitor/cloud/pkg/problem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type notificationPage = pagination.Page[
	domainnotification.Notification,
	domainnotification.Cursor,
]

type notificationServiceStub struct {
	NotificationService

	list func(
		context.Context,
		auth.Actor,
		uuid.UUID,
		pagination.Request[domainnotification.Cursor],
	) (notificationPage, error)
}

func (s *notificationServiceStub) ListByDevicePage(
	ctx context.Context,
	actor auth.Actor,
	deviceID uuid.UUID,
	req pagination.Request[domainnotification.Cursor],
) (notificationPage, error) {
	return s.list(ctx, actor, deviceID, req)
}

func TestGetDeviceNotifications_Pages(t *testing.T) {
	t.Parallel()

	actor := auth.Actor{UserID: uuid.New()}
	deviceID := uuid.New()
	sensorID := uuid.New()
	occurred := time.Date(2026, 11, 25, 12, 30, 1, 123456000, time.UTC)
	item := domainnotification.Notification{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		SensorID:   &sensorID,
		Type:       "engine_overheat",
		Severity:   "warning",
		OccurredAt: occurred,
		ReportedAt: occurred.Add(time.Second),
		ReceivedAt: occurred.Add(2 * time.Second),
		IsRead:     true,
		Data: map[string]any{
			"temperature": 95.5,
			"details":     map[string]any{"active": true},
			"empty":       nil,
		},
	}
	next := domainnotification.Cursor{OccurredAt: occurred, ID: item.ID}
	calls := 0
	service := &notificationServiceStub{
		list: func(
			ctx context.Context,
			gotActor auth.Actor,
			gotDeviceID uuid.UUID,
			req pagination.Request[domainnotification.Cursor],
		) (notificationPage, error) {
			calls++

			assert.Equal(t, actor, gotActor)
			assert.Equal(t, deviceID, gotDeviceID)
			assert.Equal(t, 1, req.Limit)
			assert.Equal(t, actor, actorFromTestContext(t, ctx))

			if calls == 1 {
				assert.Nil(t, req.After)

				return notificationPage{
					Items: []domainnotification.Notification{item},
					Next:  &next,
				}, nil
			}

			require.NotNil(t, req.After)
			assert.Equal(t, next.ID, req.After.ID)
			assert.True(t, next.OccurredAt.Equal(req.After.OccurredAt))

			return notificationPage{
				Items: []domainnotification.Notification{},
			}, nil
		},
	}
	h := &Handler{notifications: service}
	ctx := auth.WithActor(t.Context(), actor)
	params := api.GetDeviceNotificationsParams{
		DeviceID: deviceID,
		Limit:    api.NewOptInt64(1),
	}

	first, err := h.GetDeviceNotifications(ctx, params)
	require.NoError(t, err)
	require.Len(t, first.Notifications, 1)
	got := first.Notifications[0]
	assert.Equal(t, item.ID, got.ID)
	assert.Equal(t, deviceID, got.DeviceID)
	assert.Equal(t, api.NewOptUUID(sensorID), got.SensorID)
	assert.Equal(t, item.Type, got.Type)
	assert.Equal(t, api.NotificationSeverityWarning, got.Severity)
	assert.Equal(t, item.OccurredAt, got.OccurredAt)
	assert.Equal(t, item.ReportedAt, got.ReportedAt)
	assert.Equal(t, item.ReceivedAt, got.ReceivedAt)
	assert.True(t, got.IsRead)
	require.True(t, got.Data.Set)
	assert.JSONEq(t, `95.5`, string(got.Data.Value["temperature"]))
	assert.JSONEq(t, `{"active":true}`, string(got.Data.Value["details"]))
	assert.JSONEq(t, `null`, string(got.Data.Value["empty"]))
	assert.True(t, first.Pagination.HasMore)
	require.True(t, first.Pagination.NextCursor.Set)
	require.NotEmpty(t, first.Pagination.NextCursor.Value)
	cursorJSON, err := base64.RawURLEncoding.DecodeString(
		first.Pagination.NextCursor.Value,
	)
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(
		`{"occurredAt":%q,"id":%q}`,
		occurred.Format(time.RFC3339Nano), item.ID.String(),
	), string(cursorJSON))

	params.Cursor = first.Pagination.NextCursor
	second, err := h.GetDeviceNotifications(ctx, params)
	require.NoError(t, err)
	assert.NotNil(t, second.Notifications)
	assert.Empty(t, second.Notifications)
	assert.False(t, second.Pagination.HasMore)
	assert.False(t, second.Pagination.NextCursor.Set)
	assert.Equal(t, 2, calls)
}

func actorFromTestContext(t *testing.T, ctx context.Context) auth.Actor {
	t.Helper()

	actor, ok := auth.ActorFromContext(ctx)
	require.True(t, ok)

	return actor
}

func TestGetDeviceNotifications_EmptyAndOptionalFields(t *testing.T) {
	t.Parallel()

	service := &notificationServiceStub{
		list: func(
			_ context.Context,
			_ auth.Actor,
			_ uuid.UUID,
			req pagination.Request[domainnotification.Cursor],
		) (notificationPage, error) {
			assert.Zero(t, req.Limit)
			assert.Nil(t, req.After)

			return notificationPage{
				Items: []domainnotification.Notification{{}},
			}, nil
		},
	}
	h := &Handler{notifications: service}
	ctx := auth.WithActor(t.Context(), auth.Actor{UserID: uuid.New()})

	response, err := h.GetDeviceNotifications(
		ctx,
		api.GetDeviceNotificationsParams{DeviceID: uuid.New()},
	)
	require.NoError(t, err)
	require.Len(t, response.Notifications, 1)
	assert.False(t, response.Notifications[0].SensorID.Set)
	assert.False(t, response.Notifications[0].Data.Set)
	assert.False(t, response.Notifications[0].IsRead)
	assert.False(t, response.Pagination.HasMore)
	assert.False(t, response.Pagination.NextCursor.Set)
}

func TestGetDeviceNotifications_ValidLimits(t *testing.T) {
	t.Parallel()

	for _, limit := range []int64{1, pagination.MaxLimit} {
		t.Run(strconv.FormatInt(limit, 10), func(t *testing.T) {
			t.Parallel()

			called := false
			h := &Handler{notifications: &notificationServiceStub{
				list: func(
					_ context.Context,
					_ auth.Actor,
					_ uuid.UUID,
					req pagination.Request[domainnotification.Cursor],
				) (notificationPage, error) {
					called = true

					assert.Equal(t, int(limit), req.Limit)

					return notificationPage{}, nil
				},
			}}
			ctx := auth.WithActor(t.Context(), auth.Actor{UserID: uuid.New()})
			response, err := h.GetDeviceNotifications(ctx,
				api.GetDeviceNotificationsParams{
					DeviceID: uuid.New(),
					Limit:    api.NewOptInt64(limit),
				},
			)
			require.NoError(t, err)
			assert.True(t, called)
			assert.NotNil(t, response.Notifications)
			assert.Empty(t, response.Notifications)
		})
	}
}

func TestGetDeviceNotifications_InvalidPagination(t *testing.T) {
	t.Parallel()

	encode := base64.RawURLEncoding.EncodeToString

	cases := []struct {
		name   string
		cursor api.OptString
		limit  api.OptInt64
	}{
		{name: "zero limit", limit: api.NewOptInt64(0)},
		{name: "negative limit", limit: api.NewOptInt64(-1)},
		{name: "limit above maximum", limit: api.NewOptInt64(101)},
		{name: "empty cursor", cursor: api.NewOptString("")},
		{name: "invalid base64", cursor: api.NewOptString("%")},
		{name: "invalid json", cursor: api.NewOptString(encode([]byte("{")))},
		{name: "null json", cursor: api.NewOptString(encode([]byte("null")))},
		{
			name:   "missing fields",
			cursor: api.NewOptString(encode([]byte("{}"))),
		},
		{
			name: "invalid timestamp",
			cursor: api.NewOptString(encode([]byte(
				`{"occurredAt":"invalid",` +
					`"id":"e48b6e12-77bb-49a4-a826-d4f458411fdd"}`,
			))),
		},
		{
			name: "invalid uuid",
			cursor: api.NewOptString(encode([]byte(
				`{"occurredAt":"2026-11-25T12:00:00Z","id":"invalid"}`,
			))),
		},
		{
			name: "zero timestamp",
			cursor: api.NewOptString(encode([]byte(
				`{"occurredAt":"0001-01-01T00:00:00Z",` +
					`"id":"e48b6e12-77bb-49a4-a826-d4f458411fdd"}`,
			))),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			called := false
			h := &Handler{notifications: &notificationServiceStub{
				list: func(
					_ context.Context,
					_ auth.Actor,
					_ uuid.UUID,
					_ pagination.Request[domainnotification.Cursor],
				) (notificationPage, error) {
					called = true

					return notificationPage{}, nil
				},
			}}
			ctx := auth.WithActor(t.Context(), auth.Actor{UserID: uuid.New()})
			response, err := h.GetDeviceNotifications(ctx,
				api.GetDeviceNotificationsParams{
					DeviceID: uuid.New(),
					Cursor:   tc.cursor,
					Limit:    tc.limit,
				},
			)
			require.Error(t, err)

			var p *problem.Error
			require.ErrorAs(t, err, &p)
			assert.Equal(t, http.StatusBadRequest, p.Status)
			assert.Equal(t, errBadRequest.Type, p.Type)
			assert.Nil(t, response)
			assert.False(t, called)
		})
	}
}

func TestGetDeviceNotifications_Errors(t *testing.T) {
	t.Parallel()

	serviceErr := errors.New("repository unavailable")

	cases := []struct {
		name        string
		auth        bool
		serviceErr  error
		wantErr     error
		wantCalls   int
		invalidData bool
	}{
		{name: "anonymous", wantErr: errUnauthenticated},
		{
			name:       "device not found",
			auth:       true,
			serviceErr: fmt.Errorf("get device: %w", device.ErrNotFound),
			wantErr:    errDeviceNotFound,
			wantCalls:  1,
		},
		{
			name:       "service failure",
			auth:       true,
			serviceErr: serviceErr,
			wantErr:    serviceErr,
			wantCalls:  1,
		},
		{
			name:        "data cannot be encoded",
			auth:        true,
			wantCalls:   1,
			invalidData: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			h := &Handler{notifications: &notificationServiceStub{
				list: func(
					_ context.Context,
					_ auth.Actor,
					_ uuid.UUID,
					_ pagination.Request[domainnotification.Cursor],
				) (notificationPage, error) {
					calls++

					var page notificationPage
					if tc.invalidData {
						page.Items = []domainnotification.Notification{{
							Data: map[string]any{"bad": make(chan int)},
						}}
					}

					return page, tc.serviceErr
				},
			}}

			ctx := t.Context()
			if tc.auth {
				ctx = auth.WithActor(ctx, auth.Actor{UserID: uuid.New()})
			}

			response, err := h.GetDeviceNotifications(
				ctx,
				api.GetDeviceNotificationsParams{DeviceID: uuid.New()},
			)
			require.Error(t, err)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			}

			if tc.invalidData {
				assert.ErrorContains(t, err, "encode data field")
			}

			assert.Nil(t, response)
			assert.Equal(t, tc.wantCalls, calls)
		})
	}
}

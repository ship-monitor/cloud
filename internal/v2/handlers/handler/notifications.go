package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-faster/jx"
	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	domainnotification "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

type notificationCursor struct {
	OccurredAt time.Time `json:"occurredAt"`
	ID         uuid.UUID `json:"id"`
}

// GetDeviceNotifications implements [api.Handler].
func (h *Handler) GetDeviceNotifications(
	ctx context.Context,
	params api.GetDeviceNotificationsParams,
) (*api.GetDeviceNotificationsOK, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	var req pagination.Request[domainnotification.Cursor]

	if params.Limit.Set {
		if params.Limit.Value < 1 || params.Limit.Value > pagination.MaxLimit {
			return nil, errBadRequest.WithDetail("invalid notification limit")
		}

		req.Limit = int(params.Limit.Value)
	}

	if params.Cursor.Set {
		cursor, err := decodeNotificationCursor(params.Cursor.Value)
		if err != nil {
			return nil, errBadRequest.WithDetail("invalid notification cursor")
		}

		req.After = &cursor
	}

	page, err := h.notifications.ListByDevicePage(
		ctx,
		actor,
		params.DeviceID,
		req,
	)
	if errors.Is(err, device.ErrNotFound) {
		return nil, errDeviceNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("list device notifications: %w", err)
	}

	response := &api.GetDeviceNotificationsOK{
		Notifications: make([]api.Notification, 0, len(page.Items)),
		Pagination:    api.Pagination{HasMore: page.Next != nil},
	}
	for _, item := range page.Items {
		n, err := toAPINotification(item)
		if err != nil {
			return nil, fmt.Errorf("convert notification: %w", err)
		}

		response.Notifications = append(response.Notifications, n)
	}

	if page.Next != nil {
		cursor, err := encodeNotificationCursor(*page.Next)
		if err != nil {
			return nil, fmt.Errorf("encode notification cursor: %w", err)
		}

		response.Pagination.NextCursor = api.NewOptString(cursor)
	}

	return response, nil
}

func decodeNotificationCursor(
	encoded string,
) (domainnotification.Cursor, error) {
	var result domainnotification.Cursor

	data, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return result, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor notificationCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return result, fmt.Errorf("decode cursor json: %w", err)
	}

	if cursor.OccurredAt.IsZero() || cursor.ID == uuid.Nil {
		return result, errors.New("cursor is missing occurrence time or id")
	}

	return domainnotification.Cursor{
		OccurredAt: cursor.OccurredAt,
		ID:         cursor.ID,
	}, nil
}

func encodeNotificationCursor(
	cursor domainnotification.Cursor,
) (string, error) {
	data, err := json.Marshal(notificationCursor{
		OccurredAt: cursor.OccurredAt,
		ID:         cursor.ID,
	})
	if err != nil {
		return "", fmt.Errorf("encode cursor json: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func toAPINotification(
	n domainnotification.Notification,
) (api.Notification, error) {
	result := api.Notification{
		ID:         n.ID,
		DeviceID:   n.DeviceID,
		Type:       n.Type,
		Severity:   api.NotificationSeverity(n.Severity),
		OccurredAt: n.OccurredAt,
		ReportedAt: n.ReportedAt,
		ReceivedAt: n.ReceivedAt,
		IsRead:     n.IsRead,
	}
	if n.SensorID != nil {
		result.SensorID = api.NewOptUUID(*n.SensorID)
	}

	if n.Data != nil {
		data := make(api.NotificationData, len(n.Data))
		for key, value := range n.Data {
			raw, err := json.Marshal(value)
			if err != nil {
				return result, fmt.Errorf("encode data field %q: %w", key, err)
			}

			data[key] = jx.Raw(raw)
		}

		result.Data = api.NewOptNotificationData(data)
	}

	return result, nil
}

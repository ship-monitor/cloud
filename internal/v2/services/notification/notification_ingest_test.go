package notification_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	domainnotification "github.com/ship-monitor/cloud/internal/v2/domain/notification"
	"github.com/ship-monitor/cloud/internal/v2/services/notification"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"github.com/stretchr/testify/require"
)

type ingestRecorder struct {
	notification.Repo

	items []domainnotification.Notification
}

func (r *ingestRecorder) InsertNotification(
	_ context.Context,
	item domainnotification.Notification,
) error {
	r.items = append(r.items, item)

	return nil
}

//nolint:exhaustruct_v5,funlen
func TestIngestNotification_OptionalData(t *testing.T) {
	t.Parallel()

	for _, sourceType := range []asyncapi.NotificationSourceSchema{
		asyncapi.NotificationSourceSchemaDevice,
		asyncapi.NotificationSourceSchemaSensor,
	} {
		t.Run(string(sourceType), func(t *testing.T) {
			t.Parallel()

			cases := []struct {
				name string
				data string
				want map[string]any
			}{
				{name: "omitted", data: `{}`, want: map[string]any{}},

				{
					name: "empty object",
					data: `{"data":{}}`,
					want: map[string]any{},
				},
				{
					name: "properties",
					data: `{"data":{"detail":"hello"}}`,
					want: map[string]any{"detail": "hello"},
				},
			}

			for _, testCase := range cases {
				t.Run(testCase.name, func(t *testing.T) {
					t.Parallel()

					var msg asyncapi.NotificationSchema
					require.NoError(
						t,
						json.Unmarshal([]byte(testCase.data), &msg),
					)

					deviceID := uuid.New()

					sourceID := deviceID
					if sourceType == asyncapi.NotificationSourceSchemaSensor {
						sourceID = uuid.New()
					}

					msg.Source = asyncapi.SourcePropertyFromNotificationSchema{
						Id: sourceID, Type: sourceType,
					}
					msg.Type = "temperature"
					msg.Severity = asyncapi.SeveritySchemaWarning
					msg.OccurredAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
					msg.ReportedAt = msg.OccurredAt.Add(time.Second)
					repo := &ingestRecorder{}
					service := notification.New(repo, nil)

					require.NotPanics(t, func() {
						require.NoError(t, service.IngestNotification(
							t.Context(), deviceID, msg,
						))
					})
					require.Len(t, repo.items, 1)
					got := repo.items[0]
					require.Equal(t, deviceID, got.DeviceID)

					if sourceType == asyncapi.NotificationSourceSchemaSensor {
						require.NotNil(t, got.SensorID)
						require.Equal(t, sourceID, *got.SensorID)
					} else {
						require.Nil(t, got.SensorID)
					}

					require.Equal(t, msg.Type, got.Type)
					require.Equal(t, string(msg.Severity), got.Severity)
					require.Equal(t, msg.OccurredAt, got.OccurredAt)
					require.Equal(t, msg.ReportedAt, got.ReportedAt)
					require.NotEqual(t, uuid.Nil, got.ID)
					require.False(t, got.ReceivedAt.IsZero())
					require.Equal(t, testCase.want, got.Data)
				})
			}
		})
	}
}

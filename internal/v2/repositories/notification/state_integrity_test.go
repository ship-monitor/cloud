//go:build integration

//nolint:goconst
package notification_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/ship-monitor/cloud/internal/v2/repositories/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadStateRejectsMissingReferences(t *testing.T) {
	t.Parallel()

	for _, missing := range []string{"user", "notification"} {
		t.Run(missing, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			repo := notification.New(db)
			ctx := t.Context()
			userID := uuid.New()
			item := newNotification()

			if missing != "user" {
				_, err := db.ExecContext(ctx, `INSERT INTO users
					(id, name, email, password_hash, created_at)
					VALUES ($1, $2, $3, $4, $5)`,
					userID, "owner", userID.String(), "hash", time.Now())
				require.NoError(t, err)
			}

			if missing != "notification" {
				require.NoError(t, repo.InsertNotification(ctx, item))
			}

			err := repo.SetRead(ctx, userID, item.ID, true)
			require.Error(
				t,
				err,
				"read state must not reference a missing %s",
				missing,
			)

			var count int

			err = db.GetContext(ctx, &count, `SELECT count(*)
				FROM user_notification_states
				WHERE user_id = $1 AND notification_id = $2`, userID, item.ID)
			require.NoError(t, err)
			assert.Zero(t, count, "orphaned read state was stored")
		})
	}
}

func TestReadStateDoesNotOutliveItsReferences(t *testing.T) {
	t.Parallel()

	for _, parent := range []string{"user", "notification"} {
		t.Run(parent, func(t *testing.T) {
			t.Parallel()
			db := setupDB(t)
			repo := notification.New(db)
			ctx := t.Context()
			userID := uuid.New()
			item := newNotification()
			_, err := db.ExecContext(ctx, `INSERT INTO users
				(id, name, email, password_hash, created_at)
				VALUES ($1, $2, $3, $4, $5)`,
				userID, "owner", userID.String(), "hash", time.Now())
			require.NoError(t, err)
			require.NoError(t, repo.InsertNotification(ctx, item))
			require.NoError(t, repo.SetRead(ctx, userID, item.ID, true))

			if parent == "user" {
				_, err = db.ExecContext(
					ctx,
					`DELETE FROM users WHERE id = $1`,
					userID,
				)
			} else {
				err = repo.Delete(ctx, item.ID)
			}
			// A database may prevent deletion instead of cascading it; either
			// policy preserves referential integrity.
			if err != nil {
				var pgErr *pq.Error
				require.ErrorAs(t, err, &pgErr)
				require.Equal(t, "foreign_key_violation", pgErr.Code.Name())

				var count int

				query := `SELECT count(*) FROM users WHERE id = $1`
				id := userID

				if parent == "notification" {
					query = `SELECT count(*) FROM device_notifications WHERE id = $1`
					id = item.ID
				}

				require.NoError(t, db.GetContext(ctx, &count, query, id))
				require.Equal(
					t,
					1,
					count,
					"failed deletion must leave its parent intact",
				)

				return
			}

			var count int
			require.NoError(t, db.GetContext(ctx, &count, `SELECT count(*)
				FROM user_notification_states
				WHERE user_id = $1 AND notification_id = $2`, userID, item.ID))
			assert.Zero(t, count, "deletion left an orphaned read state")
		})
	}
}

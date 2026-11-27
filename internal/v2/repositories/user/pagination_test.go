//go:build integration

package user_test

import (
	"testing"

	"github.com/google/uuid"
	domainuser "github.com/ship-monitor/cloud/internal/v2/domain/user"
	"github.com/ship-monitor/cloud/internal/v2/repositories/user"
	"github.com/stretchr/testify/require"
)

func TestRepository_CursorPages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	users := user.New(db)
	firstUser := domainuser.User{
		ID: uuid.UUID{3}, Email: "a@example.org", Name: "", PasswordHash: "",
	}
	lastUser := domainuser.User{
		ID: uuid.UUID{1}, Email: "b@example.org", Name: "", PasswordHash: "",
	}
	require.NoError(t, users.Create(t.Context(), lastUser))
	require.NoError(t, users.Create(t.Context(), firstUser))
	checkPages(
		t,
		users.ListPage,
		[]domainuser.User{firstUser, lastUser},
		func(u domainuser.User) string { return u.Email },
	)
}

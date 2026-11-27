package token_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
	"github.com/ship-monitor/cloud/internal/v2/services/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_TokenPurposeSeparation(t *testing.T) {
	t.Parallel()

	service, err := token.New(&token.Config{
		SecretKey: "test-secret", TokenTTL: time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	})
	require.NoError(t, err)

	userID := uuid.New()
	access, refresh, err := service.CreateTokens(
		t.Context(), &user.User{ID: userID},
	)
	require.NoError(t, err)

	accessClaims, err := service.ParseToken(t.Context(), access)
	require.NoError(t, err)
	assert.Equal(t, userID, accessClaims.UserID)
	refreshClaims, err := service.ParseRefreshToken(t.Context(), refresh)
	require.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.UserID)

	_, err = service.ParseToken(t.Context(), refresh)
	assert.Error(t, err, "refresh tokens must not authorize access")
	_, err = service.ParseRefreshToken(t.Context(), access)
	assert.Error(t, err, "access tokens must not be parsed as refresh tokens")
	_, _, err = service.RefreshTokens(t.Context(), access)
	assert.Error(t, err, "access tokens must not mint new token pairs")

	newAccess, newRefresh, err := service.RefreshTokens(t.Context(), refresh)
	require.NoError(t, err)
	newAccessClaims, err := service.ParseToken(t.Context(), newAccess)
	require.NoError(t, err)
	assert.Equal(t, userID, newAccessClaims.UserID)
	newRefreshClaims, err := service.ParseRefreshToken(t.Context(), newRefresh)
	require.NoError(t, err)
	assert.Equal(t, userID, newRefreshClaims.UserID)
}

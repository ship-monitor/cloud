package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

func newToken(s domain.Session) string {
	return s.ID.String()
}

// Returns a hex-encoded SHA-256 hash of the session token, used as a unique
// identifier (key in store).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

type SessionStore interface {
	GetSession(ctx context.Context, hash string) (*domain.Session, error)
	GetVersion(ctx context.Context, userID uuid.UUID) (int64, error)
	DeleteSession(ctx context.Context, userID uuid.UUID, hash string) error
	ListSessions(
		ctx context.Context,
		userID uuid.UUID,
	) ([]domain.Session, error)
	IncrementVersion(ctx context.Context, userID uuid.UUID) error
	CreateSession(
		ctx context.Context,
		hash string,
		s domain.Session,
		ttl time.Duration,
	) error
	TouchSession(ctx context.Context, hash string, expiresAt time.Time) error
	RevokeByID(ctx context.Context, userID, sessionID uuid.UUID) error
	RevokeAllExcept(
		ctx context.Context,
		userID uuid.UUID,
		keepSessionID uuid.UUID,
	) error
}

type AuthConfig struct {
	SessionTTL           time.Duration
	SessionTouchInterval time.Duration
}

const (
	DefaultTTL           = time.Hour * 24
	DefaultTouchInterval = time.Minute * 15
)

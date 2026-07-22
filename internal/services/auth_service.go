package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"time"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionRevoked  = errors.New("session revoked")
	ErrSessionNotFound = errors.New("session not found")
)

type CreatedSession struct {
	Session domain.Session
	Token   string
}

func newToken(s domain.Session) string {
	return s.ID.String()
}

// Returns a hash of the session token, that is used as a unique identifier (key
// in store).
func hashToken(token string) string {
	return string(sha256.New().Sum([]byte(token)))
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
	RevokeByID(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error
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

type Sessions struct {
	sessions SessionStore
	config   *AuthConfig
	logger   *log.Logger
}

const (
	DefaultTTL           = time.Hour * 24
	DefaultTouchInterval = time.Minute * 15
)

func NewSessions(
	sessions SessionStore,
	config *viper.Viper,
	logger *log.Logger,
) *Sessions {
	config.SetDefault("auth.session.ttl", DefaultTTL)
	config.SetDefault("auth.session.touch-interval", DefaultTouchInterval)

	return &Sessions{
		sessions: sessions,
		config: &AuthConfig{
			SessionTTL: viper.GetDuration("auth.session.ttl"),
			SessionTouchInterval: config.GetDuration(
				"auth.session.touch-interval",
			),
		},
		logger: logger,
	}
}

type ClientInfo struct {
	UserAgent string
	ClientIP  string
}

func (s *Sessions) Create(
	ctx context.Context,
	userID uuid.UUID, info ClientInfo,
) (*CreatedSession, error) {
	version, err := s.sessions.GetVersion(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get session version: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(s.config.SessionTTL)

	session := domain.Session{
		ID:         uuid.New(),
		UserID:     userID,
		Version:    version,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  expiresAt,
		UserAgent:  info.UserAgent,
		IPAddress:  info.ClientIP,
	}

	token := newToken(session)

	if err := s.sessions.CreateSession(
		ctx,
		hashToken(token),
		session,
		s.config.SessionTTL,
	); err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}

	return &CreatedSession{
		Session: session,
		Token:   token,
	}, nil
}

func (s *Sessions) Authenticate(
	ctx context.Context,
	token string,
) (*domain.Principal, error) {
	hash := hashToken(token)

	storedSession, err := s.sessions.GetSession(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, fmt.Errorf("%w: %w", ErrUnauthenticated, err)
		}

		return nil, fmt.Errorf("get session: %w", err)
	}

	now := time.Now()
	if !now.Before(storedSession.ExpiresAt) {
		return nil, fmt.Errorf(
			"%w: expires at %s, now %s",
			ErrSessionExpired,
			storedSession.ExpiresAt.String(),
			now.String(),
		)
	}

	currentVersion, err := s.sessions.GetVersion(ctx, storedSession.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user session version: %w", err)
	}

	if storedSession.Version != currentVersion {
		return nil, fmt.Errorf(
			"%w: session version %d, current version %d",
			ErrSessionRevoked,
			storedSession.Version,
			currentVersion,
		)
	}

	if now.Sub(storedSession.LastSeenAt) >= s.config.SessionTouchInterval {
		// It's better not to block success auth on touch error
		if err := s.sessions.TouchSession(
			ctx,
			hash,
			now.Add(s.config.SessionTTL),
		); err != nil {
			log.Warn("Failed touch session", "error", err)
		}
	}

	return &domain.Principal{
		UserID:    storedSession.UserID,
		SessionID: storedSession.ID,
	}, nil
}

func (s *Sessions) Logout(
	ctx context.Context,
	token string,
) error {
	tokenHash := hashToken(token)

	storedSession, err := s.sessions.GetSession(ctx, tokenHash)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return fmt.Errorf("get session: %w", err)
	}

	if storedSession != nil {
		if err := s.sessions.DeleteSession(
			ctx,
			storedSession.UserID,
			tokenHash,
		); err != nil {
			return fmt.Errorf("delete session: %w", err)
		}
	}

	return nil
}

func (s *Sessions) LogoutAll(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	userID uuid.UUID,
) error {
	if err := s.sessions.IncrementVersion(ctx, userID); err != nil {
		return fmt.Errorf("increment session version: %w", err)
	}

	return nil
}

// List returns all active sessions of the user.
func (s *Sessions) List(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Session, error) {
	sessions, err := s.sessions.ListSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	return sessions, nil
}

// RevokeByID terminates a single session by its ID.
func (s *Sessions) RevokeByID(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
) error {
	if err := s.sessions.RevokeByID(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

// RevokeOthers terminates every session of the user except the current one.
func (s *Sessions) RevokeOthers(
	ctx context.Context,
	userID uuid.UUID,
	currentSessionID uuid.UUID,
) error {
	if err := s.sessions.RevokeAllExcept(ctx, userID, currentSessionID); err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}

	return nil
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	DefaultSessionTTL = 24 * time.Hour
	sessionIDBytes    = 32
)

var (
	ErrInvalidSessionID = errors.New("invalid session id")
	ErrSessionNotFound  = errors.New("session not found")
)

type StoredSession struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type SessionStore interface {
	Create(ctx context.Context, userID uuid.UUID, email string) (string, error)
	Get(ctx context.Context, sessionID string) (*StoredSession, error)
	Refresh(ctx context.Context, sessionID string) (*StoredSession, error)
	Delete(ctx context.Context, sessionID string) error
	TTL() time.Duration
}

type RedisSessionStore struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewRedisSessionStore(
	redisClient *redis.Client,
	ttl time.Duration,
) *RedisSessionStore {
	return &RedisSessionStore{
		redis: redisClient,
		ttl:   DefaultSessionTTL,
	}
}

func (s *RedisSessionStore) TTL() time.Duration {
	return s.ttl
}

func (s *RedisSessionStore) Create(
	ctx context.Context,
	userID uuid.UUID,
	email string,
) (string, error) {
	sessionID, err := newSessionID()
	if err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}

	now := time.Now().UTC()
	stored := StoredSession{
		UserID:    userID,
		Email:     email,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}

	if err := s.set(ctx, sessionID, &stored); err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *RedisSessionStore) Get(
	ctx context.Context,
	sessionID string,
) (*StoredSession, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	value, err := s.redis.Get(ctx, sessionKey(sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrSessionNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get session from redis: %w", err)
	}

	var stored StoredSession
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	if time.Now().UTC().After(stored.ExpiresAt) {
		if err := s.Delete(ctx, sessionID); err != nil {
			return nil, fmt.Errorf("delete expired session: %w", err)
		}

		return nil, ErrSessionNotFound
	}

	return &stored, nil
}

func (s *RedisSessionStore) Refresh(
	ctx context.Context,
	sessionID string,
) (*StoredSession, error) {
	stored, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	stored.ExpiresAt = time.Now().UTC().Add(s.ttl)
	if err := s.set(ctx, sessionID, stored); err != nil {
		return nil, err
	}

	return stored, nil
}

func (s *RedisSessionStore) Delete(
	ctx context.Context,
	sessionID string,
) error {
	if sessionID == "" {
		return nil
	}

	if err := s.redis.Del(ctx, sessionKey(sessionID)).Err(); err != nil {
		return fmt.Errorf("delete session from redis: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) set(
	ctx context.Context,
	sessionID string,
	stored *StoredSession,
) error {
	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := s.redis.Set(ctx, sessionKey(sessionID), data, s.ttl).
		Err(); err != nil {
		return fmt.Errorf("set session in redis: %w", err)
	}

	return nil
}

func newSessionID() (string, error) {
	buf := make([]byte, sessionIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func sessionKey(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))

	return "session:" + hex.EncodeToString(sum[:])
}

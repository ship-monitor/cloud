package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
)

const AllOccurencies = 0

var _ services.SessionStore = (*RedisSessionStore)(nil)

type RedisSessionStore struct {
	rdb *redis.Client
}

func NewRedisSessionStore(rdb *redis.Client) *RedisSessionStore {
	return &RedisSessionStore{rdb: rdb}
}

// IncrementVersion implements [services.SessionStore].
func (r *RedisSessionStore) IncrementVersion(
	ctx context.Context,
	userID uuid.UUID,
) error {
	if err := r.rdb.Incr(ctx, userSessionVersionKey(userID)).Err(); err != nil {
		return fmt.Errorf("increment version: %w", err)
	}

	return nil
}

// CreateSession implements [services.SessionStore].
func (r *RedisSessionStore) CreateSession(
	ctx context.Context,
	hash string,
	s domain.Session,
	ttl time.Duration,
) error {
	data, err := s.JSON()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	if err := r.rdb.Set(
		ctx,
		sessionKey(hash),
		data,
		ttl,
	).Err(); err != nil {
		return fmt.Errorf("set key in redis: %w", err)
	}

	if err := r.rdb.LPush(ctx, userSessionsListKey(s.UserID), hash).Err(); err != nil {
		return fmt.Errorf("push to session list: %w", err)
	}

	return nil
}

// DeleteSession implements [services.SessionStore].
func (r *RedisSessionStore) DeleteSession(
	ctx context.Context,
	userID uuid.UUID,
	hash string,
) error {
	if err := r.rdb.LRem(ctx, userSessionsListKey(userID), AllOccurencies, hash).
		Err(); err != nil {
		return fmt.Errorf("delete from session list: %w", err)
	}

	if err := r.rdb.Del(ctx, sessionKey(hash)).Err(); err != nil {
		return fmt.Errorf("delete session value: %w", err)
	}

	return nil
}

// GetSession implements [services.SessionStore].
func (r *RedisSessionStore) GetSession(
	ctx context.Context,
	hash string,
) (*domain.Session, error) {
	data, err := r.rdb.Get(ctx, sessionKey(hash)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	s := domain.Session{}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &s, nil
}

// GetVersion implements [services.SessionStore].
func (r *RedisSessionStore) GetVersion(
	ctx context.Context,
	userID uuid.UUID,
) (int64, error) {
	version, err := r.rdb.Get(ctx, userSessionVersionKey(userID)).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}

		return 0, fmt.Errorf("redis get int64: %w", err)
	}

	return version, nil
}

// ListSessions implements [services.SessionStore].
func (r *RedisSessionStore) ListSessions(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Session, error) {
	hashes, err := r.rdb.LRange(ctx, userSessionsListKey(userID), 0, -1).
		Result()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sessions := make([]domain.Session, 0, len(hashes))
	for _, hash := range hashes {
		session, err := r.GetSession(ctx, hash)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}

			return nil, fmt.Errorf("get session: %w", err)
		}

		sessions = append(sessions, *session)
	}

	return sessions, nil
}

// TouchSession implements [services.SessionStore].
func (r *RedisSessionStore) TouchSession(
	ctx context.Context,
	hash string,
	expiresAt time.Time,
) error {
	_, err := r.GetSession(ctx, hash)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	_, err = r.rdb.Expire(ctx, sessionKey(hash), time.Until(expiresAt)).
		Result()
	if err != nil {
		return fmt.Errorf("change expire: %w", err)
	}

	return nil
}

// RevokeByID implements [services.SessionStore].
func (r *RedisSessionStore) RevokeByID(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
) error {
	hash, err := r.findHashBySessionID(ctx, userID, sessionID)
	if err != nil {
		return err
	}

	if err := r.DeleteSession(ctx, userID, hash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

// RevokeAllExcept implements [services.SessionStore].
func (r *RedisSessionStore) RevokeAllExcept(
	ctx context.Context,
	userID uuid.UUID,
	keepSessionID uuid.UUID,
) error {
	hashes, err := r.rdb.LRange(ctx, userSessionsListKey(userID), 0, -1).
		Result()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	for _, hash := range hashes {
		session, err := r.GetSession(ctx, hash)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}

			return fmt.Errorf("get session: %w", err)
		}

		if session.ID == keepSessionID {
			continue
		}

		if err := r.DeleteSession(ctx, userID, hash); err != nil {
			return fmt.Errorf("delete session: %w", err)
		}
	}

	return nil
}

// findHashBySessionID resolves the store key (token hash) of a session by its ID.
func (r *RedisSessionStore) findHashBySessionID(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
) (string, error) {
	hashes, err := r.rdb.LRange(ctx, userSessionsListKey(userID), 0, -1).
		Result()
	if err != nil {
		return "", fmt.Errorf("list sessions: %w", err)
	}

	for _, hash := range hashes {
		session, err := r.GetSession(ctx, hash)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}

			return "", fmt.Errorf("get session: %w", err)
		}

		if session.ID == sessionID {
			return hash, nil
		}
	}

	return "", services.ErrSessionNotFound
}

func sessionKey(hash string) string {
	return "session:" + hash
}

func userSessionVersionKey(userID uuid.UUID) string {
	return "user:" + userID.String() + ":sessionVersion"
}

func userSessionsListKey(userID uuid.UUID) string {
	return "user:" + userID.String() + ":sessions"
}

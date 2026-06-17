package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
)

type IEmail interface {
	WriteEmail(senderName, senderEmail string) []byte
	ReceiverEmail() string
}

type EmailSender interface {
	SendEmail(ctx context.Context, email IEmail) error
}

type AuthService struct {
	logger *log.Logger
	redis  *redis.Client
	email  EmailSender
}

func NewAuthService(
	logger *log.Logger,
	redisClient *redis.Client,
	email EmailSender,
) *AuthService {
	return &AuthService{
		logger: logger.WithPrefix("Auth service"),
		redis:  redisClient,
	}
}

func (a *AuthService) GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := data.GetUser(userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	return user.Email, nil
}

func (a *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*data.User, error) {
	user, err := data.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

type EmailConfirmationData struct {
	Email  string
	UserID uuid.UUID
}

func genEmailConfirmationToken() string {
	return fmt.Sprintf("email-confirmation-%s", uuid.New().String())
}

// StartEmailConfirmation implements [handlers.AuthService].
// Returns [ErrAlreadyConfirmed] if it is so.
func (a *AuthService) StartEmailConfirmation(ctx context.Context, userID uuid.UUID) error {
	user, err := a.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if user.EmailVerified {
		return ErrAlreadyConfirmed
	}

	token := genEmailConfirmationToken()

	if err := a.redis.Set(ctx, token, &EmailConfirmationData{
		Email:  user.Email,
		UserID: userID,
	}, EmailConfirmationTTL).Err(); err != nil {
		return fmt.Errorf("set key in redis: %w", err)
	}
	e, err := NewHTMLEmail(user.Email, user.Name, "Email confirmation", fmt.Sprintf(`
		<h1>Confirm email</h1>
		<p>Click link below to go to email confirmation page</p>
		<a href="%s/confirm-email?token=%s">Confirm</a>
		<p>Link is valid till %s</p>
	`, FrontendBaseURL, token, time.Now().Add(EmailConfirmationTTL).Format(time.DateTime)))
	if err != nil {
		return fmt.Errorf("create email: %w", err)
	}

	if err := a.email.SendEmail(ctx, e); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

var ErrAlreadyConfirmed = fmt.Errorf("user email already confirmed")

// ConfirmEmail try to confirm email by token. Returns [ErrAlreadyConfirmed] if it is so.
func (a *AuthService) ConfirmEmail(ctx context.Context, token string) error {
	value, err := a.redis.Get(ctx, token).Result()
	if err != nil {
		return fmt.Errorf("get confirmation data: %w", err)
	}

	defer func() {
		if err := a.redis.Del(ctx, token).Err(); err != nil {
			log.Warn("failed delete redis key", "key", token, "error", err)
		}
	}()

	var confData EmailConfirmationData

	if err := json.Unmarshal([]byte(value), &confData); err != nil {
		return fmt.Errorf("bad confirmation data: %w", err)
	}

	user, err := a.GetUser(ctx, confData.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if confData.Email != user.Email {
		return fmt.Errorf("wrong email to confirm")
	}

	if user.EmailVerified {
		return ErrAlreadyConfirmed
	}

	if err := data.SetEmailVerified(ctx, user.ID); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

const (
	FrontendBaseURL      = "http://157.22.206.199:3000/"
	EmailConfirmationTTL = time.Minute * 30
)

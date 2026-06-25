package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/email"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/auth/data"
)

type AuthService struct {
	logger *log.Logger
	redis  *redis.Client
	email  domain.EmailSender
}

func NewAuthService(
	logger *log.Logger,
	redisClient *redis.Client,
	email domain.EmailSender,
) *AuthService {
	return &AuthService{
		logger: logger.WithPrefix("Auth service"),
		redis:  redisClient,
		email:  email,
	}
}

func (a *AuthService) GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := data.GetUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	return user.Email, nil
}

func (a *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := data.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

type EmailConfirmationData struct {
	Email  string    `json:"email"`
	UserID uuid.UUID `json:"userId"`
}

func genEmailConfirmationToken() string {
	return "email-confirmation-" + uuid.New().String()
}

// StartEmailConfirmation implements [handlers.AuthService].
// Returns [ErrEmailAlreadyConfirmed] if it is so.
func (a *AuthService) StartEmailConfirmation(ctx context.Context, userID uuid.UUID) error {
	user, err := a.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if user.EmailVerified {
		return ErrEmailAlreadyConfirmed
	}

	token := genEmailConfirmationToken()

	data, err := json.Marshal(&EmailConfirmationData{
		Email:  user.Email,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("failed marshal data: %w", err)
	}

	if err := a.redis.Set(ctx, token, data, EmailConfirmationTTL).Err(); err != nil {
		return fmt.Errorf("set key in redis: %w", err)
	}

	e := email.Email{
		To:      user.Email,
		Subject: "Email confirmation",
		Lang:    "en",
		Body: fmt.Appendf(nil, `
			<h1>Confirm email</h1>
			<p>Click link below to go to email confirmation page</p>
			<a href="%s/confirm-email?token=%s">Confirm</a>
			<p>Link is valid till %s</p>`,
			FrontendBaseURL, token, time.Now().Add(EmailConfirmationTTL).Format(time.DateTime)),
	}

	if err := a.email.SendEmail(ctx, e); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

var ErrEmailAlreadyConfirmed = errors.New("user email already confirmed")

// ConfirmEmail try to confirm email by token. Returns [ErrEmailAlreadyConfirmed] if it is so.
func (a *AuthService) ConfirmEmail(ctx context.Context, userID uuid.UUID, token string) error {
	value, err := a.redis.Get(ctx, token).Result()
	if err != nil {
		return fmt.Errorf("get confirmation data: %w", err)
	}

	var confData EmailConfirmationData

	if err := json.Unmarshal([]byte(value), &confData); err != nil {
		return fmt.Errorf("bad confirmation data: %w", err)
	}

	if confData.UserID != userID {
		return errors.New("wrong user to confirm email")
	}

	user, err := a.GetUser(ctx, confData.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if confData.Email != user.Email {
		return errors.New("wrong email to confirm")
	}

	if user.EmailVerified {
		return ErrEmailAlreadyConfirmed
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

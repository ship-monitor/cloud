package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"charm.land/log/v2"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/email"
)

const (
	FrontendBaseURL      = "http://157.22.206.199:3000/"
	EmailConfirmationTTL = time.Minute * 30
)

type UsersRepo interface {
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
	SetEmailVerified(ctx context.Context, userID uuid.UUID, verified bool) error
	SetPassword(ctx context.Context, userID uuid.UUID, hashed []byte) error
	EmailTaken(ctx context.Context, email string) (bool, error)
	SetEmail(ctx context.Context, userID uuid.UUID, email string) error
}

type AuthService struct {
	logger *log.Logger
	redis  *redis.Client
	email  domain.EmailSender
	users  UsersRepo
}

func NewAuthService(
	logger *log.Logger,
	redisClient *redis.Client,
	email domain.EmailSender,
	users UsersRepo,
) *AuthService {
	return &AuthService{
		logger: logger.WithPrefix("Auth service"),
		redis:  redisClient,
		email:  email,
		users:  users,
	}
}

var (
	ErrEmailTaken          = errors.New("email already taken")
	ErrInvalidRegisterData = errors.New("invalid register data")
)

// Register registers a new user with the given name, email, and password.
// It returns an error
// if the email is already taken ([ErrEmailTaken]),
// if the data invalid ([ErrInvalidRegisterData])
// or if there is an error while creating the user.
func (a *AuthService) Register(
	ctx context.Context,
	data domain.RegisterUserData,
) error {
	if err := validator.New().Struct(data); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	taken, err := a.users.EmailTaken(ctx, data.Email)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRegisterData, err)
	}

	if taken {
		return ErrEmailTaken
	}

	userID := uuid.New()

	user := &domain.User{
		ID:            userID,
		Name:          data.Name,
		Email:         data.Email,
		PasswordHash:  HashPassword(data.Password),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Blocked:       false,
		EmailVerified: false,
	}
	if err := a.users.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := a.StartEmailConfirmation(ctx, userID); err != nil {
		log.Error("Faield start email confirmation")
	}

	return nil
}

func (a *AuthService) GetUserEmail(
	ctx context.Context,
	userID uuid.UUID,
) (string, error) {
	user, err := a.users.GetUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	return user.Email, nil
}

func (a *AuthService) GetUser(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.User, error) {
	user, err := a.users.GetUser(ctx, userID)
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
func (a *AuthService) StartEmailConfirmation(
	ctx context.Context,
	userID uuid.UUID,
) error {
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

	if err := a.redis.Set(ctx, token, data, EmailConfirmationTTL).
		Err(); err != nil {
		return fmt.Errorf("set key in redis: %w", err)
	}

	query := url.Values{
		viper.GetString("frontend.email-confirmation-query"): {token},
	}

	confirmationURL := url.URL{
		Scheme:   "http",
		Host:     viper.GetString("frontend.host"),
		Path:     viper.GetString("frontend.email-confiramtion-path"),
		RawQuery: query.Encode(),
	}
	e := email.Email{
		To:      user.Email,
		Subject: "Email confirmation",
		Lang:    "en",
		Body: fmt.Appendf(
			nil,
			`
			<h1>Confirm email</h1>
			<p>Click link below to go to email confirmation page</p>
			<a href="%s"><em>Confirm</em></a>
			<p>Link is valid till %s</p>`,
			confirmationURL.String(),
			time.Now().Add(EmailConfirmationTTL).Format(time.DateTime),
		),
	}

	if err := a.email.SendEmail(ctx, e); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

var ErrEmailAlreadyConfirmed = errors.New("user email already confirmed")

// ConfirmEmail try to confirm email by token. Returns
// [ErrEmailAlreadyConfirmed] if it is so.
func (a *AuthService) ConfirmEmail(
	ctx context.Context,
	userID uuid.UUID,
	token string,
) error {
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

	if err := a.users.SetEmailVerified(ctx, user.ID, true); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// ErrBadCredentials is returned when the user provides invalid credentials
// (email or password).
var ErrBadCredentials = errors.New("wrong password")

func (a *AuthService) Login(
	ctx context.Context,
	email, password string,
) (*domain.User, error) {
	user, err := a.users.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if !CheckPassword(user.PasswordHash, password) {
		a.logger.Error(
			"Password don't match hash",
			"user",
			user.ID,
			"email",
			user.Email,
		)

		return nil, ErrBadCredentials
	}

	return user, nil
}

var ErrWrongOldPassword = errors.New("wrong old password")

func (a *AuthService) ChangePassword(
	ctx context.Context,
	userID uuid.UUID,
	oldPassword, newPassword string,
) error {
	user, err := a.users.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user by id: %w", err)
	}

	if !CheckPassword(user.PasswordHash, oldPassword) {
		return ErrWrongOldPassword
	}

	hashed := HashPassword(newPassword)
	if err := a.users.SetPassword(ctx, userID, hashed); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}

func (a *AuthService) ChangeEmail(
	ctx context.Context,
	userID uuid.UUID,
	newEmail string,
) error {
	_, err := a.users.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user by id: %w", err)
	}

	err = a.users.SetEmail(ctx, userID, newEmail)
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}

	err = a.StartEmailConfirmation(ctx, userID)
	if err != nil {
		a.logger.Error("Failed to start email confirmation", "error", err)
	}

	return nil
}

func HashPassword(password string) []byte {
	hashed, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		panic(err)
	}

	return hashed
}

func CheckPassword(hashedPassword []byte, password string) bool {
	return bcrypt.CompareHashAndPassword(
		hashedPassword,
		[]byte(password),
	) == nil
}

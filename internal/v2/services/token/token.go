package token

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ship-monitor/cloud/internal/v2/domain/auth"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
)

const (
	TokenTTL        = time.Minute
	RefreshTokenTTL = time.Hour * 2
)

type Config struct {
	SecretKey string `validate:"required"`
}

type Service struct {
	conf *Config
}

func New(
	conf *Config,
) (*Service, error) {
	err := validator.New().Struct(conf)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Service{
		conf: conf,
	}, nil
}

func (s *Service) ParseToken(
	ctx context.Context,
	token string,
) (*auth.Claims, error) {
	var claims auth.Claims

	_, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(t *jwt.Token) (any, error) {
			return []byte(s.conf.SecretKey), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return &claims, nil
}

func (s *Service) CreateTokens(
	ctx context.Context,
	user *user.User,
) (string, string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		auth.NewClaims(user.ID, TokenTTL),
	)

	tokenStr, err := token.SignedString([]byte(s.conf.SecretKey))
	if err != nil {
		return "", "", fmt.Errorf("get signed token: %w", err)
	}

	refreshToken := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		auth.NewRefreshClaims(user.ID, RefreshTokenTTL),
	)

	refreshTokenStr, err := refreshToken.SignedString([]byte(s.conf.SecretKey))
	if err != nil {
		return "", "", fmt.Errorf("get signed token: %w", err)
	}

	return tokenStr, refreshTokenStr, nil
}

func (s *Service) RefreshTokens(
	ctx context.Context,
	refreshToken string,
) (string, string, error) {
	return "", "", nil
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"charm.land/log/v2"
	"github.com/joho/godotenv"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

func main() {
	err := exec()
	if err != nil {
		fmt.Println("[ERROR]", err.Error())

		os.Exit(1)
	}
}

func exec() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		os.Kill,
	)
	defer cancel()

	_ = godotenv.Load()
	port := os.Getenv("PORT")
	log.Info("", "port", port)

	secSource := NewSecuritySource()

	client, err := api.NewClient(
		"http://"+"localhost:"+port,
		secSource,
	)
	if err != nil {
		return fmt.Errorf("new api client: %w", err)
	}

	email := "some@mail.org"
	password := "Pass123@@@ass"
	name := "Katy"

	err = client.RegisterUser(ctx, &api.RegisterUserReq{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		return fmt.Errorf("register user %q: %w", name, err)
	}

	return nil
}

type SecuritySource struct {
	token string
}

var _ api.SecuritySource = (*SecuritySource)(nil)

func NewSecuritySource() *SecuritySource {
	return &SecuritySource{
		token: "",
	}
}

// BearerToken implements [api.SecuritySource].
func (s *SecuritySource) BearerToken(
	ctx context.Context,
	operationName api.OperationName,
) (api.BearerToken, error) {
	return api.BearerToken{
		Token: s.token, Roles: []string{},
	}, nil
}

package di

import (
	"errors"
	"fmt"

	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/ship-monitor/cloud/internal/config"
	"go.uber.org/fx"
)

var (
	ErrSpiceDBURLMissing     = errors.New("spicedb-url is not set")
	ErrPresharedTokenMissing = errors.New("preshared-token is not set")
)

func NewSpiceDB(
	lc fx.Lifecycle,
	config *config.Config,
) (*authzed.Client, error) {
	spicedbURL := config.SpiceDB.URL
	if spicedbURL == "" {
		return nil, ErrSpiceDBURLMissing
	}

	presharedToken := config.SpiceDB.GRPCPresharedKey
	if presharedToken == "" {
		return nil, ErrPresharedTokenMissing
	}

	systemCerts, err := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
	if err != nil {
		return nil, fmt.Errorf("unable to load system CA certificates: %w", err)
	}

	client, err := authzed.NewClient(
		spicedbURL,
		systemCerts,
		grpcutil.WithBearerToken(presharedToken),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize client: %w", err)
	}

	return client, nil
}

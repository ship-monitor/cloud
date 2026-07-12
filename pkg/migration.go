package pkg

import (
	"context"
	"fmt"

	"charm.land/log/v2"
	"go.uber.org/fx"
)

type MigrationRepo interface {
	Migrate(ctx context.Context) error
}

type MigrationExecutor struct {
	logger       log.Logger
	repositories []MigrationRepo
}

func ProvideMigrations() fx.Option {
	return fx.Options(
		fx.Provide(
			func(lc fx.Lifecycle, logger *log.Logger, migrations ...MigrationRepo) *MigrationExecutor {
				exec := &MigrationExecutor{
					repositories: migrations,
					logger:       *logger,
				}
				lc.Append(fx.StartHook(exec.Start))

				return exec
			},
		),
		fx.Invoke(func(*MigrationExecutor) {}),
	)
}

func (m *MigrationExecutor) Start(ctx context.Context) error {
	m.logger.Info("Executing migrations")

	for _, repo := range m.repositories {
		if err := repo.Migrate(ctx); err != nil {
			return fmt.Errorf("failed migrate repository: %w", err)
		}
	}

	m.logger.Info("Migration successful")

	return nil
}

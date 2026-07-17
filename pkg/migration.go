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
			NewExecutor,
			fx.Annotate(NewAgregator, fx.ParamTags(`group:"repos"`)),
		),
		fx.Invoke(func(*MigrationExecutor) {}),
	)
}

func AsMigrationRepo(f any) any {
	return fx.Annotate(
		f, fx.As(new(MigrationRepo)),
		fx.ResultTags(`group:"repos"`),
	)
}

func NewExecutor(
	lc fx.Lifecycle,
	logger *log.Logger,
	repos *Agregator,
) *MigrationExecutor {
	exec := &MigrationExecutor{
		repositories: repos.repos,
		logger:       *logger,
	}
	lc.Append(fx.StartHook(exec.Start))

	return exec
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

type Agregator struct {
	repos []MigrationRepo
}

func NewAgregator(repos []MigrationRepo) *Agregator {
	return &Agregator{repos: repos}
}

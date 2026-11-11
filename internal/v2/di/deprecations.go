package di

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	"github.com/ship-monitor/cloud/internal/v2/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func deprecations() fx.Option {
	return fx.Module("Deprecations",

		fx.Provide(newCheckDB),
		fx.Invoke(func(*dbChecker) {}),
	)
}

type dbChecker struct{}

//nolint:mnd
func newCheckDB(
	lc fx.Lifecycle,
	conf *config.Config,
	logger *zap.Logger,
) *dbChecker {
	style := lipgloss.NewStyle().
		Width(40).
		Margin(1).
		Border(lipgloss.ASCIIBorder(), true).
		BorderForeground(lipgloss.Yellow).
		Padding(1).
		PaddingTop(3).
		PaddingBottom(3).
		Align(lipgloss.Center)

	lc.Append(
		fx.StartHook(func() {
			driver := os.Getenv("DB_DRIVER")
			if driver == "sqlite" {
				logger.Warn(
					"Database driver configuration is deprecated",
					zap.String("env", "DB_DRIVER"), zap.String("value", driver),
				)
			}

			isInteractive := term.IsTerminal(os.Stdin.Fd()) &&
				term.IsTerminal(os.Stdout.Fd())
			if !isInteractive {
				return
			}

			if driver == "sqlite" ||
				strings.Contains(conf.DB.DSN, "sqlite") {
				text := lipgloss.JoinVertical(lipgloss.Center,
					style.Render(
						lipgloss.JoinVertical(
							lipgloss.Left,
							lipgloss.NewStyle().
								Bold(true).
								MarginBottom(2).
								Render(
									"DEPRECATION WARNING!",
								),
							lipgloss.NewStyle().
								AlignHorizontal(lipgloss.Left).
								Render(
									"Sqlite database not supported."+
										"You probably specified DB_DRIVER as sqlite or not change DB_DSN to postgres DSN",
								),
						),
					),
				)
				fmt.Fprintln(
					os.Stderr,
					text,
				)
			}
		},
		))

	return nil
}

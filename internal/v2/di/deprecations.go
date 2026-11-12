package di

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/ship-monitor/cloud/internal/v2/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func deprecations(conf *config.Config) fx.Option {
	if !conf.IsDevelopment() {
		return fx.Options()
	}

	return fx.Module("Deprecations",

		fx.Provide(newCheckDB),
		fx.Invoke(func(*dbChecker) {}),
		fx.Provide(newCheckCorsOrigins),
		fx.Invoke(func(*corsChecker) {}),
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

			if driver != "sqlite" &&
				!strings.Contains(conf.DB.DSN, "sqlite") {
				return
			}

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
		},
		))

	return nil
}

type corsChecker struct{}

//nolint:mnd
func newCheckCorsOrigins(
	lc fx.Lifecycle,
	conf *config.Config,
	logger *zap.Logger,
) *corsChecker {
	style := lipgloss.NewStyle().
		Margin(1).
		Width(70).
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("5")).
		Padding(1).
		PaddingTop(3).
		PaddingBottom(3).
		Align(lipgloss.Center)

	lc.Append(
		fx.StartHook(func() {
			if len(conf.CorsOrigins) > 0 {
				return
			}

			text := lipgloss.JoinVertical(lipgloss.Center,
				style.Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						lipgloss.NewStyle().
							Bold(true).
							MarginBottom(2).
							BorderBottom(true).
							BorderStyle(lipgloss.MarkdownBorder()).
							BorderBottomForeground(lipgloss.Color("5")).
							Render(
								"CORS warning",
							),
						lipgloss.NewStyle().
							AlignHorizontal(lipgloss.Left).
							Render(
								"You are in development mode but doesn't specify any CORS origins. "+
									"That can lead to a problems with requsts at the frontend. "+
									"To fix this add the following line to your "+
									lipgloss.NewStyle().
										Background(lipgloss.BrightBlack).
										Render(".env")+
									" file:",
								"",
								lipgloss.NewStyle().
									MarginTop(2).
									Padding(1).PaddingLeft(2).PaddingRight(2).
									Background(lipgloss.White).
									Foreground(lipgloss.Black).
									Render("CORS_ORIGINS='http://localhost:5137,http://127.0.0.1:5173'"),
							),
					),
				),
			)
			fmt.Fprintln(
				os.Stderr,
				text,
			)
		},
		))

	return nil
}

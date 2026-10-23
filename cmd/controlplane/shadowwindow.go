// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"log/slog"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func logShadowWindows(
	ctx context.Context, log *slog.Logger, counting application.CountShadowWindows, repository domain.GovernedRepository,
) {
	windows, err := counting.Execute(ctx, repository)
	if err != nil {
		log.Error(shadowWindowRefused, slog.String(repositoryKey, repository.String()), slog.String(reasonKey, err.Error()))
		return
	}
	for _, counted := range windows {
		log.Info(shadowWindowCounted,
			slog.String(repositoryKey, repository.String()), slog.String(modeKey, string(counted.Mode)),
			slog.Int(runsKey, counted.Window.Runs), slog.Int(gapsKey, counted.Window.Gaps), slog.Int(escapesKey, counted.Window.Escapes))
	}
}

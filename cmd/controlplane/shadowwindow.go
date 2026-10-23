// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"log/slog"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func logShadowWindows(ctx context.Context, log *slog.Logger, counting application.CountShadowWindows, repository domain.GovernedRepository) {
	windows, err := counting.Execute(ctx, repository)
	if err != nil {
		log.Error("shadow window not counted", slog.String("reason", err.Error()))
		return
	}
	for _, counted := range windows {
		log.Info("shadow window counted", slog.String("repository", repository.String()), slog.String("mode", string(counted.Mode)),
			slog.Int("runs", counted.Window.Runs), slog.Int("gaps", counted.Window.Gaps), slog.Int("escapes", counted.Window.Escapes))
	}
}

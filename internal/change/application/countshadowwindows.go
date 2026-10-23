// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type CountShadowWindows struct {
	Runs    domain.ShadowRuns
	Escapes domain.EscapeHistory
}

func (useCase CountShadowWindows) Execute(ctx context.Context, repository domain.GovernedRepository) ([]domain.ModeWindow, error) {
	runs, err := useCase.Runs.ShadowedRuns(ctx)
	if err != nil {
		return nil, err
	}
	escapes, err := useCase.Escapes.RecordedEscapes(ctx)
	if err != nil {
		return nil, err
	}
	return domain.ShadowWindowsFor(repository, runs, escapes), nil
}

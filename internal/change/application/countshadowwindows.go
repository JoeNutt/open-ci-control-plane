// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type CountShadowWindows struct {
	Runs    domain.ShadowRuns
	Escapes domain.EscapeHistory
}

func (useCase CountShadowWindows) Execute(ctx context.Context, repository domain.GovernedRepository) ([]domain.ModeWindow, error) {
	runs, err := useCase.Runs.ShadowedRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("count the shadow windows of %s: read the runs: %w", repository, err)
	}
	escapes, err := useCase.Escapes.RecordedEscapes(ctx)
	if err != nil {
		return nil, fmt.Errorf("count the shadow windows of %s: read the escapes: %w", repository, err)
	}
	return domain.ShadowWindowsFor(repository, runs, escapes), nil
}

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
	evidence, err := shadowEvidenceIn(ctx, useCase.Runs, useCase.Escapes)
	if err != nil {
		return nil, fmt.Errorf("count the shadow windows of %s: %w", repository, err)
	}
	return domain.ShadowWindowsFor(repository, evidence.runs, evidence.escapes), nil
}

// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type EvaluateChange struct {
	Build BuildDocument
	Gate  domain.Gate
}

func (useCase EvaluateChange) Execute(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]domain.FamilyDecision, error) {
	document, err := useCase.Build.Execute(ctx, locator)
	if err != nil {
		return nil, err
	}
	return useCase.Gate.DecisionsFor(ctx, document), nil
}

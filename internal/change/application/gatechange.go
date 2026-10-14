// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type GateChange struct {
	Build     BuildDocument
	Gate      domain.Gate
	Publisher domain.CheckRunPublisher
}

func (useCase GateChange) Execute(
	ctx context.Context, locator domain.PullRequestLocator, id domain.DecisionID,
) error {
	document, err := useCase.Build.Execute(ctx, locator)
	if err != nil {
		return err
	}
	head, err := domain.ParseCommitSHA(document.PR.HeadSHA)
	if err != nil {
		return err
	}
	for _, decided := range useCase.Gate.DecisionsFor(ctx, document) {
		if err := useCase.Publisher.PublishCheckRun(
			ctx, locator, head, domain.CheckRunFor(decided, id)); err != nil {
			return err
		}
	}
	return nil
}

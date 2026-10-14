// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"errors"
	"fmt"

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
	var failures []error
	posting := context.WithoutCancel(ctx)
	for _, decided := range useCase.Gate.DecisionsFor(ctx, document) {
		run := domain.CheckRunFor(decided, id)
		if err := useCase.Publisher.PublishCheckRun(posting, locator, head, run); err != nil {
			failures = append(failures, fmt.Errorf("post %s: %w", run.Name, err))
		}
	}
	return errors.Join(failures...)
}

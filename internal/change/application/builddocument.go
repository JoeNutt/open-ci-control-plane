// SPDX-License-Identifier: GPL-3.0-or-later

// Package application assembles a change document from what a forge reports.
package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type BuildDocument struct {
	Source domain.ChangeSource
	Limits domain.Limits
	Rules  domain.ClassificationRules
}

func (useCase BuildDocument) Execute(
	ctx context.Context, locator domain.PullRequestLocator,
) (domain.Document, error) {
	pullRequest, err := useCase.Source.PullRequestAt(ctx, locator)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read pull request %s: %w", locator, err)
	}
	files, err := useCase.Source.FilesChanged(ctx, locator)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read changed files of %s: %w", locator, err)
	}
	bounded, err := domain.ParseBoundedFiles(files, useCase.Limits)
	if err != nil {
		return domain.Document{}, fmt.Errorf("bound %s: %w", locator, err)
	}
	return domain.ParseDocument(pullRequest, bounded, useCase.Rules), nil
}

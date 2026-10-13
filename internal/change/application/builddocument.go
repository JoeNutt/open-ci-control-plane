// SPDX-License-Identifier: GPL-3.0-or-later

// Package application assembles a change document from what a forge reports.
package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func BuildDocument(
	ctx context.Context,
	source domain.ChangeSource,
	locator domain.PullRequestLocator,
	limits domain.Limits,
) (domain.Document, error) {
	pullRequest, err := source.PullRequestAt(ctx, locator)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read pull request %s: %w", locator, err)
	}
	files, err := source.FilesChanged(ctx, locator)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read changed files of %s: %w", locator, err)
	}
	bounded, err := domain.ParseBoundedFiles(files, limits)
	if err != nil {
		return domain.Document{}, fmt.Errorf("bound %s: %w", locator, err)
	}
	return domain.ParseDocument(pullRequest, bounded), nil
}

// SPDX-License-Identifier: GPL-3.0-or-later

// Package github answers for a forge over the GitHub REST API.
package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeSource struct {
	BaseURL          string
	Client           *http.Client
	MaxResponseBytes int
}

func (source ChangeSource) PullRequestAt(
	ctx context.Context, locator domain.PullRequestLocator,
) (domain.PullRequest, error) {
	stated, err := jsonAt[pullRequest](ctx, source, pullRequestEndpoint(locator))
	if err != nil {
		return domain.PullRequest{}, fmt.Errorf("read the pull request %s: %w", locator, err)
	}
	return pullRequestOf(stated, locator)
}

func (source ChangeSource) FilesChanged(
	ctx context.Context, locator domain.PullRequestLocator,
) ([]domain.File, error) {
	listed, err := source.changedFilesAcrossPages(ctx, locator)
	if err != nil {
		return nil, fmt.Errorf("list the changed files of %s: %w", locator, err)
	}
	files, err := filesOf(listed)
	if err != nil {
		return nil, fmt.Errorf("read the changed files of %s: %w", locator, err)
	}
	return files, nil
}

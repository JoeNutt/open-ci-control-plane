// SPDX-License-Identifier: GPL-3.0-or-later

// Package github answers for a forge over the GitHub REST API.
package github

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeSource struct {
	settings Settings
}

func (source ChangeSource) String() string {
	return "the forge at " + source.settings.BaseURL
}

func (source ChangeSource) PullRequestAt(
	ctx context.Context, locator domain.PullRequestLocator,
) (domain.PullRequest, error) {
	stated, err := jsonAt[pullRequest](ctx, source, pullRequestEndpoint(locator))
	if err != nil {
		return domain.PullRequest{}, fmt.Errorf("read the pull request %s: %w", locator, err)
	}
	reported, err := pullRequestOf(stated, locator)
	if err != nil {
		return domain.PullRequest{}, err
	}
	reported.Approvals, err = source.approvalsOf(ctx, locator, reported.Author.Login)
	if err != nil {
		return domain.PullRequest{}, fmt.Errorf("read the reviews of %s: %w", locator, err)
	}
	return reported, nil
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

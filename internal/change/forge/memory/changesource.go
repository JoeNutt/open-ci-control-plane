// SPDX-License-Identifier: GPL-3.0-or-later

// Package memory answers for a forge from changes a test recorded.
package memory

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type Change struct {
	PullRequest domain.PullRequest
	Files       []domain.File
}

type ChangeSource struct {
	Changes map[domain.PullRequestLocator]Change
}

func (source ChangeSource) PullRequestAt(
	_ context.Context, locator domain.PullRequestLocator,
) (domain.PullRequest, error) {
	change, recorded := source.Changes[locator]
	if !recorded {
		return domain.PullRequest{}, fmt.Errorf("%w: %s", domain.ErrNoSuchPullRequest, locator)
	}
	return change.PullRequest, nil
}

func (source ChangeSource) FilesChanged(
	_ context.Context, locator domain.PullRequestLocator,
) ([]domain.File, error) {
	change, recorded := source.Changes[locator]
	if !recorded {
		return nil, fmt.Errorf("%w: %s", domain.ErrNoSuchPullRequest, locator)
	}
	return change.Files, nil
}

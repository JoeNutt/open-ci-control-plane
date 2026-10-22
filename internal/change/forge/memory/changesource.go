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
	Tips        domain.BranchTips
	Files       []domain.File
}

type ChangeSource struct {
	Changes map[domain.PullRequestLocator]Change
}

func (source ChangeSource) PullRequestAt(
	_ context.Context, locator domain.PullRequestLocator,
) (domain.ReportedPullRequest, error) {
	change, err := source.changeAt(locator)
	if err != nil {
		return domain.ReportedPullRequest{}, err
	}
	return domain.ReportedPullRequest{PullRequest: change.PullRequest, Tips: change.Tips}, nil
}

func (source ChangeSource) FilesChanged(
	_ context.Context, locator domain.PullRequestLocator,
) ([]domain.File, error) {
	change, err := source.changeAt(locator)
	if err != nil {
		return nil, err
	}
	return change.Files, nil
}

func (source ChangeSource) changeAt(locator domain.PullRequestLocator) (Change, error) {
	change, recorded := source.Changes[locator]
	if !recorded {
		return Change{}, fmt.Errorf("%w: %s", domain.ErrNoSuchPullRequest, locator)
	}
	return change, nil
}

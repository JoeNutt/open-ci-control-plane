// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func (source ChangeSource) MergeBaseOf(
	ctx context.Context, locator domain.PullRequestLocator, tips domain.BranchTips,
) (domain.CommitSHA, error) {
	compared, err := jsonAt[comparison](ctx, source.forge, compareEndpoint(locator, tips))
	if err != nil {
		return domain.CommitSHA{}, fmt.Errorf("compare the tips of %s: %w", locator, err)
	}
	mergeBase, err := domain.ParseCommitSHA(compared.MergeBaseCommit.SHA)
	if err != nil {
		return domain.CommitSHA{}, fmt.Errorf("the merge base of %s: %w", locator, err)
	}
	return mergeBase, nil
}

const threeDotRange = "..."

func compareEndpoint(locator domain.PullRequestLocator, tips domain.BranchTips) string {
	return fmt.Sprintf("/repos/%s/%s/compare/%s",
		url.PathEscape(locator.Owner()), url.PathEscape(locator.Name()),
		url.PathEscape(tips.Base.String()+threeDotRange+tips.Head.String()))
}

type comparison struct {
	MergeBaseCommit struct {
		SHA string `json:"sha"`
	} `json:"merge_base_commit"`
}

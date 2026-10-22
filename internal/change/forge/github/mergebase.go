// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrTipsNotComparable = errors.New("the forge cannot compare the base and head tips")

func (source ChangeSource) MergeBaseOf(
	ctx context.Context, locator domain.PullRequestLocator, tips domain.BranchTips,
) (domain.CommitSHA, error) {
	compared, err := exchanged[comparison](ctx, source.forge, exchange{
		method:    http.MethodGet,
		endpoint:  compareEndpoint(locator, tips),
		succeeded: http.StatusOK,
		notFound:  ErrTipsNotComparable,
	})
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

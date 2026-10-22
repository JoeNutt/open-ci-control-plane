// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func (source ChangeSource) mergeBaseOf(
	ctx context.Context, locator domain.PullRequestLocator, base, head domain.CommitSHA,
) (domain.CommitSHA, error) {
	compared, err := jsonAt[comparison](ctx, source.forge, compareEndpoint(locator, base, head))
	if err != nil {
		return domain.CommitSHA{}, err
	}
	return domain.ParseCommitSHA(compared.MergeBaseCommit.SHA)
}

const threeDotRange = "..."

func compareEndpoint(locator domain.PullRequestLocator, base, head domain.CommitSHA) string {
	return fmt.Sprintf("/repos/%s/%s/compare/%s",
		url.PathEscape(locator.Owner()), url.PathEscape(locator.Name()),
		url.PathEscape(base.String()+threeDotRange+head.String()))
}

type comparison struct {
	MergeBaseCommit struct {
		SHA string `json:"sha"`
	} `json:"merge_base_commit"`
}

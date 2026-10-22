// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeBases struct {
	MergeBases map[domain.BranchTips]domain.CommitSHA
}

func (bases ChangeBases) MergeBaseOf(
	_ context.Context, locator domain.PullRequestLocator, tips domain.BranchTips,
) (domain.CommitSHA, error) {
	mergeBase, recorded := bases.MergeBases[tips]
	if !recorded {
		return domain.CommitSHA{}, fmt.Errorf("%w: %s has no merge base recorded for %s...%s",
			domain.ErrNoMergeBase, locator, tips.Base(), tips.Head())
	}
	return mergeBase, nil
}

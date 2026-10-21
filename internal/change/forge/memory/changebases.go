// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeBases struct {
	Bases map[domain.PullRequestLocator]domain.CommitSHA
}

func (bases ChangeBases) BaseCommitOf(
	_ context.Context, locator domain.PullRequestLocator,
) (domain.CommitSHA, error) {
	base, recorded := bases.Bases[locator]
	if !recorded {
		return domain.CommitSHA{}, fmt.Errorf("%w: %s", domain.ErrNoSuchPullRequest, locator)
	}
	return base, nil
}

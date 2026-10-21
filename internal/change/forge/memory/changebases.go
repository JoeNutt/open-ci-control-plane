// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ChangeBases struct {
	Bases map[domain.PullRequestLocator]domain.CommitSHA
}

func (bases ChangeBases) BaseCommitOf(
	_ context.Context, locator domain.PullRequestLocator,
) (domain.CommitSHA, error) {
	return bases.Bases[locator], nil
}

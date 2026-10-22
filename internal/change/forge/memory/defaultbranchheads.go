// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type DefaultBranchHeads struct {
	Heads map[domain.PolicyRepository]domain.CommitSHA
}

func (heads DefaultBranchHeads) DefaultBranchHeadOf(
	_ context.Context, repository domain.PolicyRepository,
) (domain.CommitSHA, error) {
	head, recorded := heads.Heads[repository]
	if !recorded {
		return domain.CommitSHA{}, fmt.Errorf("%w: %s", domain.ErrNoSuchRepository, repository)
	}
	return head, nil
}

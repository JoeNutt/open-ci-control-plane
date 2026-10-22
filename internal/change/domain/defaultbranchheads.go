// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
)

var ErrNoSuchBranch = errors.New("the forge reports no such branch in the repository")

type DefaultBranchHeads interface {
	DefaultBranchHeadOf(ctx context.Context, repository PolicyRepository) (CommitSHA, error)
}

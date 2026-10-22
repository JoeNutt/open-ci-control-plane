// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type DefaultBranchHeads interface {
	DefaultBranchHeadOf(ctx context.Context, repository PolicyRepository) (CommitSHA, error)
}

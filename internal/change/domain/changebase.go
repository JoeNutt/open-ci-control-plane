// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type ChangeBase interface {
	MergeBaseOf(ctx context.Context, locator PullRequestLocator, tips BranchTips) (CommitSHA, error)
}

type BranchTips struct {
	Base CommitSHA
	Head CommitSHA
}

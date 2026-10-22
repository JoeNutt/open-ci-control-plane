// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
)

type ChangeBase interface {
	MergeBaseOf(ctx context.Context, locator PullRequestLocator, tips BranchTips) (CommitSHA, error)
}

var ErrNoMergeBase = errors.New("the forge holds no merge base for the branch tips")

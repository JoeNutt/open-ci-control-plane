// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type ChangeBase interface {
	BaseCommitOf(ctx context.Context, locator PullRequestLocator) (CommitSHA, error)
}

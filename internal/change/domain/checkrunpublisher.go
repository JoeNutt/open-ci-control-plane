// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type CheckRunPublisher interface {
	PublishCheckRun(
		ctx context.Context, locator PullRequestLocator, head CommitSHA, run CheckRun,
	) error
}

// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type ChangeSource interface {
	PullRequestAt(ctx context.Context, locator PullRequestLocator) (ReportedPullRequest, error)
	FilesChanged(ctx context.Context, locator PullRequestLocator) ([]File, error)
}

// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type CheckRun struct {
	Name       string
	Conclusion Conclusion
	Title      string
	Summary    string
	Text       string
}

type CheckRunPublisher interface {
	PublishCheckRun(
		ctx context.Context, locator PullRequestLocator, headSHA string, run CheckRun,
	) error
}

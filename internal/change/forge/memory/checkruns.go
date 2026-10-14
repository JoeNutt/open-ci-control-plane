// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"maps"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type commit struct {
	locator domain.PullRequestLocator
	headSHA string
}

type CheckRuns struct {
	published map[commit]map[string]domain.CheckRun
}

func (runs *CheckRuns) PublishCheckRun(
	_ context.Context, locator domain.PullRequestLocator, headSHA string, run domain.CheckRun,
) error {
	if runs.published == nil {
		runs.published = map[commit]map[string]domain.CheckRun{}
	}
	key := commit{locator: locator, headSHA: headSHA}
	if runs.published[key] == nil {
		runs.published[key] = map[string]domain.CheckRun{}
	}
	runs.published[key][run.Name] = run
	return nil
}

func (runs *CheckRuns) PublishedOn(
	locator domain.PullRequestLocator, headSHA string,
) map[string]domain.CheckRun {
	return maps.Clone(runs.published[commit{locator: locator, headSHA: headSHA}])
}

// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.CheckRunPublisher = CheckRuns{}

type CheckRuns struct {
	forge connection
}

func ParseCheckRuns(settings Settings) (CheckRuns, error) {
	forge, err := parseConnection(settings)
	if err != nil {
		return CheckRuns{}, err
	}
	return CheckRuns{forge: forge}, nil
}

func (runs CheckRuns) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, headSHA string, run domain.CheckRun,
) error {
	head, err := parseCommitSHA(headSHA)
	if err != nil {
		return err
	}
	listed, err := jsonAt[checkRunListing](ctx, runs.forge,
		checkRunsOnCommitEndpoint(locator, head, run.Name))
	if err != nil {
		return err
	}
	if len(listed.CheckRuns) == 0 {
		return runs.send(ctx, creationOf(locator, head, run))
	}
	return runs.send(ctx, updateOf(locator, listed.CheckRuns[0].ID, run))
}

func (runs CheckRuns) send(ctx context.Context, publication exchange) error {
	_, err := exchanged[struct{}](ctx, runs.forge, publication)
	return err
}

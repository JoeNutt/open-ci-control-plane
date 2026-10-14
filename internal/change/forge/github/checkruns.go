// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"
	"net/http"

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
	listed, err := jsonAt[checkRunListing](ctx, runs.forge, checkRunsOnCommitEndpoint(locator, head, run.Name))
	if err != nil {
		return err
	}
	if len(listed.CheckRuns) > 0 {
		_, err = exchanged[struct{}](ctx, runs.forge, exchange{
			method:    http.MethodPatch,
			endpoint:  fmt.Sprintf("/repos/%s/%s/check-runs/%d", locator.Owner(), locator.Name(), listed.CheckRuns[0].ID),
			sent:      verdictOf(run),
			succeeded: http.StatusOK,
		})
		return err
	}
	_, err = exchanged[struct{}](ctx, runs.forge, exchange{
		method:    http.MethodPost,
		endpoint:  checkRunsEndpoint(locator),
		sent:      newCheckRun{Name: run.Name, HeadSHA: string(head), checkRunVerdict: verdictOf(run)},
		succeeded: http.StatusCreated,
	})
	return err
}

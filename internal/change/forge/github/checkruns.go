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
	pending, err := parsePublication(locator, headSHA, run)
	if err != nil {
		return err
	}
	listed, err := jsonAt[checkRunListing](ctx, runs.forge, pending.listingEndpoint())
	if err != nil {
		return err
	}
	chosen, err := pending.exchangeGiven(listed)
	if err != nil {
		return err
	}
	_, err = exchanged[struct{}](ctx, runs.forge, chosen)
	return err
}

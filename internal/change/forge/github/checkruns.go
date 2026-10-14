// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.CheckRunPublisher = CheckRuns{}

var ErrAmbiguousCheckRun = errors.New("the commit carries more than one check run of the name")

var ErrInconsistentListing = errors.New("the forge's check run count disagrees with the runs it listed")

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
	pending := publication{locator: locator, head: head, run: run}
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

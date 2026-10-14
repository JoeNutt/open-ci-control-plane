// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"fmt"

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

func (runs CheckRuns) String() string {
	return "the check runs at " + runs.forge.settings.BaseURL
}

func (runs CheckRuns) GoString() string {
	return fmt.Sprintf("github.CheckRuns{settings: %#v}", runs.forge.settings)
}

func (runs CheckRuns) Format(state fmt.State, verb rune) {
	formatRedacted(state, verb, runs)
}

func (runs CheckRuns) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, headSHA string, run domain.CheckRun,
) error {
	pending, err := parsePublication(locator, headSHA, run)
	if err != nil {
		return err
	}
	if err := runs.publish(ctx, pending); err != nil {
		return fmt.Errorf("publish %s: %w", pending, err)
	}
	return nil
}

func (runs CheckRuns) publish(ctx context.Context, pending publication) error {
	listed, err := jsonAt[checkRunListing](ctx, runs.forge, pending.listingEndpoint())
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	chosen, err := pending.exchangeGiven(listed)
	if err != nil {
		return err
	}
	_, err = exchanged[struct{}](ctx, runs.forge, chosen)
	return err
}

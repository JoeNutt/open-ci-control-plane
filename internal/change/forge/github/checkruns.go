// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.CheckRunPublisher = CheckRuns{}

type CheckRuns struct {
	forge connection
	owner appID
}

func ParseCheckRuns(settings Settings) (CheckRuns, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return CheckRuns{}, err
	}
	owner, err := parseAppID(settings.AppID)
	if err != nil {
		return CheckRuns{}, err
	}
	return CheckRuns{forge: forge, owner: owner}, nil
}

func (runs CheckRuns) String() string {
	return "the check runs at " + runs.forge.settings.BaseURL
}

func (runs CheckRuns) GoString() string {
	return fmt.Sprintf("github.CheckRuns{settings: %#v}", runs.forge.settings)
}

func (runs CheckRuns) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	pending, err := parsePublication(locator, head, run)
	if err != nil {
		return err
	}
	for range domain.OutboundAttemptLimit {
		err = runs.publish(ctx, pending)
		if !errors.Is(err, ErrForgeUnavailable) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("publish %s: %w", pending, err)
	}
	return nil
}

func (runs CheckRuns) publish(ctx context.Context, pending publication) error {
	listed, err := exchanged[checkRunListing](ctx, runs.forge, pending.listing(runs.owner))
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	chosen, err := pending.exchangeGiven(listed, runs.owner)
	if err != nil {
		return err
	}
	return delivered(ctx, runs.forge, chosen)
}

// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.CheckRunPublisher = CheckRuns{}

type CheckRuns struct {
	forge connection
	owner appID
	pause func(ctx context.Context, wait time.Duration) error
}

const FirstRetryPause = time.Second

func ParseCheckRuns(settings Settings) (CheckRuns, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return CheckRuns{}, err
	}
	owner, err := parseAppID(settings.AppID)
	if err != nil {
		return CheckRuns{}, err
	}
	return CheckRuns{forge: forge, owner: owner, pause: waitFor}, nil
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
	if err := runs.publishWithinAttempts(ctx, pending); err != nil {
		return fmt.Errorf("publish %s: %w", pending, err)
	}
	return nil
}

func (runs CheckRuns) publishWithinAttempts(ctx context.Context, pending publication) error {
	bounded, cancel := context.WithTimeout(ctx, domain.OutboundRetryWindow)
	defer cancel()
	err := runs.publish(bounded, pending)
	wait := FirstRetryPause
	for attempt := 1; attempt < domain.OutboundAttemptLimit && errors.Is(err, ErrForgeUnavailable); attempt++ {
		pause := max(wait, askedPauseOf(err))
		if deadline, _ := bounded.Deadline(); time.Until(deadline) < pause {
			return err
		}
		if pauseErr := runs.pause(bounded, pause); pauseErr != nil {
			return errors.Join(err, pauseErr)
		}
		wait *= 2
		err = runs.publish(bounded, pending)
	}
	return err
}

func askedPauseOf(err error) time.Duration {
	var answer unavailableAnswer
	if !errors.As(err, &answer) {
		return 0
	}
	return answer.retryAfter
}

func waitFor(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

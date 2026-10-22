// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const FirstRetryPause = time.Second

var ErrRetriesExhausted = errors.New("the forge stayed unavailable through every retry")

func (runs CheckRuns) publishWithinAttempts(ctx context.Context, pending publication) error {
	bounded, cancel := context.WithTimeout(ctx, domain.OutboundRetryWindow)
	defer cancel()
	err := runs.publishRetrying(bounded, pending)
	if errors.Is(err, ErrForgeUnavailable) {
		return fmt.Errorf("%w: %w", ErrRetriesExhausted, err)
	}
	return err
}

func (runs CheckRuns) publishRetrying(bounded context.Context, pending publication) error {
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

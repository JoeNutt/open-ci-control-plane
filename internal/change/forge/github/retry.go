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

type pauseFunc func(ctx context.Context, wait time.Duration) error

func withinAttempts(ctx context.Context, pause pauseFunc, operation func(context.Context) error) error {
	bounded, cancel := context.WithTimeout(ctx, domain.OutboundRetryWindow)
	defer cancel()
	err := retrying(bounded, pause, operation)
	if errors.Is(err, ErrForgeUnavailable) {
		return fmt.Errorf("%w: %w", ErrRetriesExhausted, err)
	}
	return err
}

func retrying(bounded context.Context, pause pauseFunc, operation func(context.Context) error) error {
	err := operation(bounded)
	wait := FirstRetryPause
	for attempt := 1; attempt < domain.OutboundAttemptLimit && errors.Is(err, ErrForgeUnavailable); attempt++ {
		delay := max(wait, askedPauseOf(err))
		if deadline, _ := bounded.Deadline(); time.Until(deadline) < delay {
			return err
		}
		if pauseErr := pause(bounded, delay); pauseErr != nil {
			return errors.Join(err, pauseErr)
		}
		wait *= 2
		err = operation(bounded)
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

// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"context"
	"errors"
)

var ErrNoPlanForHead = errors.New("no plan was dispatched for this head")

var ErrRunAlreadyReported = errors.New("this head's run was already reported")

type DispatchedPlan struct {
	Locator PullRequestLocator
	Plan    TestPlan
}

type PlanLedger interface {
	DispatchedPlanFor(ctx context.Context, repository GovernedRepository, head CommitSHA) (DispatchedPlan, error)
	RecordReport(ctx context.Context, repository GovernedRepository, head CommitSHA) error
	RecordDispatch(ctx context.Context, dispatched DispatchedPlan) error
}

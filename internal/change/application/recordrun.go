// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type RecordRun struct {
	Ledger    domain.PlanLedger
	Publisher domain.CheckRunPublisher
}

func (useCase RecordRun) Execute(ctx context.Context, report domain.RunReport) error {
	dispatched, err := useCase.Ledger.DispatchedPlanFor(ctx, report.Repository(), report.Head())
	if err != nil {
		return fmt.Errorf("record the run of %s: %w", report.Head(), err)
	}
	if err := useCase.Ledger.RecordReport(ctx, report.Repository(), report.Head()); err != nil {
		return fmt.Errorf("record the run of %s: %w", report.Head(), err)
	}
	return useCase.Publisher.PublishCheckRun(ctx, dispatched.Locator, report.Head(), domain.CheckRun{
		Name: domain.TestsCheckRunName, Conclusion: dispatched.ConclusionOf(report.Results()), DecisionID: dispatched.DecisionID,
	})
}

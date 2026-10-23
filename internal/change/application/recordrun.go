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
	Callers   domain.CallerRepositories
	Escapes   domain.EscapeLog
}

type RunUpload struct {
	Token  domain.IdentityToken
	Report domain.RunReport
}

func (useCase RecordRun) Execute(ctx context.Context, upload RunUpload) error {
	report := upload.Report
	if err := callerIsRepository(ctx, useCase.Callers, upload.Token, report.Repository()); err != nil {
		return fmt.Errorf("authorise the run of %s: %w", report.Head(), err)
	}
	dispatched, err := useCase.reportedPlanOf(ctx, report)
	if err != nil {
		return fmt.Errorf("record the run of %s: %w", report.Head(), err)
	}
	return useCase.Publisher.PublishCheckRun(ctx, dispatched.Locator, report.Head(), dispatched.TestsCheckRunOf(report.Results()))
}

func (useCase RecordRun) reportedPlanOf(ctx context.Context, report domain.RunReport) (domain.DispatchedPlan, error) {
	dispatched, err := useCase.Ledger.DispatchedPlanFor(ctx, report.Repository(), report.Head())
	if err != nil {
		return domain.DispatchedPlan{}, err
	}
	if !dispatched.IsReportedBy(report) {
		return domain.DispatchedPlan{}, fmt.Errorf("%w: reported under %s, dispatched under %s",
			domain.ErrNoPlanForHead, report.DecisionID(), dispatched.DecisionID)
	}
	if err := useCase.recordEscapes(ctx, dispatched.EscapesIn(report.Results())); err != nil {
		return domain.DispatchedPlan{}, err
	}
	if err := useCase.Ledger.RecordReport(ctx, report.Repository(), report.Head()); err != nil {
		return domain.DispatchedPlan{}, err
	}
	return dispatched, nil
}

func (useCase RecordRun) recordEscapes(ctx context.Context, escapes []domain.Escape) error {
	for _, escape := range escapes {
		if err := useCase.Escapes.RecordEscape(ctx, escape); err != nil {
			return fmt.Errorf("record the %s escape of %s: %w", escape.Mode, escape.Test.Name, err)
		}
	}
	return nil
}

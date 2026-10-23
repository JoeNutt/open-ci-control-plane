// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PlanTests struct {
	Maps    domain.CoverageMaps
	History domain.ShadowHistory
	Runs    domain.ShadowRuns
	Escapes domain.EscapeHistory
	Now     func() time.Time
}

func (useCase PlanTests) Execute(ctx context.Context, subject domain.PlanSubject) (domain.TestPlan, error) {
	if useCase.History == nil {
		return domain.TestPlan{}, domain.ErrNoShadowHistory
	}
	if err := useCase.History.VerifyReadable(ctx); err != nil {
		return domain.TestPlan{}, fmt.Errorf("read the shadow history: %w", err)
	}
	coverage, err := useCase.Maps.RepositoryCoverageOf(ctx, subject.Binding.Repository())
	if err != nil {
		return domain.TestPlan{}, fmt.Errorf("read the coverage of %s: %w", subject.Binding.Repository(), err)
	}
	plan, err := domain.PlanFor(subject, coverage)
	if err != nil || !plan.IsEnforcing() {
		return plan, err
	}
	return useCase.gated(ctx, plan)
}

func (useCase PlanTests) gated(ctx context.Context, plan domain.TestPlan) (domain.TestPlan, error) {
	runs, err := useCase.Runs.ShadowedRuns(ctx)
	if err != nil {
		return domain.TestPlan{}, fmt.Errorf("gate the plan for %s: read the shadow runs: %w", plan.Repository, err)
	}
	escapes, err := useCase.Escapes.RecordedEscapes(ctx)
	if err != nil {
		return domain.TestPlan{}, fmt.Errorf("gate the plan for %s: read the escapes: %w", plan.Repository, err)
	}
	return plan.GatedOn(runs, escapes, useCase.Now()), nil
}

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
	evidence, err := shadowEvidenceIn(ctx, useCase.Runs, useCase.Escapes)
	if err != nil {
		return domain.TestPlan{}, fmt.Errorf("gate the plan for %s: %w", plan.Repository, err)
	}
	return plan.GatedOn(evidence.runs, evidence.escapes, useCase.Now()), nil
}

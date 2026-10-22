// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PlanTests struct {
	Maps    domain.CoverageMaps
	History domain.ShadowHistory
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
	return domain.PlanFor(subject, coverage)
}

// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PlanTests struct {
	Maps domain.CoverageMaps
}

type PlanRequest struct {
	Document   domain.Document
	Binding    domain.Binding
	DecisionID domain.DecisionID
	Revision   domain.BundleRevision
}

func (useCase PlanTests) Execute(ctx context.Context, request PlanRequest) (domain.TestPlan, error) {
	coverage, err := useCase.Maps.RepositoryCoverageOf(ctx, request.Binding.Repository())
	if err != nil {
		return domain.TestPlan{}, fmt.Errorf("read the coverage of %s: %w", request.Binding.Repository(), err)
	}
	return domain.PlanFor(domain.PlanBasis{
		Document:   request.Document,
		Coverage:   coverage,
		Binding:    request.Binding,
		DecisionID: request.DecisionID,
		Revision:   request.Revision,
	})
}

// SPDX-License-Identifier: GPL-3.0-or-later

package memory

import (
	"context"
	"slices"
	"sync"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type PlanDispatches struct {
	guard      sync.Mutex
	dispatched []domain.TestPlan
}

func (dispatches *PlanDispatches) DispatchPlan(_ context.Context, plan domain.TestPlan) error {
	dispatches.guard.Lock()
	defer dispatches.guard.Unlock()

	dispatches.dispatched = append(dispatches.dispatched, plan)
	return nil
}

func (dispatches *PlanDispatches) Dispatched() []domain.TestPlan {
	dispatches.guard.Lock()
	defer dispatches.guard.Unlock()

	return slices.Clone(dispatches.dispatched)
}

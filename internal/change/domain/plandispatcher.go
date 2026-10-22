// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "context"

type PlanDispatcher interface {
	DispatchPlan(ctx context.Context, plan TestPlan) error
}

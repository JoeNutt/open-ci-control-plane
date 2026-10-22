// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type plannedHeads struct {
	dispatched map[domain.CommitSHA]domain.DispatchedPlan
}

func (ledger *plannedHeads) DispatchedPlanFor(
	_ context.Context, _ domain.GovernedRepository, head domain.CommitSHA,
) (domain.DispatchedPlan, error) {
	dispatched, isPlanned := ledger.dispatched[head]
	if !isPlanned {
		return domain.DispatchedPlan{}, domain.ErrNoPlanForHead
	}
	return dispatched, nil
}

func TestAReportForAHeadWithNoDispatchedPlanIsRefusedAndPostsNothing(t *testing.T) {
	published := &publishedCheckRuns{}
	useCase := application.RecordRun{Ledger: &plannedHeads{}, Publisher: published}
	head := commitSHAOf(t, recordedHeadSHA)

	err := useCase.Execute(context.Background(), domain.RunReport{Repository: gatedLocator(t).GovernedRepository(), Head: head})

	if !errors.Is(err, domain.ErrNoPlanForHead) {
		t.Errorf("a report for an unplanned head gave %v, want %v", err, domain.ErrNoPlanForHead)
	}
	if runs := published.PublishedOn(gatedLocator(t), head); len(runs) != 0 {
		t.Errorf("posted %d check runs for an unplanned head", len(runs))
	}
}

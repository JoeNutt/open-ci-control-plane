// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type GateChange struct {
	Build      BuildDocument
	Gate       domain.Gate
	Publisher  domain.CheckRunPublisher
	Recorder   domain.DecisionRecorder
	Bindings   domain.BindingSource
	Planner    PlanTests
	Dispatcher domain.PlanDispatcher
}

func (useCase GateChange) Execute(
	ctx context.Context, locator domain.PullRequestLocator, id domain.DecisionID,
) error {
	document, err := useCase.Build.Execute(ctx, locator)
	var over domain.ChangeOverBound
	if errors.As(err, &over) {
		return useCase.refuseOverBound(context.WithoutCancel(ctx), locator, over, id)
	}
	if err != nil {
		return err
	}
	head, err := domain.ParseCommitSHA(document.PR.HeadSHA)
	if err != nil {
		return err
	}
	var failures []error
	posting := context.WithoutCancel(ctx)
	for _, decided := range useCase.Gate.DecisionsFor(ctx, document) {
		run := domain.CheckRunFor(decided, id)
		failures = append(failures,
			useCase.record(posting, decided, document, id),
			useCase.publish(posting, locator, head, run))
		if isStructuralPass(decided, run) {
			failures = append(failures, useCase.planAndDispatch(posting, document, id, decided.Decision.Revision()))
		}
	}
	return errors.Join(failures...)
}

func isStructuralPass(decided domain.FamilyDecision, run domain.CheckRun) bool {
	return decided.Family == domain.PolicyFamilyStructural && run.Conclusion == domain.ConclusionSuccess
}

func (useCase GateChange) planAndDispatch(
	ctx context.Context, document domain.Document, id domain.DecisionID, revision domain.BundleRevision,
) error {
	plan, err := useCase.planOf(ctx, domain.PlanSubject{Document: document, DecisionID: id, Revision: revision})
	if err != nil {
		return fmt.Errorf("plan the tests of %s: %w", document.PR.Repository, err)
	}
	if err := useCase.Dispatcher.DispatchPlan(ctx, plan); err != nil {
		return fmt.Errorf("dispatch the plan for %s: %w", document.PR.Repository, err)
	}
	return nil
}

func (useCase GateChange) planOf(ctx context.Context, subject domain.PlanSubject) (domain.TestPlan, error) {
	repository, err := domain.ParseGovernedRepository(subject.Document.PR.Repository)
	if err != nil {
		return domain.TestPlan{}, err
	}
	subject.Binding, err = useCase.Bindings.BindingOf(ctx, repository)
	if err != nil {
		return domain.TestPlan{}, err
	}
	return useCase.Planner.Execute(ctx, subject)
}

func (useCase GateChange) refuseOverBound(
	ctx context.Context, locator domain.PullRequestLocator, over domain.ChangeOverBound, id domain.DecisionID,
) error {
	return errors.Join(over, useCase.publish(ctx, locator, over.Head, domain.CheckRunForBreach(over, id)))
}

func (useCase GateChange) record(
	ctx context.Context, decided domain.FamilyDecision, document domain.Document, id domain.DecisionID,
) error {
	if err := useCase.writeRecordOf(ctx, decided, document, id); err != nil {
		return fmt.Errorf("record %s: %w", decided.Family, err)
	}
	return nil
}

func (useCase GateChange) writeRecordOf(
	ctx context.Context, decided domain.FamilyDecision, document domain.Document, id domain.DecisionID,
) error {
	record, err := domain.RecordFor(decided, document, id)
	if err != nil {
		return err
	}
	return useCase.Recorder.RecordDecision(ctx, record)
}

func (useCase GateChange) publish(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	if err := useCase.Publisher.PublishCheckRun(ctx, locator, head, run); err != nil {
		return fmt.Errorf("post %s: %w", run.Name, err)
	}
	return nil
}

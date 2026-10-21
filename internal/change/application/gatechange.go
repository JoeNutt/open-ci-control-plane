// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type GateChange struct {
	Build     BuildDocument
	Gate      domain.Gate
	Publisher domain.CheckRunPublisher
	Recorder  domain.DecisionRecorder
}

func (useCase GateChange) Execute(
	ctx context.Context, locator domain.PullRequestLocator, id domain.DecisionID,
) error {
	document, err := useCase.Build.Execute(ctx, locator)
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
		failures = append(failures,
			useCase.record(posting, decided, document, id),
			useCase.publish(posting, locator, head, domain.CheckRunFor(decided, id)))
	}
	return errors.Join(failures...)
}

func (useCase GateChange) record(
	ctx context.Context, decided domain.FamilyDecision, document domain.Document, id domain.DecisionID,
) error {
	record, err := domain.RecordFor(decided, document, id)
	if err != nil {
		return fmt.Errorf("record %s: %w", decided.Family, err)
	}
	if err := useCase.Recorder.RecordDecision(ctx, record); err != nil {
		return fmt.Errorf("record %s: %w", decided.Family, err)
	}
	return nil
}

func (useCase GateChange) publish(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	if err := useCase.Publisher.PublishCheckRun(ctx, locator, head, run); err != nil {
		return fmt.Errorf("post %s: %w", run.Name, err)
	}
	return nil
}

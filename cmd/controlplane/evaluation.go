// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
)

const evaluationBound = 6 * time.Minute

type evaluating struct {
	app      appForge
	gate     domain.Gate
	rules    domain.ClassificationRules
	recorder domain.DecisionRecorder
	log      *slog.Logger
}

type installationEvaluation struct {
	evaluating   evaluating
	installation int64
}

func gatesFor(installations []int64, shared evaluating) map[int64]httpapi.ChangeGate {
	gates := make(map[int64]httpapi.ChangeGate, len(installations))
	for _, installation := range installations {
		gates[installation] = installationEvaluation{evaluating: shared, installation: installation}
	}
	return gates
}

func (evaluation installationEvaluation) Execute(
	ctx context.Context, locator domain.PullRequestLocator, id domain.DecisionID,
) error {
	go func() {
		bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), evaluationBound)
		defer cancel()
		if err := evaluation.evaluated(bounded, locator, id); err != nil {
			evaluation.evaluating.log.Error(failedEvaluation,
				slog.String(decisionIDKey, id.String()), slog.String(reasonKey, err.Error()))
		}
	}()
	return nil
}

func (evaluation installationEvaluation) evaluated(
	ctx context.Context, locator domain.PullRequestLocator, id domain.DecisionID,
) error {
	forge, err := evaluation.evaluating.app.installationForgeFor(evaluation.installation)
	if err != nil {
		return err
	}
	return application.GateChange{
		Build: application.BuildDocument{
			Source:    forge.source,
			Bases:     forge.source,
			Contents:  forge.contents,
			Extractor: golang.Extractor{},
			Limits:    domain.DeclaredLimits(),
			Rules:     evaluation.evaluating.rules,
		},
		Gate:      evaluation.evaluating.gate,
		Publisher: forge.checkRuns,
		Recorder:  evaluation.evaluating.recorder,
	}.Execute(ctx, locator, id)
}

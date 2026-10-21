// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"log/slog"
	"sync"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type evaluation struct {
	gate     ChangeGate
	ids      domain.DecisionIDSource
	log      *slog.Logger
	refusing refusals
	accepted *acceptedDeliveries
}

type acceptedDeliveries struct {
	mutex sync.Mutex
	byID  map[string]acceptance
}

func (evaluating evaluation) answerFor(ctx context.Context, received delivery, stated payload) answerBody {
	evaluating.accepted.mutex.Lock()
	original, isReplay := evaluating.accepted.byID[received.id]
	evaluating.accepted.mutex.Unlock()
	if isReplay {
		return original
	}
	locator, err := stated.pullRequest()
	if err != nil {
		return evaluating.refusing.logged(ctx, received, unparseablePullRequestProblem())
	}
	id, err := evaluating.ids.NextDecisionID()
	if err != nil {
		return evaluating.failedAnswer(ctx, err, slog.String(deliveryIDKey, received.id))
	}
	if err := evaluating.gate.Execute(ctx, locator, id); err != nil {
		return evaluating.failedAnswer(ctx, err,
			slog.String(deliveryIDKey, received.id), slog.String(decisionIDKey, id.String()))
	}
	accepted := acceptanceOf(received, id)
	evaluating.accepted.mutex.Lock()
	evaluating.accepted.byID[received.id] = accepted
	evaluating.accepted.mutex.Unlock()
	return accepted
}

func (evaluating evaluation) failedAnswer(ctx context.Context, cause error, named ...slog.Attr) answerBody {
	evaluating.log.LogAttrs(ctx, slog.LevelError, failedEvaluationLine,
		append(named, slog.String(reasonKey, cause.Error()))...)
	return failedEvaluationProblem()
}

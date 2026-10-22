// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type slowEngine struct {
	decided domain.Decision
	slowest time.Duration
}

func (engine slowEngine) Decide(ctx context.Context, _ domain.Document) (domain.Decision, error) {
	select {
	case <-ctx.Done():
		return domain.Decision{}, ctx.Err()
	case <-time.After(engine.slowest):
		return engine.decided, nil
	}
}

func TestAnEngineSlowerThanTheEvaluationDeadlineIsDeadlineExceeded(t *testing.T) {
	slow := slowEngine{decided: decidingEngine(t).decision, slowest: 8 * domain.EvaluationDeadline}
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{domain.PolicyFamilyStructural: slow})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}

	decisions := gate.DecisionsFor(context.Background(), domain.Document{})

	if outcome := decisions[0].Decision.Outcome(); outcome != domain.OutcomeDeadlineExceeded {
		t.Errorf("outcome is %q, want %q", outcome, domain.OutcomeDeadlineExceeded)
	}
}

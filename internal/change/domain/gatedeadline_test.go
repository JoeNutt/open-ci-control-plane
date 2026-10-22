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

type deadlineRecordingEngine struct {
	given *time.Time
}

func (engine deadlineRecordingEngine) Decide(ctx context.Context, _ domain.Document) (domain.Decision, error) {
	*engine.given, _ = ctx.Deadline()
	return domain.Decision{}, nil
}

func TestAnEngineIsGivenExactlyTheEvaluationDeadline(t *testing.T) {
	var given time.Time
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilyStructural: deadlineRecordingEngine{given: &given},
	})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	before := time.Now()

	gate.DecisionsFor(context.Background(), domain.Document{})

	after := time.Now()
	if given.Before(before.Add(domain.EvaluationDeadline)) || given.After(after.Add(domain.EvaluationDeadline)) {
		t.Errorf("the engine's deadline is %s after the call began, want %s",
			given.Sub(before), domain.EvaluationDeadline)
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

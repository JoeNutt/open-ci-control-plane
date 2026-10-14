// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type statedEngine struct {
	decision domain.Decision
	fault    error
	asked    int
}

func (engine *statedEngine) Decide(
	_ context.Context, _ domain.Document,
) (domain.Decision, error) {
	engine.asked++
	if engine.fault != nil {
		return domain.Decision{}, engine.fault
	}
	return engine.decision, nil
}

func decidingEngine(t *testing.T) *statedEngine {
	t.Helper()

	decided, err := domain.ParseDecision([]byte(wellFormedDecision))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	return &statedEngine{decision: decided}
}

func TestAFailureInOneFamilyCannotSuppressAnother(t *testing.T) {
	broken := &statedEngine{fault: errors.New("rule conflict")}
	engines := map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilySecurity:   broken,
		domain.PolicyFamilyStructural: decidingEngine(t),
		domain.PolicyFamilyQuality:    decidingEngine(t),
		domain.PolicyFamilyRouting:    decidingEngine(t),
	}
	gate, err := domain.ParseGate(engines)
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}

	decisions := gate.DecisionsFor(context.Background(), domain.Document{})

	if len(decisions) != len(domain.DeclaredPolicyFamilies()) {
		t.Fatalf("got %d decisions, want one per family", len(decisions))
	}
	for _, decided := range decisions {
		want := domain.OutcomeDecided
		if decided.Family == domain.PolicyFamilySecurity {
			want = domain.OutcomeEngineError
		}
		if decided.Decision.Outcome() != want {
			t.Errorf("%s outcome is %q, want %q",
				decided.Family, decided.Decision.Outcome(), want)
		}
	}
	if broken.asked != 1 {
		t.Errorf("the broken family was asked %d times, want 1", broken.asked)
	}
}

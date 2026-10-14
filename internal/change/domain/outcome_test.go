// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestEveryOutcomeTheSchemaDeclaresIsNamed(t *testing.T) {
	declared := map[string]domain.Outcome{
		"decided":           domain.OutcomeDecided,
		"undefined":         domain.OutcomeUndefined,
		"engine_error":      domain.OutcomeEngineError,
		"deadline_exceeded": domain.OutcomeDeadlineExceeded,
	}

	for name, outcome := range declared {
		t.Run(name, func(t *testing.T) {
			parsed, err := domain.ParseOutcome(name)
			if err != nil {
				t.Fatalf("parse %q: %v", name, err)
			}
			if parsed != outcome {
				t.Errorf("%q parsed to %q, want %q", name, parsed, outcome)
			}
		})
	}
}

func TestParseOutcomeRefusesAnythingElse(t *testing.T) {
	for _, name := range []string{"", "allowed", "DECIDED", "timeout", "error"} {
		t.Run(name, func(t *testing.T) {
			if _, err := domain.ParseOutcome(name); !errors.Is(err, domain.ErrUnknownOutcome) {
				t.Fatalf("%q gave %v, want ErrUnknownOutcome", name, err)
			}
		})
	}
}

func TestOnlyADecidedOutcomeMayCarryAVerdict(t *testing.T) {
	carrying := map[domain.Outcome]bool{
		domain.OutcomeDecided:          true,
		domain.OutcomeUndefined:        false,
		domain.OutcomeEngineError:      false,
		domain.OutcomeDeadlineExceeded: false,
	}

	for outcome, want := range carrying {
		t.Run(string(outcome), func(t *testing.T) {
			if outcome.CarriesVerdict() != want {
				t.Fatalf("%q carries a verdict: %v, want %v",
					outcome, outcome.CarriesVerdict(), want)
			}
		})
	}
}

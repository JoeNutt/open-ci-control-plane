// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const wellFormedDecision = `{
	"input_wellformed": true,
	"schema_version_accepted": 1,
	"effect": "block",
	"violations": [
		{ "rule_id": "AUTHZ-1", "message": "auth middleware changed with no authz test",
		  "severity": "high", "owning_team": "security-platform",
		  "docs_url": "https://example.test/policy/AUTHZ-1" }
	],
	"reviewers": ["acme/security-platform"],
	"fail_closed": true
}`

func TestADecisionCarryingTheAssertionIsHonoured(t *testing.T) {
	decided, err := domain.ParseDecision([]byte(wellFormedDecision))

	if err != nil {
		t.Fatalf("parse a well-formed decision: %v", err)
	}
	if decided.Outcome() != domain.OutcomeDecided {
		t.Fatalf("outcome is %q, want %q", decided.Outcome(), domain.OutcomeDecided)
	}
	if decided.Effect() != domain.EffectBlock {
		t.Errorf("effect is %q, want %q", decided.Effect(), domain.EffectBlock)
	}
	if len(decided.Violations()) != 1 {
		t.Errorf("violations are %v, want one", decided.Violations())
	}
}

func TestADecisionWithoutTheAssertionIsUndecidedWhateverItSays(t *testing.T) {
	withheld := map[string]string{
		"the assertion absent": `{
			"schema_version_accepted": 1, "effect": "allow",
			"violations": [], "reviewers": [], "fail_closed": true
		}`,
		"the assertion false": `{
			"input_wellformed": false, "schema_version_accepted": 1, "effect": "allow",
			"violations": [], "reviewers": [], "fail_closed": true
		}`,
		"the assertion absent while blocking": `{
			"schema_version_accepted": 1, "effect": "block",
			"violations": [], "reviewers": [], "fail_closed": true
		}`,
	}

	for form, stated := range withheld {
		t.Run(form, func(t *testing.T) {
			decided, err := domain.ParseDecision([]byte(stated))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if decided.Outcome() != domain.OutcomeUndefined {
				t.Fatalf("outcome is %q, want %q", decided.Outcome(), domain.OutcomeUndefined)
			}
			if decided.Outcome().CarriesVerdict() {
				t.Fatal("an undecided document was allowed to carry a verdict")
			}
		})
	}
}

func TestADecisionAcceptingAnotherMajorIsRefused(t *testing.T) {
	mismatched := map[string]string{
		"the next major": `{
			"input_wellformed": true, "schema_version_accepted": 2, "effect": "allow",
			"violations": [], "reviewers": [], "fail_closed": true
		}`,
		"no major declared": `{
			"input_wellformed": true, "effect": "allow",
			"violations": [], "reviewers": [], "fail_closed": true
		}`,
	}

	for form, stated := range mismatched {
		t.Run(form, func(t *testing.T) {
			_, err := domain.ParseDecision([]byte(stated))

			if !errors.Is(err, domain.ErrSchemaMajorMismatch) {
				t.Fatalf("want ErrSchemaMajorMismatch, got %v", err)
			}
		})
	}
}

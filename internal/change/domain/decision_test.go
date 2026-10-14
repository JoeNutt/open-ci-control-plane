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
		  "severity": "deny", "owning_team": "security-platform",
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
		"another major with the assertion withheld": `{
			"input_wellformed": false, "schema_version_accepted": 2, "effect": "allow",
			"violations": [], "reviewers": [], "fail_closed": false
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

func TestADecisionWhoseViolationCarriesAnUnknownSeverityIsMalformed(t *testing.T) {
	stated := `{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": "block",
		"violations": [
			{ "rule_id": "AUTHZ-1", "message": "auth middleware changed with no authz test",
			  "severity": "high", "owning_team": "security-platform",
			  "docs_url": "https://example.test/policy/AUTHZ-1" }
		],
		"reviewers": [], "fail_closed": true
	}`

	_, err := domain.ParseDecision([]byte(stated))

	if !errors.Is(err, domain.ErrMalformedDecision) {
		t.Fatalf("want ErrMalformedDecision, got %v", err)
	}
	if !errors.Is(err, domain.ErrUnknownEnforcementSeverity) {
		t.Errorf("the refusal does not name the unknown severity: %v", err)
	}
}

func TestADecisionWhoseViolationOmitsItsSeverityIsMalformed(t *testing.T) {
	stated := `{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": "block",
		"violations": [
			{ "rule_id": "AUTHZ-1", "message": "auth middleware changed with no authz test",
			  "owning_team": "security-platform",
			  "docs_url": "https://example.test/policy/AUTHZ-1" }
		],
		"reviewers": [], "fail_closed": true
	}`

	_, err := domain.ParseDecision([]byte(stated))

	if !errors.Is(err, domain.ErrMalformedDecision) {
		t.Fatalf("want ErrMalformedDecision, got %v", err)
	}
}

func TestAnUndecidedDocumentKeepsTheFailurePostureItDeclares(t *testing.T) {
	declared := map[string]domain.FailurePosture{
		`"fail_closed": true`:  domain.FailurePostureClosed,
		`"fail_closed": false`: domain.FailurePostureOpen,
	}

	for field, want := range declared {
		t.Run(field, func(t *testing.T) {
			decided, err := domain.ParseDecision([]byte(`{
				"input_wellformed": false, "schema_version_accepted": 1, "effect": "allow",
				"violations": [], "reviewers": [], ` + field + `
			}`))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if decided.FailurePosture() != want {
				t.Errorf("posture is %q, want %q", decided.FailurePosture(), want)
			}
		})
	}
}

func TestADocumentThatDeclaresNoFailurePostureIsNotReadAsOpen(t *testing.T) {
	for _, wellformed := range []string{"true", "false"} {
		t.Run("input_wellformed "+wellformed, func(t *testing.T) {
			decided, err := domain.ParseDecision([]byte(`{
				"input_wellformed": ` + wellformed + `, "schema_version_accepted": 1,
				"effect": "allow", "violations": [], "reviewers": []
			}`))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if decided.FailurePosture() != "" {
				t.Errorf("an omitted fail_closed reads as %q, want no declaration",
					decided.FailurePosture())
			}
		})
	}
}

func TestADecidedDocumentReportsTheFailurePostureItDeclares(t *testing.T) {
	decided, err := domain.ParseDecision([]byte(wellFormedDecision))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if decided.FailurePosture() != domain.FailurePostureClosed {
		t.Errorf("posture is %q, want %q", decided.FailurePosture(), domain.FailurePostureClosed)
	}
}

// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
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

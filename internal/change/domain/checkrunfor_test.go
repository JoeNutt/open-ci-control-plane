// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPassedDecisionIsASuccessNamedForItsFamily(t *testing.T) {
	passed, err := domain.ParseDecision([]byte(`{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": "allow",
		"violations": [], "reviewers": [], "fail_closed": true
	}`))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}

	run := domain.CheckRunFor(domain.FamilyDecision{
		Family: domain.PolicyFamilySecurity, Decision: passed,
	}, id)

	if run.Name != "policy/security" {
		t.Errorf("name is %q, want %q", run.Name, "policy/security")
	}
	if run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionSuccess)
	}
}

func TestABlockingEffectIsAFailure(t *testing.T) {
	blocked, err := domain.ParseDecision([]byte(`{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": "block",
		"violations": [], "reviewers": [], "fail_closed": true
	}`))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}

	run := domain.CheckRunFor(domain.FamilyDecision{
		Family: domain.PolicyFamilySecurity, Decision: blocked,
	}, id)

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
}

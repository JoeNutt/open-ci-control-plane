// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"fmt"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPassedDecisionIsASuccessNamedForItsFamily(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectAllow, noViolations))

	if run.Name != "policy/security" {
		t.Errorf("name is %q, want %q", run.Name, "policy/security")
	}
	if run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionSuccess)
	}
}

func TestABlockingEffectIsAFailure(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectBlock, noViolations))

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
}

const noViolations = `[]`

func decided(t *testing.T, effect domain.Effect, violations string) domain.Decision {
	t.Helper()

	stated := fmt.Sprintf(`{
		"input_wellformed": true, "schema_version_accepted": 1, "effect": %q,
		"violations": %s, "reviewers": [], "fail_closed": true
	}`, effect, violations)
	parsed, err := domain.ParseDecision([]byte(stated))
	if err != nil {
		t.Fatalf("parse decision: %v", err)
	}
	return parsed
}

func checkRunFor(
	t *testing.T, family domain.PolicyFamily, decision domain.Decision,
) domain.CheckRun {
	t.Helper()

	id, err := domain.ParseDecisionID(wellFormedDecisionID)
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	return domain.CheckRunFor(domain.FamilyDecision{Family: family, Decision: decision}, id)
}

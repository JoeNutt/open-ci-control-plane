// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func TestAnEngineErrorIsAFailure(t *testing.T) {
	failed := evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})

	run := checkRunFor(t, domain.PolicyFamilyQuality, failed)

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
}

func TestADenyIsAFailureWhateverTheEffect(t *testing.T) {
	denied := decided(t, domain.EffectAllow, violationsAt(domain.EnforcementSeverityDeny))

	run := checkRunFor(t, domain.PolicyFamilySecurity, denied)

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
}

func TestAWarnIsNeutral(t *testing.T) {
	warned := decided(t, domain.EffectAllow, violationsAt(domain.EnforcementSeverityWarn))

	run := checkRunFor(t, domain.PolicyFamilySecurity, warned)

	if run.Conclusion != domain.ConclusionNeutral {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionNeutral)
	}
}

func TestAnUndecidedFamilyThatFailsOpenIsNeutral(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilyQuality, domain.UndecidedDecision())

	if run.Conclusion != domain.ConclusionNeutral {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionNeutral)
	}
}

func TestAViolationIsRenderedWithItsMessage(t *testing.T) {
	denied := decided(t, domain.EffectBlock, violationsAt(domain.EnforcementSeverityDeny))

	run := checkRunFor(t, domain.PolicyFamilySecurity, denied)

	if !strings.Contains(run.Text, "auth middleware changed with no authz test") {
		t.Errorf("text does not carry the violation's message:\n%s", run.Text)
	}
}

func TestAMessageCannotForgeMarkdownStructure(t *testing.T) {
	forged := "fine\n\n# Passed\n[docs](javascript:alert(1)) <img src=x onerror=alert(1)>"
	denied := decided(t, domain.EffectBlock, violationsJSON(t, domain.Violation{
		RuleID: "AUTHZ1", Message: forged, Severity: domain.EnforcementSeverityDeny,
		OwningTeam: "platform", DocsURL: "https://example.test/AUTHZ1",
	}))

	run := checkRunFor(t, domain.PolicyFamilySecurity, denied)

	encoded := `fine&#10;&#10;\# Passed&#10;\[docs\]\(javascript\:alert\(1\)\) ` +
		`\<img src\=x onerror\=alert\(1\)\>`
	if !strings.Contains(run.Text, encoded) {
		t.Errorf("text does not carry the encoded message %q:\n%s", encoded, run.Text)
	}
	for _, line := range strings.Split(run.Text, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " "), "#") {
			t.Errorf("the message forged a heading: %q", line)
		}
	}
	for _, raw := range []string{"](", " <img"} {
		if strings.Contains(run.Text, raw) {
			t.Errorf("text carries raw %q:\n%s", raw, run.Text)
		}
	}
}

const noViolations = `[]`

func violationsJSON(t *testing.T, violations ...domain.Violation) string {
	t.Helper()

	stated, err := json.Marshal(violations)
	if err != nil {
		t.Fatalf("marshal violations: %v", err)
	}
	return string(stated)
}

func violationsAt(severity domain.EnforcementSeverity) string {
	return fmt.Sprintf(`[{
		"rule_id": "AUTHZ-1", "message": "auth middleware changed with no authz test",
		"severity": %q, "owning_team": "security-platform",
		"docs_url": "https://example.test/policy/AUTHZ-1"
	}]`, severity)
}

func evaluatedWith(t *testing.T, engine domain.FamilyEngine) domain.Decision {
	t.Helper()

	engines := map[domain.PolicyFamily]domain.FamilyEngine{}
	for _, family := range domain.DeclaredPolicyFamilies() {
		engines[family] = engine
	}
	gate, err := domain.ParseGate(engines)
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	return gate.DecisionsFor(context.Background(), domain.Document{})[0].Decision
}

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

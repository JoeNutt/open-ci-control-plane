// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAPassedDecisionIsASuccessNamedForItsFamily(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectAllow))

	if run.Name != "policy/security" {
		t.Errorf("name is %q, want %q", run.Name, "policy/security")
	}
	if run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionSuccess)
	}
}

func TestABlockingEffectIsAFailure(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectBlock))

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

func TestAnEngineErrorSaysThePolicyFailedAndNamesTheOwnerNotARule(t *testing.T) {
	failed := evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})

	run := checkRunFor(t, domain.PolicyFamilySecurity, failed)

	for _, stated := range []string{
		"Policy security failed to evaluate. This is not a problem with your change.",
		"Owning team: security",
		wellFormedDecisionID,
	} {
		if !strings.Contains(run.Summary, stated) {
			t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
		}
	}
	for _, leaked := range []string{"**Rule**", "conflicting rule outputs"} {
		if strings.Contains(run.Summary+run.Text, leaked) {
			t.Errorf("a policy error rendered %q:\n%s\n%s", leaked, run.Summary, run.Text)
		}
	}
}

func TestAnExceededDeadlineIsAPolicyError(t *testing.T) {
	late := evaluatedWith(t, &statedEngine{fault: context.DeadlineExceeded})
	if late.Outcome() != domain.OutcomeDeadlineExceeded {
		t.Fatalf("fixture outcome is %q, want %q", late.Outcome(), domain.OutcomeDeadlineExceeded)
	}

	run := checkRunFor(t, domain.PolicyFamilyQuality, late)

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
	if stated := "Policy quality failed to evaluate."; !strings.Contains(run.Summary, stated) {
		t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
	}
}

func TestAnUndeclaredOutcomeIsAPolicyError(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilyQuality, domain.Decision{})

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
	if stated := "Policy quality failed to evaluate."; !strings.Contains(run.Summary, stated) {
		t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
	}
}

func TestADenyIsAFailureWhateverTheEffect(t *testing.T) {
	denied := decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityDeny))

	run := checkRunFor(t, domain.PolicyFamilySecurity, denied)

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
}

func TestAWarnIsNeutral(t *testing.T) {
	warned := decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityWarn))

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

func TestTheSummaryCarriesTheDecisionIdentifier(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectAllow))

	if !strings.Contains(run.Summary, wellFormedDecisionID) {
		t.Errorf("summary does not carry %s:\n%s", wellFormedDecisionID, run.Summary)
	}
}

func TestAnUndecidedFamilyThatFailsOpenSaysItAdmittedWithoutADecision(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilyQuality, domain.UndecidedDecision())

	for _, stated := range []string{
		"Policy quality could not be evaluated and this change was admitted without a decision.",
		wellFormedDecisionID,
	} {
		if !strings.Contains(run.Summary, stated) {
			t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
		}
	}
}

func TestAViolationIsRenderedWithItsMessage(t *testing.T) {
	denied := decided(t, domain.EffectBlock, violationAt(domain.EnforcementSeverityDeny))

	run := checkRunFor(t, domain.PolicyFamilySecurity, denied)

	if !strings.Contains(run.Text, "auth middleware changed with no authz test") {
		t.Errorf("text does not carry the violation's message:\n%s", run.Text)
	}
}

func TestAViolationNamesItsRuleAndOwningTeam(t *testing.T) {
	denied := decided(t, domain.EffectBlock, violationAt(domain.EnforcementSeverityDeny))

	run := checkRunFor(t, domain.PolicyFamilySecurity, denied)

	for _, named := range []string{`AUTHZ\-1`, `security\-platform`} {
		if !strings.Contains(run.Text, named) {
			t.Errorf("text does not name %q:\n%s", named, run.Text)
		}
	}
}

func TestAViolationLinksItsDocumentation(t *testing.T) {
	denied := decided(t, domain.EffectBlock, violationAt(domain.EnforcementSeverityDeny))

	run := checkRunFor(t, domain.PolicyFamilySecurity, denied)

	if link := "<https://example.test/policy/AUTHZ-1>"; !strings.Contains(run.Text, link) {
		t.Errorf("text does not link %s:\n%s", link, run.Text)
	}
}

func TestADocumentationLinkThatIsNotHTTPSIsWithheld(t *testing.T) {
	refused := map[string]string{
		"a script":              "javascript:alert(1)",
		"inline data":           "data:text/html,<script>alert(1)</script>",
		"plain http":            "http://example.test/policy/AUTHZ-1",
		"no scheme":             "//example.test/policy/AUTHZ-1",
		"no host":               "https:///policy/AUTHZ-1",
		"empty":                 "",
		"a space":               "https://example.test/a b",
		"an angle bracket":      "https://example.test/a>b",
		"a forged line":         "https://example.test/a\n# Passed",
		"upper-case look-alike": "HTTPS://example.test/policy/AUTHZ-1",
	}

	for form, docsURL := range refused {
		t.Run(form, func(t *testing.T) {
			violation := violationAt(domain.EnforcementSeverityDeny)
			violation.DocsURL = docsURL

			run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectBlock, violation))

			if !strings.Contains(run.Text, "- Documentation: withheld, not an https link") {
				t.Errorf("text does not say the link was withheld:\n%s", run.Text)
			}
			if docsURL != "" && strings.Contains(run.Text, docsURL) {
				t.Errorf("text carries the refused link %q:\n%s", docsURL, run.Text)
			}
		})
	}
}

func TestAMessageCannotForgeMarkdownStructure(t *testing.T) {
	forged := "fine\n\n# Passed\n[docs](javascript:alert(1)) <img src=x onerror=alert(1)>"
	denied := decided(t, domain.EffectBlock, domain.Violation{
		RuleID: "AUTHZ1", Message: forged, Severity: domain.EnforcementSeverityDeny,
		OwningTeam: "platform", DocsURL: "https://example.test/AUTHZ1",
	})

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

func violationAt(severity domain.EnforcementSeverity) domain.Violation {
	return domain.Violation{
		RuleID:     "AUTHZ-1",
		Message:    "auth middleware changed with no authz test",
		Severity:   severity,
		OwningTeam: "security-platform",
		DocsURL:    "https://example.test/policy/AUTHZ-1",
	}
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

func decided(t *testing.T, effect domain.Effect, violations ...domain.Violation) domain.Decision {
	t.Helper()

	stated, err := json.Marshal(map[string]any{
		"input_wellformed":        true,
		"schema_version_accepted": 1,
		"effect":                  effect,
		"violations":              append([]domain.Violation{}, violations...),
		"reviewers":               []string{},
		"fail_closed":             true,
	})
	if err != nil {
		t.Fatalf("marshal decision: %v", err)
	}
	parsed, err := domain.ParseDecision(stated)
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

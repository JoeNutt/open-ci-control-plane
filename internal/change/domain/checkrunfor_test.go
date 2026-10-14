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
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectAllow))

	if run.Name != "policy/security" {
		t.Errorf("name is %q, want %q", run.Name, "policy/security")
	}
	if run.Conclusion != domain.ConclusionSuccess {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionSuccess)
	}
}

func TestAnEngineErrorIsAFailure(t *testing.T) {
	failed := evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})

	run := checkRunFor(t, domain.PolicyFamilyQuality, failed)

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
}

func TestAnEngineErrorSaysThePolicyFailedAndClaimsNoOwnerOrRule(t *testing.T) {
	failed := evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})

	run := checkRunFor(t, domain.PolicyFamilySecurity, failed)

	for _, stated := range []string{
		"Policy security failed to evaluate. This is not a problem with your change.",
		wellFormedDecisionID,
	} {
		if !strings.Contains(run.Summary, stated) {
			t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
		}
	}
	for _, claimed := range []string{"**Rule**", "Owning team"} {
		if strings.Contains(run.Summary+run.Text, claimed) {
			t.Errorf("a policy error claims %q:\n%s\n%s", claimed, run.Summary, run.Text)
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

func TestTheMostRestrictiveOfEffectAndSeveritiesConcludesInAnyOrder(t *testing.T) {
	deny := violationAt(domain.EnforcementSeverityDeny)
	warn := violationAt(domain.EnforcementSeverityWarn)
	dryRun := violationAt(domain.EnforcementSeverityDryRun)
	mixed := map[string]struct {
		effect     domain.Effect
		violations []domain.Violation
		conclusion domain.Conclusion
		title      string
	}{
		"block alone":            {domain.EffectBlock, nil, domain.ConclusionFailure, "Blocked"},
		"deny under allow":       {domain.EffectAllow, []domain.Violation{deny}, domain.ConclusionFailure, "Blocked"},
		"warn alone":             {domain.EffectAllow, []domain.Violation{warn}, domain.ConclusionNeutral, "Advisory"},
		"review alone":           {domain.EffectReview, nil, domain.ConclusionSuccess, "Review required"},
		"warn then deny":         {domain.EffectAllow, []domain.Violation{warn, deny}, domain.ConclusionFailure, "Blocked"},
		"deny then warn":         {domain.EffectAllow, []domain.Violation{deny, warn}, domain.ConclusionFailure, "Blocked"},
		"dry run then warn":      {domain.EffectAllow, []domain.Violation{dryRun, warn}, domain.ConclusionNeutral, "Advisory"},
		"warn then dry run":      {domain.EffectAllow, []domain.Violation{warn, dryRun}, domain.ConclusionNeutral, "Advisory"},
		"block with a dry run":   {domain.EffectBlock, []domain.Violation{dryRun}, domain.ConclusionFailure, "Blocked"},
		"review with a dry run":  {domain.EffectReview, []domain.Violation{dryRun}, domain.ConclusionNeutral, "Review required"},
		"allow with a late deny": {domain.EffectAllow, []domain.Violation{dryRun, warn, deny}, domain.ConclusionFailure, "Blocked"},
	}

	for form, want := range mixed {
		t.Run(form, func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, want.effect, want.violations...))

			if run.Conclusion != want.conclusion || run.Title != want.title {
				t.Errorf("concluded %q titled %q, want %q titled %q",
					run.Conclusion, run.Title, want.conclusion, want.title)
			}
		})
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

func TestAnUndecidedFamilyThatFailsClosedIsAPolicyError(t *testing.T) {
	for _, family := range []domain.PolicyFamily{
		domain.PolicyFamilySecurity, domain.PolicyFamilyStructural,
	} {
		t.Run(string(family), func(t *testing.T) {
			run := checkRunFor(t, family, domain.UndecidedDecision())

			if run.Conclusion != domain.ConclusionFailure {
				t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
			}
			stated := fmt.Sprintf("Policy %s failed to evaluate.", family)
			if !strings.Contains(run.Summary, stated) {
				t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
			}
		})
	}
}

func TestAWarnIsMarkedAdvisoryAndDoesNotPreventTheMerge(t *testing.T) {
	warned := decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityWarn))

	run := checkRunFor(t, domain.PolicyFamilyQuality, warned)

	if marked := "- Enforcement: advisory"; !strings.Contains(run.Text, marked) {
		t.Errorf("text does not carry %q:\n%s", marked, run.Text)
	}
	if stated := "Advisory only. The merge is not prevented."; !strings.Contains(run.Summary, stated) {
		t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
	}
}

func TestAPassedOrBlockedVerdictIsStated(t *testing.T) {
	stated := map[string]struct {
		decision func(t *testing.T) domain.Decision
		headline string
	}{
		"passed": {
			decision: func(t *testing.T) domain.Decision { return decided(t, domain.EffectAllow) },
			headline: "Passed. No rule fired.",
		},
		"blocked": {
			decision: func(t *testing.T) domain.Decision {
				return decided(t, domain.EffectBlock, violationAt(domain.EnforcementSeverityDeny))
			},
			headline: "Blocked by the rules below.",
		},
	}

	for verdict, want := range stated {
		t.Run(verdict, func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilySecurity, want.decision(t))

			if !strings.HasPrefix(run.Summary, want.headline) {
				t.Errorf("summary does not open with %q:\n%s", want.headline, run.Summary)
			}
		})
	}
}

func TestEveryStateIsTitled(t *testing.T) {
	titled := map[string]struct {
		family   domain.PolicyFamily
		decision func(t *testing.T) domain.Decision
		title    string
	}{
		"passed": {domain.PolicyFamilySecurity, func(t *testing.T) domain.Decision {
			return decided(t, domain.EffectAllow)
		}, "Passed"},
		"recorded": {domain.PolicyFamilySecurity, func(t *testing.T) domain.Decision {
			return decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityDryRun))
		}, "Recorded only"},
		"warned": {domain.PolicyFamilySecurity, func(t *testing.T) domain.Decision {
			return decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityWarn))
		}, "Advisory"},
		"blocked": {domain.PolicyFamilySecurity, func(t *testing.T) domain.Decision {
			return decided(t, domain.EffectBlock)
		}, "Blocked"},
		"policy error": {domain.PolicyFamilySecurity, func(t *testing.T) domain.Decision {
			return evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})
		}, "Policy failed to evaluate"},
		"admitted without a decision": {domain.PolicyFamilyQuality, func(*testing.T) domain.Decision {
			return domain.UndecidedDecision()
		}, "Admitted without a decision"},
	}

	for state, want := range titled {
		t.Run(state, func(t *testing.T) {
			run := checkRunFor(t, want.family, want.decision(t))

			if run.Title != want.title {
				t.Errorf("title is %q, want %q", run.Title, want.title)
			}
		})
	}
}

func TestABlockNamingNoRuleSaysSo(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectBlock))

	if stated := "Blocked. The policy named no rule."; !strings.HasPrefix(run.Summary, stated) {
		t.Errorf("summary does not open with %q:\n%s", stated, run.Summary)
	}
	if strings.Contains(run.Summary, "rules below") {
		t.Errorf("a block with no rule points at rules below:\n%s", run.Summary)
	}
}

func TestAReviewWithNothingFiredSaysReviewIsRequired(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectReview))

	if stated := "Review required. No rule fired."; !strings.HasPrefix(run.Summary, stated) {
		t.Errorf("summary does not open with %q:\n%s", stated, run.Summary)
	}
	if strings.Contains(run.Summary, "Passed") {
		t.Errorf("a review reads as passed:\n%s", run.Summary)
	}
}

func TestAReviewWithADryRunDoesNotClaimTheChangeWasUnaffected(t *testing.T) {
	reviewed := decided(t, domain.EffectReview, violationAt(domain.EnforcementSeverityDryRun))

	run := checkRunFor(t, domain.PolicyFamilySecurity, reviewed)

	stated := "Review required. The violations below are recorded only, from a policy in dry run."
	if !strings.HasPrefix(run.Summary, stated) {
		t.Errorf("summary does not open with %q:\n%s", stated, run.Summary)
	}
	if strings.Contains(run.Summary, "did not affect") {
		t.Errorf("a review claims the change was unaffected:\n%s", run.Summary)
	}
}

func TestADryRunIsRecordedOnly(t *testing.T) {
	dryRun := decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityDryRun))

	run := checkRunFor(t, domain.PolicyFamilyQuality, dryRun)

	if run.Conclusion != domain.ConclusionNeutral {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionNeutral)
	}
	stated := "Recorded only. This policy is in dry run and did not affect your pull request."
	if !strings.Contains(run.Summary, stated) {
		t.Errorf("summary does not say %q:\n%s", stated, run.Summary)
	}
	if strings.Contains(run.Summary, "Advisory") {
		t.Errorf("a dry run is worded as advisory:\n%s", run.Summary)
	}
}

func TestEveryViolationIsMarkedWithItsEnforcement(t *testing.T) {
	marking := map[domain.EnforcementSeverity]string{
		domain.EnforcementSeverityDeny:   "- Enforcement: blocking",
		domain.EnforcementSeverityWarn:   "- Enforcement: advisory",
		domain.EnforcementSeverityDryRun: "- Enforcement: recorded only, dry run",
	}

	for severity, marked := range marking {
		t.Run(string(severity), func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilyQuality,
				decided(t, domain.EffectAllow, violationAt(severity)))

			if !strings.Contains(run.Text, marked) {
				t.Errorf("text does not carry %q:\n%s", marked, run.Text)
			}
		})
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

func TestPastTwentyViolationsTheListIsTruncatedButTheCountIsComplete(t *testing.T) {
	counts := map[string]struct {
		fired     int
		truncated string
	}{
		"54 fired":         {fired: 54, truncated: "and 34 more — 54 violations in total. This list is truncated."},
		"21 fired":         {fired: 21, truncated: "and 1 more — 21 violations in total. This list is truncated."},
		"exactly 20 fired": {fired: 20},
	}

	for form, want := range counts {
		t.Run(form, func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilySecurity,
				decided(t, domain.EffectBlock, numberedViolations(want.fired)...))

			if listed := strings.Count(run.Text, "**Rule**"); listed != min(want.fired, 20) {
				t.Errorf("listed %d violations, want %d", listed, min(want.fired, 20))
			}
			if want.truncated != "" && !strings.Contains(run.Text, want.truncated) {
				t.Errorf("text does not say %q:\n%s", want.truncated, run.Text)
			}
			if want.truncated == "" && strings.Contains(run.Text, "truncated") {
				t.Errorf("an untruncated list says it is truncated:\n%s", run.Text)
			}
			if strings.Contains(run.Text, "RULE21") {
				t.Errorf("the 21st violation is listed:\n%s", run.Text)
			}
		})
	}
}

func numberedViolations(fired int) []domain.Violation {
	violations := make([]domain.Violation, 0, fired)
	for number := 1; number <= fired; number++ {
		violation := violationAt(domain.EnforcementSeverityDeny)
		violation.RuleID = fmt.Sprintf("RULE%02d", number)
		violations = append(violations, violation)
	}
	return violations
}

func TestNoPolicySuppliedTextCanForgeMarkdownStructure(t *testing.T) {
	forged := "fine\n\n# Passed\n[docs](javascript:alert(1)) <img src=x onerror=alert(1)>"
	encoded := `fine&#10;&#10;\# Passed&#10;\[docs\]\(javascript\:alert\(1\)\) ` +
		`\<img src\=x onerror\=alert\(1\)\>`
	runs := map[string]domain.CheckRun{
		"message":     violationRunWith(t, func(forging *domain.Violation) { forging.Message = forged }),
		"rule_id":     violationRunWith(t, func(forging *domain.Violation) { forging.RuleID = forged }),
		"owning_team": violationRunWith(t, func(forging *domain.Violation) { forging.OwningTeam = forged }),
		"family": checkRunFor(t, domain.PolicyFamily(forged),
			evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})),
	}

	for field, run := range runs {
		t.Run(field, func(t *testing.T) {
			rendered := run.Summary + "\n" + run.Text
			if !strings.Contains(rendered, encoded) {
				t.Errorf("output does not carry the encoded %s %q:\n%s", field, encoded, rendered)
			}
			for _, line := range strings.Split(rendered, "\n") {
				if strings.HasPrefix(strings.TrimLeft(line, " "), "#") {
					t.Errorf("the %s forged a heading: %q", field, line)
				}
			}
			for _, raw := range []string{"](", " <img"} {
				if strings.Contains(rendered, raw) {
					t.Errorf("output carries raw %q:\n%s", raw, rendered)
				}
			}
		})
	}
}

func violationRunWith(t *testing.T, forge func(*domain.Violation)) domain.CheckRun {
	t.Helper()

	violation := violationAt(domain.EnforcementSeverityDeny)
	forge(&violation)
	return checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectBlock, violation))
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

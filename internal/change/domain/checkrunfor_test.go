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

	if stated := "Policy security failed to evaluate. This is not a problem with your change."; run.Headline != stated {
		t.Errorf("headline is %q, want %q", run.Headline, stated)
	}
	if len(run.Violations) != 0 || run.ViolationCount != 0 {
		t.Errorf("a policy error carries rules: %+v", run.Violations)
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
	if stated := "Policy quality failed to evaluate."; !strings.Contains(run.Headline, stated) {
		t.Errorf("headline does not say %q:\n%s", stated, run.Headline)
	}
}

func TestAnUndeclaredOutcomeIsAPolicyError(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilyQuality, domain.Decision{})

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
	if stated := "Policy quality failed to evaluate."; !strings.Contains(run.Headline, stated) {
		t.Errorf("headline does not say %q:\n%s", stated, run.Headline)
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
		headline   string
	}{
		"block alone":            {domain.EffectBlock, nil, domain.ConclusionFailure, "Blocked", ""},
		"deny under allow":       {domain.EffectAllow, []domain.Violation{deny}, domain.ConclusionFailure, "Blocked", ""},
		"warn alone":             {domain.EffectAllow, []domain.Violation{warn}, domain.ConclusionNeutral, "Advisory", ""},
		"review alone":           {domain.EffectReview, nil, domain.ConclusionSuccess, "Review required", ""},
		"warn then deny":         {domain.EffectAllow, []domain.Violation{warn, deny}, domain.ConclusionFailure, "Blocked", ""},
		"deny then warn":         {domain.EffectAllow, []domain.Violation{deny, warn}, domain.ConclusionFailure, "Blocked", ""},
		"dry run then warn":      {domain.EffectAllow, []domain.Violation{dryRun, warn}, domain.ConclusionNeutral, "Advisory", ""},
		"warn then dry run":      {domain.EffectAllow, []domain.Violation{warn, dryRun}, domain.ConclusionNeutral, "Advisory", ""},
		"block with a dry run":   {domain.EffectBlock, []domain.Violation{dryRun}, domain.ConclusionFailure, "Blocked", ""},
		"review with a dry run":  {domain.EffectReview, []domain.Violation{dryRun}, domain.ConclusionNeutral, "Review required", ""},
		"allow with a late deny": {domain.EffectAllow, []domain.Violation{dryRun, warn, deny}, domain.ConclusionFailure, "Blocked", ""},
		"deny under review": {domain.EffectReview, []domain.Violation{deny}, domain.ConclusionFailure, "Blocked",
			"Blocked by the rules below."},
		"warn and dry run under review": {domain.EffectReview, []domain.Violation{warn, dryRun}, domain.ConclusionNeutral,
			"Review required", "Review required. The violations below are advisory or recorded only."},
		"warn under review": {domain.EffectReview, []domain.Violation{warn}, domain.ConclusionNeutral, "Review required",
			"Review required. The violations below are advisory or recorded only."},
	}

	for form, want := range mixed {
		t.Run(form, func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, want.effect, want.violations...))

			if run.Conclusion != want.conclusion || run.Title != want.title {
				t.Errorf("concluded %q titled %q, want %q titled %q",
					run.Conclusion, run.Title, want.conclusion, want.title)
			}
			if want.headline != "" && run.Headline != want.headline {
				t.Errorf("headline is %q, want %q", run.Headline, want.headline)
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

func TestEveryRunCarriesItsDecisionIdentifier(t *testing.T) {
	runs := map[string]domain.CheckRun{
		"decided":   checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectAllow)),
		"undecided": checkRunFor(t, domain.PolicyFamilyQuality, domain.UndecidedDecision()),
		"policy error": checkRunFor(t, domain.PolicyFamilySecurity,
			evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})),
	}

	for state, run := range runs {
		t.Run(state, func(t *testing.T) {
			if run.DecisionID.String() != wellFormedDecisionID {
				t.Errorf("decision identifier is %q, want %q", run.DecisionID, wellFormedDecisionID)
			}
		})
	}
}

func TestTheViolationCountIsAlwaysComplete(t *testing.T) {
	for _, fired := range []int{0, 1, 20, 21, 54} {
		t.Run(fmt.Sprint(fired), func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilySecurity,
				decided(t, domain.EffectBlock, numberedViolations(fired)...))

			if run.ViolationCount != fired {
				t.Errorf("violation count is %d, want %d", run.ViolationCount, fired)
			}
		})
	}
}

func TestTheFirstTwentyViolationsAreCarriedPlainlyInOrder(t *testing.T) {
	fired := numberedViolations(21)
	fired[0].Message = "a `quoted` *message* with <html>"

	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectBlock, fired...))

	if len(run.Violations) != 20 {
		t.Fatalf("carried %d violations, want 20", len(run.Violations))
	}
	for index, carried := range run.Violations {
		want := domain.CheckRunViolation{
			RuleID:     fired[index].RuleID,
			Message:    fired[index].Message,
			OwningTeam: fired[index].OwningTeam,
			DocsURL:    fired[index].DocsURL,
			Marking:    "blocking",
		}
		if carried != want {
			t.Errorf("violation %d is %+v, want %+v", index, carried, want)
		}
	}
}

func TestAnUndecidedFamilyThatFailsOpenSaysItAdmittedWithoutADecision(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilyQuality, domain.UndecidedDecision())

	for _, stated := range []string{
		"Policy quality could not be evaluated and this change was admitted without a decision.",
	} {
		if !strings.Contains(run.Headline, stated) {
			t.Errorf("headline does not say %q:\n%s", stated, run.Headline)
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
			if !strings.Contains(run.Headline, stated) {
				t.Errorf("headline does not say %q:\n%s", stated, run.Headline)
			}
		})
	}
}

func TestAWarnIsMarkedAdvisoryAndDoesNotPreventTheMerge(t *testing.T) {
	warned := decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityWarn))

	run := checkRunFor(t, domain.PolicyFamilyQuality, warned)

	if marked := "advisory"; run.Violations[0].Marking != marked {
		t.Errorf("marking is %q, want %q", run.Violations[0].Marking, marked)
	}
	if stated := "Advisory only. The merge is not prevented."; !strings.Contains(run.Headline, stated) {
		t.Errorf("headline does not say %q:\n%s", stated, run.Headline)
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

			if !strings.HasPrefix(run.Headline, want.headline) {
				t.Errorf("headline does not open with %q:\n%s", want.headline, run.Headline)
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

	if stated := "Blocked. The policy named no rule."; !strings.HasPrefix(run.Headline, stated) {
		t.Errorf("headline does not open with %q:\n%s", stated, run.Headline)
	}
	if strings.Contains(run.Headline, "rules below") {
		t.Errorf("a block with no rule points at rules below:\n%s", run.Headline)
	}
}

func TestAReviewWithNothingFiredSaysReviewIsRequired(t *testing.T) {
	run := checkRunFor(t, domain.PolicyFamilySecurity, decided(t, domain.EffectReview))

	if stated := "Review required. No rule fired."; !strings.HasPrefix(run.Headline, stated) {
		t.Errorf("headline does not open with %q:\n%s", stated, run.Headline)
	}
	if strings.Contains(run.Headline, "Passed") {
		t.Errorf("a review reads as passed: %s", run.Headline)
	}
}

func TestAReviewWithADryRunDoesNotClaimTheChangeWasUnaffected(t *testing.T) {
	reviewed := decided(t, domain.EffectReview, violationAt(domain.EnforcementSeverityDryRun))

	run := checkRunFor(t, domain.PolicyFamilySecurity, reviewed)

	stated := "Review required. The violations below are recorded only, from a policy in dry run."
	if !strings.HasPrefix(run.Headline, stated) {
		t.Errorf("headline does not open with %q:\n%s", stated, run.Headline)
	}
	if strings.Contains(run.Headline, "did not affect") {
		t.Errorf("a review claims the change was unaffected:\n%s", run.Headline)
	}
}

func TestAReviewWithAWarningDoesNotClaimTheMergeIsUnprevented(t *testing.T) {
	reviewed := decided(t, domain.EffectReview, violationAt(domain.EnforcementSeverityWarn))

	run := checkRunFor(t, domain.PolicyFamilySecurity, reviewed)

	stated := "Review required. The violations below are advisory or recorded only."
	if !strings.HasPrefix(run.Headline, stated) {
		t.Errorf("headline does not open with %q:\n%s", stated, run.Headline)
	}
	if strings.Contains(run.Headline, "not prevented") {
		t.Errorf("a review claims the merge is not prevented:\n%s", run.Headline)
	}
}

func TestABlockWhoseViolationsDidNotBlockDoesNotBlameThem(t *testing.T) {
	for _, severity := range []domain.EnforcementSeverity{
		domain.EnforcementSeverityDryRun, domain.EnforcementSeverityWarn,
	} {
		t.Run(string(severity), func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilySecurity,
				decided(t, domain.EffectBlock, violationAt(severity)))

			stated := "Blocked by the policy. The violations below did not cause the block."
			if !strings.HasPrefix(run.Headline, stated) {
				t.Errorf("headline does not open with %q:\n%s", stated, run.Headline)
			}
			if run.Conclusion != domain.ConclusionFailure {
				t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
			}
		})
	}
}

func TestEveryStateCarriesAPlainHeadline(t *testing.T) {
	headed := map[string]struct {
		family   domain.PolicyFamily
		decision func(t *testing.T) domain.Decision
		headline string
	}{
		"passed": {domain.PolicyFamilySecurity, func(t *testing.T) domain.Decision {
			return decided(t, domain.EffectAllow)
		}, "Passed. No rule fired."},
		"policy error": {domain.PolicyFamilySecurity, func(t *testing.T) domain.Decision {
			return evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})
		}, "Policy security failed to evaluate. This is not a problem with your change."},
		"admitted": {domain.PolicyFamilyQuality, func(*testing.T) domain.Decision {
			return domain.UndecidedDecision()
		}, "Policy quality could not be evaluated and this change was admitted without a decision."},
	}

	headed["punctuated family"] = struct {
		family   domain.PolicyFamily
		decision func(t *testing.T) domain.Decision
		headline string
	}{domain.PolicyFamily("supply-chain"), func(t *testing.T) domain.Decision {
		return evaluatedWith(t, &statedEngine{fault: errors.New("conflicting rule outputs")})
	}, "Policy supply-chain failed to evaluate. This is not a problem with your change."}

	for state, want := range headed {
		t.Run(state, func(t *testing.T) {
			run := checkRunFor(t, want.family, want.decision(t))

			if run.Headline != want.headline {
				t.Errorf("headline is %q, want %q", run.Headline, want.headline)
			}
		})
	}
}

func TestADryRunIsRecordedOnly(t *testing.T) {
	dryRun := decided(t, domain.EffectAllow, violationAt(domain.EnforcementSeverityDryRun))

	run := checkRunFor(t, domain.PolicyFamilyQuality, dryRun)

	if run.Conclusion != domain.ConclusionNeutral {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionNeutral)
	}
	stated := "Recorded only. This policy is in dry run and did not affect your pull request."
	if !strings.Contains(run.Headline, stated) {
		t.Errorf("headline does not say %q:\n%s", stated, run.Headline)
	}
	if strings.Contains(run.Headline, "Advisory") {
		t.Errorf("a dry run is worded as advisory:\n%s", run.Headline)
	}
}

func TestEveryViolationIsMarkedWithItsEnforcement(t *testing.T) {
	marking := map[domain.EnforcementSeverity]string{
		domain.EnforcementSeverityDeny:   "blocking",
		domain.EnforcementSeverityWarn:   "advisory",
		domain.EnforcementSeverityDryRun: "recorded only",
	}

	for severity, marked := range marking {
		t.Run(string(severity), func(t *testing.T) {
			run := checkRunFor(t, domain.PolicyFamilyQuality,
				decided(t, domain.EffectAllow, violationAt(severity)))

			if run.Violations[0].Marking != marked {
				t.Errorf("marking is %q, want %q", run.Violations[0].Marking, marked)
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

func TestAnUndecidedPolicyIsGovernedByThePostureItDeclaresOverItsFamilys(t *testing.T) {
	declared := map[string]struct {
		family domain.PolicyFamily
		stated string
		want   domain.Conclusion
	}{
		"closed in an open family": {domain.PolicyFamilyQuality, "true", domain.ConclusionFailure},
		"open in a closed family":  {domain.PolicyFamilySecurity, "false", domain.ConclusionNeutral},
		"open in a family that defers to its policies": {
			domain.PolicyFamilyStructural, "false", domain.ConclusionNeutral},
	}

	for form, policy := range declared {
		t.Run(form, func(t *testing.T) {
			undecided, err := domain.ParseDecision([]byte(`{
				"input_wellformed": false, "schema_version_accepted": 1, "effect": "allow",
				"violations": [], "reviewers": [], "fail_closed": ` + policy.stated + `
			}`))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			run := checkRunFor(t, policy.family, undecided)

			if run.Conclusion != policy.want {
				t.Errorf("conclusion is %q, want %q", run.Conclusion, policy.want)
			}
		})
	}
}

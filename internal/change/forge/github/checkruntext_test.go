// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const checkRunTextTestCapacity = 65535

func TestCheckRunTextCarriesTheTitle(t *testing.T) {
	output, err := checkRunTextFor(checkRunTextFixture(t), checkRunTextTestCapacity)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if output.title != "Blocked" {
		t.Errorf("title is %q, want %q", output.title, "Blocked")
	}
}

func TestCheckRunTextSummaryIsTheHeadlineThenTheDecision(t *testing.T) {
	output, err := checkRunTextFor(checkRunTextFixture(t), checkRunTextTestCapacity)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	want := "Blocked by the rules below.\n\nDecision `dec_0123456789abcdef0123456789abcdef`"
	if output.summary != want {
		t.Errorf("summary is %q, want %q", output.summary, want)
	}
}

func TestCheckRunTextLaysOutEachViolationWithItsTenantTextInCodeSpans(t *testing.T) {
	output, err := checkRunTextFor(checkRunTextFixture(t), checkRunTextTestCapacity)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, line := range []string{
		"**Rule** ` AUTHZ-1 ` \u2014 blocking",
		"Message: ` auth middleware changed with no authz test `",
		"- Owning team: ` security-platform `",
		"- Documentation: <https://example.test/policy/AUTHZ-1>",
	} {
		if !strings.Contains(output.text, line) {
			t.Errorf("text does not carry %q:\n%s", line, output.text)
		}
	}
}

func TestCheckRunTextCountsWhatTheDomainLeftOut(t *testing.T) {
	counted := map[string]struct {
		count     int
		truncated string
	}{
		"54 counted, 1 carried": {54, "and 53 more \u2014 54 violations in total. This list is truncated.\n"},
		"1 counted, 1 carried":  {1, ""},
	}

	for form, want := range counted {
		t.Run(form, func(t *testing.T) {
			run := checkRunTextFixture(t)
			run.ViolationCount = want.count

			output, err := checkRunTextFor(run, checkRunTextTestCapacity)
			if err != nil {
				t.Fatalf("render: %v", err)
			}

			if want.truncated != "" && !strings.HasSuffix(output.text, want.truncated) {
				t.Errorf("text does not end %q:\n%s", want.truncated, output.text)
			}
			if want.truncated == "" && strings.Contains(output.text, "truncated") {
				t.Errorf("a complete list says it is truncated:\n%s", output.text)
			}
		})
	}
}

func TestCheckRunTextListsWholeViolationsThatFitAndCountsTheRest(t *testing.T) {
	const capacity = 600
	oversized := checkRunTextFixture(t)
	oversized.Violations[0].Message = strings.Repeat("m", capacity)
	several := checkRunTextFixture(t)
	for number := 2; number <= 5; number++ {
		violation := several.Violations[0]
		violation.RuleID = fmt.Sprintf("RULE%02d", number)
		several.Violations = append(several.Violations, violation)
	}
	several.ViolationCount = 5
	fitting := map[string]struct {
		run       domain.CheckRun
		listed    int
		truncated string
	}{
		"one message beyond the capacity": {oversized, 0,
			"No violation could be listed here \u2014 1 violation in total. This list is truncated.\n"},
		"five where two fit": {several, 2,
			"and 3 more \u2014 5 violations in total. This list is truncated.\n"},
	}

	for form, want := range fitting {
		t.Run(form, func(t *testing.T) {
			output, err := checkRunTextFor(want.run, capacity)
			if err != nil {
				t.Fatalf("render: %v", err)
			}

			if characters := utf8.RuneCountInString(output.text); characters > capacity {
				t.Errorf("text is %d characters, capacity %d", characters, capacity)
			}
			if listed := strings.Count(output.text, "**Rule**"); listed != want.listed {
				t.Errorf("listed %d violations, want %d:\n%s", listed, want.listed, output.text)
			}
			if !strings.HasSuffix(output.text, want.truncated) {
				t.Errorf("text does not end %q:\n%s", want.truncated, output.text)
			}
			if strings.Contains(output.text, "mmm") && want.listed == 0 {
				t.Errorf("a cut message appears:\n%s", output.text)
			}
		})
	}
}

func TestCheckRunTextSaysWhenNothingAboveIsListed(t *testing.T) {
	unlisted := map[string]struct {
		leftOut, total int
		line           string
	}{
		"one of one":   {1, 1, "No violation could be listed here \u2014 1 violation in total. This list is truncated.\n"},
		"all of 54":    {54, 54, "No violation could be listed here \u2014 54 violations in total. This list is truncated.\n"},
		"53 of 54":     {53, 54, "and 53 more \u2014 54 violations in total. This list is truncated.\n"},
		"none omitted": {0, 3, ""},
	}

	for form, want := range unlisted {
		t.Run(form, func(t *testing.T) {
			if got := checkRunTextTruncationOf(want.leftOut, want.total); got != want.line {
				t.Errorf("line is %q, want %q", got, want.line)
			}
		})
	}
}

func TestCheckRunTextRefusesWhatCannotFitEvenWithNothingListed(t *testing.T) {
	cramped := map[string]int{
		"summary beyond the capacity":         40,
		"truncation line beyond the capacity": 60,
		"a capacity of nothing":               0,
	}

	for form, capacity := range cramped {
		t.Run(form, func(t *testing.T) {
			run := checkRunTextFixture(t)
			run.Headline = "Blocked."
			run.Violations[0].Message = strings.Repeat("m", 200)

			_, err := checkRunTextFor(run, capacity)

			if !errors.Is(err, errCheckRunTextOverCapacity) {
				t.Fatalf("got %v, want errCheckRunTextOverCapacity", err)
			}
		})
	}
}

func TestCheckRunTextRefusesASummaryBeyondTheCapacityWhenTheTextIsEmpty(t *testing.T) {
	run := checkRunTextFixture(t)
	run.Violations, run.ViolationCount = nil, 0

	_, err := checkRunTextFor(run, 40)

	if !errors.Is(err, errCheckRunTextOverCapacity) {
		t.Fatalf("got %v, want errCheckRunTextOverCapacity", err)
	}
}

func TestCheckRunTextRefusesACountBelowTheViolationsCarried(t *testing.T) {
	undercounted := checkRunTextFixture(t)
	undercounted.ViolationCount = 0

	_, err := checkRunTextFor(undercounted, checkRunTextTestCapacity)

	if !errors.Is(err, errCheckRunTextUndercounted) {
		t.Fatalf("got %v, want errCheckRunTextUndercounted", err)
	}
}

func TestCheckRunTextRefusesMoreListedViolationsThanTheDomainEverCarries(t *testing.T) {
	overlisted := checkRunTextWithViolations(t, domain.ListedViolationLimit+1)

	_, err := checkRunTextFor(overlisted, checkRunTextTestCapacity)

	if !errors.Is(err, errCheckRunTextOverlisted) {
		t.Fatalf("got %v, want errCheckRunTextOverlisted", err)
	}
}

func TestCheckRunTextRefusesARunWithNoDecisionIdentifier(t *testing.T) {
	unidentified := checkRunTextFixture(t)
	unidentified.DecisionID = domain.DecisionID{}

	_, err := checkRunTextFor(unidentified, checkRunTextTestCapacity)

	if !errors.Is(err, errCheckRunTextUnidentified) {
		t.Fatalf("got %v, want errCheckRunTextUnidentified", err)
	}
}

func checkRunTextFixture(t *testing.T) domain.CheckRun {
	t.Helper()

	id, err := domain.ParseDecisionID("dec_0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("parse decision id: %v", err)
	}
	return domain.CheckRun{
		Name:       "policy/security",
		Conclusion: domain.ConclusionFailure,
		Title:      "Blocked",
		Headline:   "Blocked by the rules below.",
		DecisionID: id,
		Violations: []domain.CheckRunViolation{{
			RuleID:     "AUTHZ-1",
			Message:    "auth middleware changed with no authz test",
			OwningTeam: "security-platform",
			DocsURL:    "https://example.test/policy/AUTHZ-1",
			Marking:    "blocking",
		}},
		ViolationCount: 1,
	}
}

// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"strings"
	"testing"

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
		"**Rule** ` AUTHZ-1 ` — blocking",
		"Message: ` auth middleware changed with no authz test `",
		"- Owning team: ` security-platform `",
		"- Documentation: <https://example.test/policy/AUTHZ-1>",
	} {
		if !strings.Contains(output.text, line) {
			t.Errorf("text does not carry %q:\n%s", line, output.text)
		}
	}
}

func TestCheckRunTextRefusesARunWithNoDecisionIdentifier(t *testing.T) {
	unidentified := checkRunTextFixture(t)
	unidentified.DecisionID = domain.DecisionID{}

	_, err := checkRunTextFor(unidentified, checkRunTextTestCapacity)

	if !errors.Is(err, checkRunTextErrUnidentified) {
		t.Fatalf("got %v, want checkRunTextErrUnidentified", err)
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

// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
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

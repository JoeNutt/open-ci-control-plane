// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestGateChangePublishesOneCheckRunPerFamilyOnTheHeadCommit(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	gate, _ := recordingGate(t)
	published := &memory.CheckRuns{}
	id := decisionIDOf(t, "dec_0123456789abcdef0123456789abcdef")
	useCase := application.GateChange{
		Build: buildOverRecorded(locator, domain.File{
			Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			Additions: 12, Deletions: 3, PatchBytes: 1840,
		}),
		Gate:      gate,
		Publisher: published,
	}

	if err := useCase.Execute(context.Background(), locator, id); err != nil {
		t.Fatalf("gate change: %v", err)
	}

	runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if len(runs) != len(domain.DeclaredPolicyFamilies()) {
		t.Fatalf("published %d check runs, want one per family: %v", len(runs), runs)
	}
	for _, family := range domain.DeclaredPolicyFamilies() {
		run, present := runs["policy/"+string(family)]
		if !present {
			t.Errorf("no check run for %s", family)
			continue
		}
		if run.DecisionID != id {
			t.Errorf("%s carries decision %q, want %q", family, run.DecisionID, id)
		}
	}
}

func decisionIDOf(t *testing.T, stated string) domain.DecisionID {
	t.Helper()

	id, err := domain.ParseDecisionID(stated)
	if err != nil {
		t.Fatalf("parse decision identifier: %v", err)
	}
	return id
}

func commitSHAOf(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	head, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit SHA: %v", err)
	}
	return head
}

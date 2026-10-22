// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

const noPolicyLoaded = "no policy loaded"

func TestAnUndefinedStructuralDecisionFailsReadingNoPolicyLoaded(t *testing.T) {
	run := structuralCheckRunAfter(t, answeringEngine{decided: domain.UndecidedDecision(revisionForTest(t))})

	if run.Conclusion != domain.ConclusionFailure {
		t.Errorf("conclusion is %q, want %q", run.Conclusion, domain.ConclusionFailure)
	}
	if !strings.Contains(strings.ToLower(run.Title), noPolicyLoaded) {
		t.Errorf("title %q does not read %q", run.Title, noPolicyLoaded)
	}
}

func structuralCheckRunAfter(t *testing.T, engine domain.FamilyEngine) domain.CheckRun {
	t.Helper()

	locator := locatorOf(t, "acme/payments", 412)
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{
		domain.PolicyFamilyStructural: engine,
	})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	published := &memory.CheckRuns{}
	useCase := application.GateChange{
		Build:     buildOverAnAuthChange(t, locator),
		Gate:      gate,
		Publisher: published,
		Recorder:  &recordedDecisions{},
	}

	if err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}
	runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if len(runs) != 1 {
		t.Fatalf("published %d check runs, want one: %v", len(runs), runs)
	}
	run, present := runs["policy/"+string(domain.PolicyFamilyStructural)]
	if !present {
		t.Fatalf("no structural check run among %v", runs)
	}
	return run
}

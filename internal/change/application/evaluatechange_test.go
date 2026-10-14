// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

type recordingEngine struct {
	received []domain.Document
}

func (engine *recordingEngine) Decide(
	_ context.Context, document domain.Document,
) (domain.Decision, error) {
	engine.received = append(engine.received, document)
	return domain.UndecidedDecision(), nil
}

func TestEveryFamilyEvaluatesOnlyTheDocumentBuiltFromTheForge(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	build := application.BuildDocument{
		Source: memory.ChangeSource{
			Changes: map[domain.PullRequestLocator]memory.Change{
				locator: {
					PullRequest: domain.PullRequest{Number: 412, Repository: "acme/payments"},
					Files: []domain.File{
						{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
							Additions: 12, Deletions: 3, PatchBytes: 1840},
					},
				},
			},
		},
		Limits: domain.DeclaredLimits(),
	}
	engines := map[domain.PolicyFamily]*recordingEngine{}
	held := map[domain.PolicyFamily]domain.FamilyEngine{}
	for _, family := range domain.DeclaredPolicyFamilies() {
		engines[family] = &recordingEngine{}
		held[family] = engines[family]
	}
	gate, err := domain.ParseGate(held)
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}

	decisions, err := application.EvaluateChange{Build: build, Gate: gate}.
		Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("evaluate change: %v", err)
	}
	if len(decisions) != len(domain.DeclaredPolicyFamilies()) {
		t.Fatalf("got %d decisions, want one per family", len(decisions))
	}
	built, err := build.Execute(context.Background(), locator)
	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	for family, engine := range engines {
		if len(engine.received) != 1 {
			t.Fatalf("%s was asked %d times, want once", family, len(engine.received))
		}
		if !reflect.DeepEqual(engine.received[0], built) {
			t.Errorf("%s evaluated %+v, want the built document %+v",
				family, engine.received[0], built)
		}
	}
}

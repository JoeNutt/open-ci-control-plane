// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
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

func TestEveryFamilyDecidesOnlyTheDocumentBuiltFromTheForge(t *testing.T) {
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

	decisions, err := application.DecideChange{Build: build, Gate: gate}.
		Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("decide change: %v", err)
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
			t.Errorf("%s decided %+v, want the built document %+v",
				family, engine.received[0], built)
		}
	}
}

func TestAChangeThatCannotBeBuiltIsNeverDecided(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 413)
	oversized := domain.DeclaredLimits().MaxPatchBytesPerFile + 1
	build := application.BuildDocument{
		Source: memory.ChangeSource{
			Changes: map[domain.PullRequestLocator]memory.Change{
				locator: {
					PullRequest: domain.PullRequest{Number: 413, Repository: "acme/payments"},
					Files: []domain.File{
						{Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
							Additions: 1, PatchBytes: oversized},
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

	decisions, err := application.DecideChange{Build: build, Gate: gate}.
		Execute(context.Background(), locator)

	if !errors.Is(err, domain.ErrFilePatchTooLarge) {
		t.Fatalf("want ErrFilePatchTooLarge, got %v", err)
	}
	if decisions != nil {
		t.Errorf("a change that was never built produced decisions %v", decisions)
	}
	for family, engine := range engines {
		if len(engine.received) != 0 {
			t.Errorf("%s was asked about a change that was never built", family)
		}
	}
}

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

func TestEveryFamilyDecidesOnlyTheDocumentBuiltFromTheForge(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	build := buildOverRecorded(locator, domain.File{
		Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
		Additions: 12, Deletions: 3, PatchBytes: 1840,
	})
	gate, engines := recordingGate(t)
	useCase := application.GateChange{Build: build, Gate: gate, Publisher: &memory.CheckRuns{}}

	err := useCase.Execute(context.Background(), locator,
		decisionIDOf(t, "dec_0123456789abcdef0123456789abcdef"))

	if err != nil {
		t.Fatalf("gate change: %v", err)
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

func TestAChangeThatCannotBeBuiltIsNeverDecidedOrPublished(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 413)
	oversized := domain.DeclaredLimits().MaxPatchBytesPerFile + 1
	build := buildOverRecorded(locator, domain.File{
		Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
		Additions: 1, PatchBytes: oversized,
	})
	gate, engines := recordingGate(t)
	published := &memory.CheckRuns{}
	useCase := application.GateChange{Build: build, Gate: gate, Publisher: published}

	err := useCase.Execute(context.Background(), locator,
		decisionIDOf(t, "dec_0123456789abcdef0123456789abcdef"))

	if !errors.Is(err, domain.ErrFilePatchTooLarge) {
		t.Fatalf("want ErrFilePatchTooLarge, got %v", err)
	}
	for family, engine := range engines {
		if len(engine.received) != 0 {
			t.Errorf("%s was asked about a change that was never built", family)
		}
	}
	if runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA)); len(runs) != 0 {
		t.Errorf("a change that was never built was published: %v", runs)
	}
}

func TestAFamilyWhoseCheckRunCannotBePostedDoesNotStopTheOthers(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	gate, _ := recordingGate(t)
	recorded := &memory.CheckRuns{}
	refused := errors.New("the forge refused the post")
	useCase := application.GateChange{
		Build: buildOverRecorded(locator, domain.File{
			Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
			Additions: 12, Deletions: 3, PatchBytes: 1840,
		}),
		Gate:      gate,
		Publisher: refusingOne{name: "policy/structural", refusal: refused, rest: recorded},
	}

	err := useCase.Execute(context.Background(), locator,
		decisionIDOf(t, "dec_0123456789abcdef0123456789abcdef"))

	if !errors.Is(err, refused) {
		t.Fatalf("want the refusal reported, got %v", err)
	}
	runs := recorded.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	if want := len(domain.DeclaredPolicyFamilies()) - 1; len(runs) != want {
		t.Errorf("published %d check runs, want the other %d: %v", len(runs), want, runs)
	}
}

type refusingOne struct {
	name    string
	refusal error
	rest    domain.CheckRunPublisher
}

func (publisher refusingOne) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	if run.Name == publisher.name {
		return publisher.refusal
	}
	return publisher.rest.PublishCheckRun(ctx, locator, head, run)
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

func recordingGate(t *testing.T) (domain.Gate, map[domain.PolicyFamily]*recordingEngine) {
	t.Helper()

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
	return gate, engines
}

const recordedHeadSHA = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"

func buildOverRecorded(locator domain.PullRequestLocator, file domain.File) application.BuildDocument {
	return application.BuildDocument{
		Source: memory.ChangeSource{
			Changes: map[domain.PullRequestLocator]memory.Change{
				locator: {
					PullRequest: domain.PullRequest{
						Number: locator.Number(), Repository: locator.Repository(),
						HeadSHA: recordedHeadSHA},
					Files: []domain.File{file},
				},
			},
		},
		Limits: domain.DeclaredLimits(),
	}
}

type recordingEngine struct {
	received []domain.Document
}

func (engine *recordingEngine) Decide(
	_ context.Context, document domain.Document,
) (domain.Decision, error) {
	engine.received = append(engine.received, document)
	return domain.UndecidedDecision(), nil
}

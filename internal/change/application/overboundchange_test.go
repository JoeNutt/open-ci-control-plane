// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestAnOverBoundChangePostsOneFailureNamingTheBoundAndEvaluatesNothing(t *testing.T) {
	locator := gatedLocator(t)
	build := buildOverRecorded(t, locator, domain.File{
		Path: "internal/auth/middleware.go", Action: domain.FileActionModified,
		Additions: 1, PatchBytes: domain.DeclaredLimits().MaxPatchBytesPerFile + 1,
	})
	contents := &countingContents{}
	build.Contents = contents
	engine := &recordingEngine{revision: revisionForTest(t)}
	gate, err := domain.ParseGate(map[domain.PolicyFamily]domain.FamilyEngine{domain.PolicyFamilyStructural: engine})
	if err != nil {
		t.Fatalf("parse gate: %v", err)
	}
	published := &memory.CheckRuns{}
	useCase := application.GateChange{Build: build, Gate: gate, Publisher: published, Recorder: &recordedDecisions{}}

	err = useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, domain.ErrFilePatchTooLarge) {
		t.Errorf("error is %v, want %v", err, domain.ErrFilePatchTooLarge)
	}
	runs := published.PublishedOn(locator, commitSHAOf(t, recordedHeadSHA))
	run, present := runs[structuralCheckRunName]
	if len(runs) != 1 || !present {
		t.Fatalf("published %v, want one structural check run", runs)
	}
	if run.Conclusion != domain.ConclusionFailure || !strings.Contains(run.Headline, "patch byte bound") {
		t.Errorf("the run concludes %q reading %q, want failure naming the patch byte bound", run.Conclusion, run.Headline)
	}
	if len(engine.received) != 0 || contents.asked != 0 {
		t.Errorf("the engine was asked %d times and contents %d times, want neither", len(engine.received), contents.asked)
	}
}

type countingContents struct {
	asked int
}

func (contents *countingContents) ContentsAt(context.Context, domain.FileAtCommit) ([]byte, error) {
	contents.asked++
	return nil, errors.New("contents are not served to an over-bound change")
}

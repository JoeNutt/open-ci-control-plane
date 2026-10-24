// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
)

func TestAPassingChangeIsPlannedWithEachChangedFilesLines(t *testing.T) {
	stated := domain.StatedFile{
		File:  domain.File{Path: "infra/iam.tf", Action: domain.FileActionModified},
		Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	}
	build := evidenceBuildOver(t, gatedLocator(t), recordedFiles{}, golang.Extractor{}, stated)

	subject := plannedSubjectOf(t, build)

	if want := evidenceOf(t, nil, stated); !reflect.DeepEqual(subject.Evidence, want) {
		t.Fatalf("planned with evidence %+v, want %+v", subject.Evidence, want)
	}
}

func TestAChangedTestFileIsReadAtHeadOnceForItsImportsAndItsTests(t *testing.T) {
	locator := gatedLocator(t)
	stated := ownerTestModified()
	contents := &countedFiles{recorded: recordedAtBothEnds(t, locator, stated.File.Path, ownerTestAtBase, ownerTestAtHead)}

	plannedSubjectOf(t, evidenceBuildOver(t, locator, contents, golang.Extractor{}, stated))

	if contents.asked != 2 {
		t.Fatalf("the forge was asked for %d file contents, want one at base and one at head", contents.asked)
	}
}

func TestATestFileTheChangeRemovesIsPlannedWithNoEditedTests(t *testing.T) {
	locator := gatedLocator(t)
	stated := domain.StatedFile{File: domain.File{Path: "internal/auth/owner_test.go", Action: domain.FileActionRemoved}}
	contents := recordedAt(t, locator, recordedBaseSHA, stated.File.Path, ownerTestAtBase)

	subject := plannedSubjectOf(t, evidenceBuildOver(t, locator, contents, golang.Extractor{}, stated))

	if want := evidenceOf(t, nil, stated); !reflect.DeepEqual(subject.Evidence, want) {
		t.Fatalf("planned with evidence %+v, want %+v", subject.Evidence, want)
	}
}

func TestATestFileWhoseBodyDoesNotParseRunsTheFullMatrixNamingWhy(t *testing.T) {
	locator := gatedLocator(t)
	stated := ownerTestModified()
	broken := "package auth\n\nimport \"testing\"\n\nfunc TestAdmitsAnOwner(t *testing.T) {\n\tt.Log(1) :=\n}\n"
	dispatched := &dispatchedPlans{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, dispatched)
	useCase.Build = evidenceBuildOver(t, locator,
		recordedAtBothEnds(t, locator, stated.File.Path, ownerTestAtBase, broken), golang.Extractor{}, stated)

	err := useCase.Execute(context.Background(), locator, decisionIDOf(t, gatedDecisionID))

	if !errors.Is(err, golang.ErrUnparsableSource) {
		t.Errorf("the parse failure was not returned for the log: %v", err)
	}
	plans := dispatched.Dispatched()
	if len(plans) != 1 || !plans[0].FullMatrix || plans[0].Trigger != domain.PlanTriggerPlanFailed {
		t.Errorf("an unparsable test file dispatched %+v, want the full matrix with %q", plans, domain.PlanTriggerPlanFailed)
	}
}

func TestATestFileTheImportExtractorLeavesUnclaimedIsReadAtHeadForItsTests(t *testing.T) {
	locator := gatedLocator(t)
	stated := ownerTestModified()
	contents := recordedAt(t, locator, recordedHeadSHA, stated.File.Path, ownerTestAtHead)

	subject := plannedSubjectOf(t, evidenceBuildOver(t, locator, contents, claimsNothing{}, stated))

	if want := evidenceOf(t, editedTestsAtHead(t, stated, ownerTestAtHead), stated); !reflect.DeepEqual(subject.Evidence, want) {
		t.Fatalf("planned with evidence %+v, want %+v", subject.Evidence, want)
	}
}

func plannedSubjectOf(t *testing.T, build application.BuildDocument) domain.PlanSubject {
	t.Helper()

	planner := &recordingPlanner{}
	useCase := passingGateChange(t, &publishedCheckRuns{}, &dispatchedPlans{})
	useCase.Build = build
	useCase.Planner = planner
	if err := useCase.Execute(context.Background(), gatedLocator(t), decisionIDOf(t, gatedDecisionID)); err != nil {
		t.Fatalf("gate change: %v", err)
	}
	if len(planner.planned) != 1 {
		t.Fatalf("planned %d subjects, want 1", len(planner.planned))
	}
	return planner.planned[0]
}

func evidenceOf(t *testing.T, edited []domain.EditedTests, stated ...domain.StatedFile) domain.ChangeEvidence {
	t.Helper()

	files := make([]domain.ChangedFile, 0, len(stated))
	for _, file := range stated {
		changed, err := domain.ParseChangedFile(file)
		if err != nil {
			t.Fatalf("parse changed file: %v", err)
		}
		files = append(files, changed)
	}
	binding, err := bindingForEveryRepository{}.BindingOf(context.Background(), gatedLocator(t).GovernedRepository())
	if err != nil {
		t.Fatalf("binding: %v", err)
	}
	evidence, err := domain.ParseChangeEvidence(files, edited, binding)
	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	return evidence
}

func ownerTestModified() domain.StatedFile {
	return domain.StatedFile{
		File:  domain.File{Path: "internal/auth/owner_test.go", Action: domain.FileActionModified},
		Patch: ownerTestPatch, IsPatchStated: true,
	}
}

func evidenceBuildOver(
	t *testing.T, locator domain.PullRequestLocator, contents domain.FileContents,
	imports domain.ImportExtractor, stated ...domain.StatedFile,
) application.BuildDocument {
	t.Helper()

	return application.BuildDocument{
		Source:    changeSourceStating(t, locator, stated...),
		Bases:     recordedBases(t),
		Contents:  contents,
		Extractor: imports,
		Tests:     golang.Extractor{},
		Limits:    domain.DeclaredLimits(),
	}
}

type countedFiles struct {
	recorded recordedFiles
	asked    int
}

func (contents *countedFiles) ContentsAt(ctx context.Context, file domain.FileAtCommit) ([]byte, error) {
	contents.asked++
	return contents.recorded.ContentsAt(ctx, file)
}

type claimsNothing struct{}

func (claimsNothing) Claims(string) bool { return false }

func (claimsNothing) ImportsIn(domain.BoundedSource) ([]domain.GoImportPath, error) {
	return nil, errors.New("no file is claimed, so none is asked for")
}

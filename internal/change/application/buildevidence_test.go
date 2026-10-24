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

func TestBuildDocumentReadsAChangedTestFileAtHeadOnceForItsImportsAndItsTests(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	stated := ownerTestModified()
	contents := &countedFiles{recorded: recordedAtBothEnds(t, locator, stated.File.Path, ownerTestAtBase, ownerTestAtHead)}
	useCase := evidenceBuildOver(t, locator, contents, golang.Extractor{}, stated)

	built, err := useCase.Execute(context.Background(), locator)

	if err != nil || built.EvidenceFailure != nil {
		t.Fatalf("build document: %v, evidence: %v", err, built.EvidenceFailure)
	}
	if contents.asked != 2 {
		t.Fatalf("the forge was asked for %d file contents, want one at base and one at head", contents.asked)
	}
}

func TestBuildDocumentReadsNoTestsFromATestFileTheChangeRemoves(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	stated := domain.StatedFile{File: domain.File{Path: "internal/auth/owner_test.go", Action: domain.FileActionRemoved}}
	contents := recordedAt(t, locator, recordedBaseSHA, stated.File.Path, ownerTestAtBase)
	useCase := evidenceBuildOver(t, locator, contents, golang.Extractor{}, stated)

	built, err := useCase.Execute(context.Background(), locator)

	if err != nil || built.EvidenceFailure != nil {
		t.Fatalf("build document: %v, evidence: %v", err, built.EvidenceFailure)
	}
	if built.Edited != nil {
		t.Fatalf("edited tests are %+v, want none from a removed file", built.Edited)
	}
}

func TestBuildDocumentRecordsATestFileWhoseBodyDoesNotParseAsAnEvidenceFailure(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	stated := ownerTestModified()
	broken := "package auth\n\nimport \"testing\"\n\nfunc TestAdmitsAnOwner(t *testing.T) {\n\tt.Log(1) :=\n}\n"
	contents := recordedAtBothEnds(t, locator, stated.File.Path, ownerTestAtBase, broken)
	useCase := evidenceBuildOver(t, locator, contents, golang.Extractor{}, stated)

	built, err := useCase.Execute(context.Background(), locator)

	if err != nil {
		t.Fatalf("build document: %v", err)
	}
	if !errors.Is(built.EvidenceFailure, golang.ErrUnparsableSource) || built.Edited != nil {
		t.Fatalf("evidence failure is %v with edited %+v, want %v and none",
			built.EvidenceFailure, built.Edited, golang.ErrUnparsableSource)
	}
}

func TestBuildDocumentReadsATestFileTheImportExtractorLeavesUnclaimedAtHead(t *testing.T) {
	locator := locatorOf(t, "acme/payments", 412)
	stated := ownerTestModified()
	contents := recordedAt(t, locator, recordedHeadSHA, stated.File.Path, ownerTestAtHead)
	useCase := evidenceBuildOver(t, locator, contents, claimsNothing{}, stated)

	built, err := useCase.Execute(context.Background(), locator)

	if err != nil || built.EvidenceFailure != nil {
		t.Fatalf("build document: %v, evidence: %v", err, built.EvidenceFailure)
	}
	if want := editedTestsAtHead(t, stated, ownerTestAtHead); !reflect.DeepEqual(built.Edited, want) {
		t.Fatalf("edited tests are %+v, want %+v", built.Edited, want)
	}
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

// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"errors"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestEvidenceUnderABindingIsTheChangesFilesAndTestsResolvedByItsRoots(t *testing.T) {
	changed, err := domain.ParseChangedFile(domain.StatedFile{
		File:  domain.File{Path: "internal/auth/owner_test.go", Action: domain.FileActionModified},
		Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	})
	if err != nil {
		t.Fatalf("parse changed file: %v", err)
	}
	function, err := domain.ParseTestFunction("TestAdmitsAnOwner", 1, 3)
	if err != nil {
		t.Fatalf("parse test function: %v", err)
	}
	built := BuiltChange{
		Files:  []domain.ChangedFile{changed},
		Edited: []domain.EditedTests{domain.EditedTestsIn(changed, []domain.TestFunction{function})},
	}
	binding := bindingRootedAtTheRepository(t)

	evidence, err := built.evidenceUnder(binding)

	if err != nil {
		t.Fatalf("evidence under the binding: %v", err)
	}
	want, err := domain.ParseChangeEvidence(built.Files, built.Edited, binding)
	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	if !reflect.DeepEqual(evidence, want) {
		t.Fatalf("evidence is %+v, want %+v", evidence, want)
	}
}

func TestEvidenceUnderABindingIsTheFailureTheChangeRecordedWhenItsEvidenceCouldNotBeRead(t *testing.T) {
	failure := errors.New("the change evidence could not be read")

	evidence, err := BuiltChange{EvidenceFailure: failure}.evidenceUnder(bindingRootedAtTheRepository(t))

	if !errors.Is(err, failure) || !reflect.DeepEqual(evidence, domain.ChangeEvidence{}) {
		t.Fatalf("evidence is %+v with %v, want none with %v", evidence, err, failure)
	}
}

func bindingRootedAtTheRepository(t *testing.T) domain.Binding {
	t.Helper()

	binding, err := domain.ParseBinding(domain.BindingStatement{
		Repository: "acme/payments",
		Workflow:   "open-ci-tests.yml",
		Policies: []domain.BoundPolicyStatement{{
			Use:         "clean-architecture",
			ImportRoots: []domain.ImportRootStatement{{Prefix: "example.com/app", Path: ""}},
		}},
	})
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	return binding
}

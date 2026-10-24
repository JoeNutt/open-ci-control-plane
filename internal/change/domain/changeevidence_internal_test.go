// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"reflect"
	"testing"
)

func TestChangeEvidenceCarriesEveryChangedFile(t *testing.T) {
	changed := changedFileFrom(t, StatedFile{
		File:  File{Path: "internal/auth/owner.go", Action: FileActionModified},
		Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	})

	evidence, err := ParseChangeEvidence([]ChangedFile{changed}, nil, bindingRootedAt(t, "example.com/app"))

	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	if !reflect.DeepEqual(evidence.files, []ChangedFile{changed}) {
		t.Fatalf("files are %+v, want %+v", evidence.files, changed)
	}
}

func TestChangeEvidenceNamesEachEditedTestWithItsPackage(t *testing.T) {
	changed := changedFileFrom(t, StatedFile{
		File:  File{Path: "internal/auth/owner_test.go", Action: FileActionModified},
		Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	})
	edited := EditedTestsIn(changed, []TestFunction{testFunctionFrom(t, "TestAdmitsAnOwner", 1, 3)})

	evidence, err := ParseChangeEvidence([]ChangedFile{changed}, []EditedTests{edited}, bindingRootedAt(t, "example.com/app"))

	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	pkg, err := ParseGoImportPath("example.com/app/internal/auth")
	if err != nil {
		t.Fatalf("parse go import path: %v", err)
	}
	if want := []changedTest{{pkg: pkg, name: edited.names[0]}}; !reflect.DeepEqual(evidence.tests, want) {
		t.Fatalf("tests are %+v, want %+v", evidence.tests, want)
	}
}

func TestChangeEvidenceNamesTheTestFileWhosePackageNoImportRootNames(t *testing.T) {
	changed := changedFileFrom(t, StatedFile{
		File:  File{Path: "internal/auth/owner_test.go", Action: FileActionModified},
		Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	})
	edited := EditedTestsIn(changed, []TestFunction{testFunctionFrom(t, "TestAdmitsAnOwner", 1, 3)})
	binding, err := ParseBinding(BindingStatement{
		Repository: "acme/app", Workflow: "tests.yml",
		Policies: []BoundPolicyStatement{{Use: "clean-architecture"}},
	})
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}

	evidence, err := ParseChangeEvidence([]ChangedFile{changed}, []EditedTests{edited}, binding)

	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	if !reflect.DeepEqual(evidence.testFilesOfUnknownPackage, []RepositoryPath{changed.path}) || evidence.tests != nil {
		t.Fatalf("evidence is %+v, want the test file under an unknown package and no packaged tests", evidence)
	}
}

func TestChangeEvidenceNamesEveryTestFileItReadWhetherOrNotItEditsATest(t *testing.T) {
	editing := changedFileFrom(t, StatedFile{
		File:  File{Path: "internal/auth/owner_test.go", Action: FileActionModified},
		Patch: "@@ -1,1 +1,1 @@\n-old\n+new", IsPatchStated: true,
	})
	helping := changedFileFrom(t, StatedFile{
		File:  File{Path: "internal/auth/fixture_test.go", Action: FileActionModified},
		Patch: "@@ -9,1 +9,1 @@\n-old\n+new", IsPatchStated: true,
	})
	edited := []EditedTests{
		EditedTestsIn(editing, []TestFunction{testFunctionFrom(t, "TestAdmitsAnOwner", 1, 3)}),
		EditedTestsIn(helping, []TestFunction{testFunctionFrom(t, "TestRefusesAStranger", 1, 3)}),
	}

	evidence, err := ParseChangeEvidence([]ChangedFile{editing, helping}, edited, bindingRootedAt(t, "example.com/app"))

	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	if want := []RepositoryPath{editing.path, helping.path}; !reflect.DeepEqual(evidence.testFiles, want) {
		t.Fatalf("test files are %v, want %v", evidence.testFiles, want)
	}
}

func bindingRootedAt(t *testing.T, module string) Binding {
	t.Helper()

	binding, err := ParseBinding(BindingStatement{
		Repository: "acme/app",
		Workflow:   "tests.yml",
		Policies: []BoundPolicyStatement{{
			Use:         "clean-architecture",
			ImportRoots: []ImportRootStatement{{Prefix: module, Path: ""}},
		}},
	})
	if err != nil {
		t.Fatalf("parse binding: %v", err)
	}
	return binding
}

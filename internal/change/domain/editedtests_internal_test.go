// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"testing"
)

func TestEditedTestsOfAModifiedFileAreTheTestsWhoseLinesTheChangeTouches(t *testing.T) {
	changed := changedFileFrom(t, StatedFile{
		File:  File{Path: "internal/auth/owner_test.go", Action: FileActionModified},
		Patch: "@@ -10,3 +10,3 @@\n context\n-old\n+new\n context", IsPatchStated: true,
	})
	untouched := testFunctionFrom(t, "TestAdmitsAnOwner", 1, 9)
	touched := testFunctionFrom(t, "TestRefusesAStranger", 10, 20)

	edited := EditedTestsIn(changed, []TestFunction{untouched, touched})

	if edited.path != changed.path || !slices.Equal(edited.names, []TopLevelTestName{touched.name}) {
		t.Fatalf("edited tests are %+v, want only %s in %s", edited, touched.name.name, changed.path)
	}
}

func TestEditedTestsOfAFileWhoseLinesAreUnknownAreEveryTestInIt(t *testing.T) {
	changed := changedFileFrom(t, StatedFile{
		File: File{Path: "internal/auth/owner_test.go", Action: FileActionModified},
	})
	functions := []TestFunction{
		testFunctionFrom(t, "TestAdmitsAnOwner", 1, 9),
		testFunctionFrom(t, "TestRefusesAStranger", 10, 20),
	}

	edited := EditedTestsIn(changed, functions)

	want := []TopLevelTestName{functions[0].name, functions[1].name}
	if !slices.Equal(edited.names, want) {
		t.Fatalf("edited tests are %+v, want every test in the file", edited)
	}
}

func changedFileFrom(t *testing.T, stated StatedFile) ChangedFile {
	t.Helper()

	changed, err := ParseChangedFile(stated)
	if err != nil {
		t.Fatalf("parse changed file: %v", err)
	}
	return changed
}

func testFunctionFrom(t *testing.T, name string, firstLine, lastLine int) TestFunction {
	t.Helper()

	function, err := ParseTestFunction(name, firstLine, lastLine)
	if err != nil {
		t.Fatalf("parse test function: %v", err)
	}
	return function
}

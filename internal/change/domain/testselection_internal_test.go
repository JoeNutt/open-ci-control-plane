// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const (
	selectionModule = "example.com/app"
	ordersPackage   = selectionModule + "/orders"
	ordersSource    = "orders/order.go"
	placingLine     = 1
	cancellingLine  = 5
	refundingLine   = 12
	ordersTestFile  = "orders/order_test.go"
)

func TestSelectionNamesTheTestsCoveringABaseSideChangedLine(t *testing.T) {
	evidence := evidenceOf(t, modifiedAt(ordersSource, cancellingLine))

	selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidence)

	if err != nil || trigger != PlanTriggerNone {
		t.Fatalf("selection triggered %q (%v), want %q", trigger, err, PlanTriggerNone)
	}
	want := []PackageTests{namedTestsIn(t, ordersPackage, "TestCancels")}
	if !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected %+v, want %+v", selected, want)
	}
}

func TestSelectionOfAChangedLineInAFileTheIndexHasNeverSeenIsTheFullMatrixNeverAnEmptyRow(t *testing.T) {
	for _, unseen := range []string{"orders/refund.go", "README.md", ".github/workflows/deploy.yaml"} {
		evidence := evidenceOf(t, modifiedAt(ordersSource, cancellingLine), modifiedAt(unseen, cancellingLine))

		selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidence)

		if err != nil || trigger != PlanTriggerUnmappedFile || selected != nil {
			t.Errorf("%s: selected %+v with %q (%v), want no selection and %q", unseen, selected, trigger, err, PlanTriggerUnmappedFile)
		}
	}
}

func TestSelectionRunsAPackageWholeWhenEveryTestTheIndexHoldsForItIsSelected(t *testing.T) {
	evidence := evidenceOf(t, modifiedAt(ordersSource, placingLine, cancellingLine))

	selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidence)

	want := []PackageTests{wholePackageOf(parsedImportPath(t, ordersPackage))}
	if err != nil || trigger != PlanTriggerNone || !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected %+v with %q (%v), want %+v with %q", selected, trigger, err, want, PlanTriggerNone)
	}
}

func TestSelectionOverAnIndexHoldingNoTestIsTheFullMatrixWithNoMap(t *testing.T) {
	evidence := evidenceOf(t, modifiedAt(ordersSource, cancellingLine))

	selected, trigger, err := RepositoryCoverage{}.coverageIndex().testsSelectedFor(evidence)

	if err != nil || trigger != PlanTriggerNoMap || selected != nil {
		t.Fatalf("selected %+v with %q (%v), want no selection and %q", selected, trigger, err, PlanTriggerNoMap)
	}
}

func TestSelectionOfAnIndexedFileWhoseChangedLinesAreUnknownIsEveryTestReachingIt(t *testing.T) {
	evidence := evidenceOf(t, StatedFile{File: File{Path: ordersSource, Action: FileActionModified}})

	selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidence)

	want := []PackageTests{wholePackageOf(parsedImportPath(t, ordersPackage))}
	if err != nil || trigger != PlanTriggerNone || !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected %+v with %q (%v), want %+v with %q", selected, trigger, err, want, PlanTriggerNone)
	}
}

func TestSelectionOfADeletedOrRenamedFileIsEveryTestThatReachedItsBasePath(t *testing.T) {
	for form, stated := range map[string]StatedFile{
		"deleted": {
			File:  File{Path: ordersSource, Action: FileActionRemoved},
			Patch: "@@ -1,5 +0,0 @@\n-a\n-b\n-c\n-d\n-e", IsPatchStated: true,
		},
		"renamed with an edit": {
			File: File{Path: "orders/placing.go", Action: FileActionRenamed}, PreviousPath: ordersSource,
			Patch: oneLineEditAt(cancellingLine), IsPatchStated: true,
		},
		"renamed with unknown lines": {
			File: File{Path: "orders/placing.go", Action: FileActionRenamed}, PreviousPath: ordersSource,
		},
	} {
		selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidenceOf(t, stated))

		want := []PackageTests{wholePackageOf(parsedImportPath(t, ordersPackage))}
		if err != nil || trigger != PlanTriggerNone || !reflect.DeepEqual(selected, want) {
			t.Errorf("%s: selected %+v with %q (%v), want %+v with %q", form, selected, trigger, err, want, PlanTriggerNone)
		}
	}
}

func TestSelectionRunsEveryTestTheChangeAddsOrEditsWhateverTheIndexSays(t *testing.T) {
	evidence := evidenceEditing(t, ordersTestFile,
		testFunctionFrom(t, "TestCancels", cancellingLine, cancellingLine),
		testFunctionFrom(t, "TestRefunds", refundingLine, refundingLine),
	)

	selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidence)

	want := []PackageTests{namedTestsIn(t, ordersPackage, "TestCancels", "TestRefunds")}
	if err != nil || trigger != PlanTriggerNone || !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected %+v with %q (%v), want %+v with %q", selected, trigger, err, want, PlanTriggerNone)
	}
}

func TestSelectionRunsByNameTheTestsAddedToAPackageTheIndexHoldsNoTestOf(t *testing.T) {
	evidence := evidenceEditing(t, "billing/invoice_test.go", testFunctionFrom(t, "TestInvoices", cancellingLine, cancellingLine))

	selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidence)

	want := []PackageTests{namedTestsIn(t, selectionModule+"/billing", "TestInvoices")}
	if err != nil || trigger != PlanTriggerNone || !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected %+v with %q (%v), want %+v with %q", selected, trigger, err, want, PlanTriggerNone)
	}
}

func TestSelectionRunsAPackageWholeWhenTheIndexAndTheEditedTestsTogetherSelectEveryIndexedTest(t *testing.T) {
	evidence := evidenceEditingBeside(t, []StatedFile{modifiedAt(ordersSource, placingLine)},
		ordersTestFile, testFunctionFrom(t, "TestCancels", cancellingLine, cancellingLine))

	selected, trigger, err := ordersCoverage(t).coverageIndex().testsSelectedFor(evidence)

	want := []PackageTests{wholePackageOf(parsedImportPath(t, ordersPackage))}
	if err != nil || trigger != PlanTriggerNone || !reflect.DeepEqual(selected, want) {
		t.Fatalf("selected %+v with %q (%v), want %+v with %q", selected, trigger, err, want, PlanTriggerNone)
	}
}

func ordersCoverage(t *testing.T) RepositoryCoverage {
	t.Helper()

	return coverageHolding(t, map[string][]ReportedTest{
		ordersPackage: {
			testRunning(t, "TestPlaces", fileHitting(t, ordersSource, placingLine)),
			testRunning(t, "TestCancels", fileHitting(t, ordersSource, cancellingLine)),
		},
	})
}

func modifiedAt(path string, lines ...int) StatedFile {
	hunks := make([]string, 0, len(lines))
	for _, line := range lines {
		hunks = append(hunks, oneLineEditAt(line))
	}
	return StatedFile{File: File{Path: path, Action: FileActionModified}, Patch: strings.Join(hunks, "\n"), IsPatchStated: true}
}

func oneLineEditAt(line int) string { return fmt.Sprintf("@@ -%d,1 +%d,1 @@\n-old\n+new", line, line) }

func evidenceOf(t *testing.T, stated ...StatedFile) ChangeEvidence {
	t.Helper()

	files := make([]ChangedFile, 0, len(stated))
	for _, each := range stated {
		files = append(files, changedFileFrom(t, each))
	}
	evidence, err := ParseChangeEvidence(files, nil, bindingRootedAt(t, selectionModule))
	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	return evidence
}

func evidenceEditing(t *testing.T, path string, functions ...TestFunction) ChangeEvidence {
	t.Helper()

	return evidenceEditingBeside(t, nil, path, functions...)
}

func evidenceEditingBeside(t *testing.T, sources []StatedFile, path string, functions ...TestFunction) ChangeEvidence {
	t.Helper()

	files := make([]ChangedFile, 0, len(sources)+1)
	for _, source := range sources {
		files = append(files, changedFileFrom(t, source))
	}
	lines := make([]int, 0, len(functions))
	for _, function := range functions {
		lines = append(lines, function.lines.first)
	}
	testFile := changedFileFrom(t, modifiedAt(path, lines...))
	edited := EditedTestsIn(testFile, functions)
	evidence, err := ParseChangeEvidence(append(files, testFile), []EditedTests{edited}, bindingRootedAt(t, selectionModule))
	if err != nil {
		t.Fatalf("parse change evidence: %v", err)
	}
	return evidence
}

func namedTestsIn(t *testing.T, pkg string, names ...string) PackageTests {
	t.Helper()

	parsed := make([]TopLevelTestName, 0, len(names))
	for _, name := range names {
		parsed = append(parsed, testNamed(t, name))
	}
	named, err := namedTestsOf(parsedImportPath(t, pkg), parsed)
	if err != nil {
		t.Fatalf("name tests in %s: %v", pkg, err)
	}
	return named
}

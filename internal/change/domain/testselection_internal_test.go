// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"reflect"
	"testing"
)

const (
	selectionModule = "example.com/app"
	ordersPackage   = selectionModule + "/orders"
	ordersSource    = "orders/order.go"
	placingLine     = 1
	cancellingLine  = 5
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

func ordersCoverage(t *testing.T) RepositoryCoverage {
	t.Helper()

	return coverageHolding(t, map[string][]ReportedTest{
		ordersPackage: {
			testRunning(t, "TestPlaces", fileHitting(t, ordersSource, placingLine)),
			testRunning(t, "TestCancels", fileHitting(t, ordersSource, cancellingLine)),
		},
	})
}

func modifiedAt(path string, line int) StatedFile {
	return StatedFile{File: File{Path: path, Action: FileActionModified}, Patch: oneLineEditAt(line), IsPatchStated: true}
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

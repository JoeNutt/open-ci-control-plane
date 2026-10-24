// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"slices"
	"testing"
)

func parsedImportPath(t *testing.T, stated string) GoImportPath {
	t.Helper()
	path, err := ParseGoImportPath(stated)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNamedTestsOfRefusesAPackageWithNoNames(t *testing.T) {
	_, err := namedTestsOf(parsedImportPath(t, "example.com/parser"), nil)

	if !errors.Is(err, errNoTestNames) {
		t.Fatalf("want errNoTestNames, got %v", err)
	}
}

func parsedTestName(t *testing.T, stated string) TopLevelTestName {
	t.Helper()
	name, err := ParseTopLevelTestName(stated)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func TestNamedTestsOfHoldsEachNameOnceInOrder(t *testing.T) {
	alpha, beta := parsedTestName(t, "TestAlpha"), parsedTestName(t, "TestBeta")

	named, err := namedTestsOf(parsedImportPath(t, "example.com/parser"), []TopLevelTestName{beta, alpha, beta})
	if err != nil {
		t.Fatal(err)
	}

	if want := []TopLevelTestName{alpha, beta}; !slices.Equal(named.names, want) {
		t.Fatalf("want %v, got %v", want, named.names)
	}
}

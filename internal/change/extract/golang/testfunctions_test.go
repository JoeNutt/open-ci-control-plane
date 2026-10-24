// SPDX-License-Identifier: GPL-3.0-or-later

package golang_test

import (
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
)

func TestTestFunctionsInNameEveryTopLevelTestWithTheLinesItSpans(t *testing.T) {
	functions, err := golang.Extractor{}.TestFunctionsIn(boundedFrom(t, fixtureNamed(t, "tests.go")))

	if err != nil {
		t.Fatalf("test functions in tests.go: %v", err)
	}
	want := []domain.TestFunction{
		testFunctionOf(t, "TestAdmitsAnOwner", 5, 7),
		testFunctionOf(t, "TestRefusesAStranger", 21, 24),
	}
	if !slices.Equal(functions, want) {
		t.Fatalf("test functions are %+v, want %+v", functions, want)
	}
}

func testFunctionOf(t *testing.T, name string, firstLine, lastLine int) domain.TestFunction {
	t.Helper()

	function, err := domain.ParseTestFunction(name, firstLine, lastLine)
	if err != nil {
		t.Fatalf("parse test function %s: %v", name, err)
	}
	return function
}

// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAnEntryWithNoNamesRunsItsWholePackageAndOneWithNamesRunsThoseNames(t *testing.T) {
	pkg, err := domain.ParseGoImportPath(ordersSuite)
	if err != nil {
		t.Fatal(err)
	}

	whole := domain.NewPackageTests(pkg, nil)
	named := domain.NewPackageTests(pkg, []domain.TopLevelTestName{mustParseTestName(t, effectTest)})

	if !whole.IsWholePackage() || named.IsWholePackage() || whole.Package() != pkg || named.Package() != pkg {
		t.Errorf("no names made %+v and one name made %+v, want the whole %s and a named %s", whole, named, pkg, pkg)
	}
}

func TestAnEntryNamesEachOfItsTestsOnceInOrder(t *testing.T) {
	pkg, err := domain.ParseGoImportPath(ordersSuite)
	if err != nil {
		t.Fatal(err)
	}
	alpha, beta := mustParseTestName(t, "TestAlpha"), mustParseTestName(t, "TestBeta")

	got := domain.NewPackageTests(pkg, []domain.TopLevelTestName{beta, alpha, beta}).Names()

	if !slices.Equal(got, []domain.TopLevelTestName{alpha, beta}) {
		t.Errorf("the entry names %v, want %v", got, []domain.TopLevelTestName{alpha, beta})
	}
}

func TestATopLevelTestNameReadsAsTheNameItWasParsedFrom(t *testing.T) {
	if got := mustParseTestName(t, effectTest).String(); got != effectTest {
		t.Errorf("the name reads %q, want %q", got, effectTest)
	}
}

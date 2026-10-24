// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const effectTest = "TestEffect"

func TestNewReportedTestHoldsOneFileForEachPathWithEveryLineItHit(t *testing.T) {
	name := mustParseTestName(t, effectTest)
	held := mustParseRepositoryPath(t, heldFile)
	other := mustParseRepositoryPath(t, otherHeldFile)

	reported := domain.NewReportedTest(name, []domain.TestedFile{
		domain.NewTestedFile(held, []uint32{3}),
		domain.NewTestedFile(other, []uint32{1}),
		domain.NewTestedFile(held, []uint32{2, 3}),
	})

	want := domain.NewReportedTest(name, []domain.TestedFile{
		domain.NewTestedFile(held, []uint32{2, 3}),
		domain.NewTestedFile(other, []uint32{1}),
	})
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func TestNewReportedTestHoldsNoFileItHitNoLineOf(t *testing.T) {
	name := mustParseTestName(t, effectTest)
	held := mustParseRepositoryPath(t, heldFile)
	other := mustParseRepositoryPath(t, otherHeldFile)

	reported := domain.NewReportedTest(name, []domain.TestedFile{
		domain.NewTestedFile(held, nil),
		domain.NewTestedFile(other, nil),
		domain.NewTestedFile(held, []uint32{4}),
	})

	want := domain.NewReportedTest(name, []domain.TestedFile{domain.NewTestedFile(held, []uint32{4})})
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v, want %+v", reported, want)
	}
}

func mustParseTestName(t *testing.T, stated string) domain.TopLevelTestName {
	t.Helper()

	name, err := domain.ParseTopLevelTestName(stated)
	if err != nil {
		t.Fatalf("parse test name %q: %v", stated, err)
	}
	return name
}

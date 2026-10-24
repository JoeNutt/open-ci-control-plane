// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestArtifactsReadAnLCOVBodyAsTheReportItDeclares(t *testing.T) {
	body := testSection(effectTest, record(effectPath, lineHit))

	reported, err := lcov.Artifacts{}.ArtifactOf(int64(len(body)), strings.NewReader(body)).
		ReportedTests(domain.DeclaredCoverageLimits())

	want := []domain.ReportedTest{domain.NewReportedTest(testName(t, effectTest),
		[]domain.TestedFile{domain.NewTestedFile(repositoryPath(t, effectPath), []uint32{lineHitNumber})})}
	if err != nil || !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported %+v (%v), want one test hitting %s", reported, err, effectPath)
	}
}

func TestArtifactsRefuseABodyDeclaredOverTheCompressedBoundUnread(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()

	_, err := lcov.Artifacts{}.ArtifactOf(limits.MaxCompressedBytes+1, unreadBody{t}).ReportedTests(limits)

	if !errors.Is(err, domain.ErrCompressedArtifactTooLarge) {
		t.Fatalf("error is %v, want %v", err, domain.ErrCompressedArtifactTooLarge)
	}
}

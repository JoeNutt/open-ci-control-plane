// SPDX-License-Identifier: GPL-3.0-or-later

package lcov_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestArtifactsReadAnLCOVBodyAsTheReportItDeclares(t *testing.T) {
	body := record(effectPath, lineHit)

	reported, err := lcov.Artifacts{}.ArtifactOf(int64(len(body)), strings.NewReader(body)).
		ReportedFiles(domain.DeclaredCoverageLimits())

	if err != nil || len(reported) != 1 || reported[0].Path() != repositoryPath(t, effectPath) {
		t.Fatalf("reported %+v (%v), want one entry for %s", reported, err, effectPath)
	}
}

func TestArtifactsRefuseABodyDeclaredOverTheCompressedBoundUnread(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	unread := &failingReader{}

	_, err := lcov.Artifacts{}.ArtifactOf(limits.MaxCompressedBytes+1, unread).ReportedFiles(limits)

	if !errors.Is(err, domain.ErrCompressedArtifactTooLarge) || unread.isRead {
		t.Fatalf("error %v, read %t: want ErrCompressedArtifactTooLarge with the body unread", err, unread.isRead)
	}
}

type failingReader struct{ isRead bool }

func (reader *failingReader) Read([]byte) (int, error) {
	reader.isRead = true
	return 0, errors.New("the body was read")
}

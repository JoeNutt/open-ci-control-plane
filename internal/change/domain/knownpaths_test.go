// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	heldFile      = "internal/change/domain/effect.go"
	otherHeldFile = "internal/change/domain/gate.go"
	absentFile    = "internal/change/domain/missing.go"
)

func TestKnownPathsContainOnlyWhatTheRepositoryHolds(t *testing.T) {
	held := mustParseRepositoryPath(t, heldFile)
	absent := mustParseRepositoryPath(t, absentFile)

	known := domain.NewKnownPaths([]domain.RepositoryPath{held})

	if !known.Contains(held) {
		t.Errorf("a path the repository holds is not contained")
	}
	if known.Contains(absent) {
		t.Errorf("a path the repository does not hold is contained")
	}
}

func TestParseKnownPathsHoldsEveryPathTheRepositoryStates(t *testing.T) {
	known, err := domain.ParseKnownPaths([]string{heldFile, otherHeldFile})

	if err != nil {
		t.Fatalf("parse known paths: %v", err)
	}
	if !known.Contains(mustParseRepositoryPath(t, heldFile)) || !known.Contains(mustParseRepositoryPath(t, otherHeldFile)) {
		t.Error("a stated path is not held")
	}
}

func TestParseKnownPathsRefusesAStatedPathThatIsNotARepositoryPath(t *testing.T) {
	for _, stated := range []string{"docs/release notes.md", "/etc/passwd", "internal/../go.mod"} {
		if _, err := domain.ParseKnownPaths([]string{heldFile, stated}); !errors.Is(err, domain.ErrPathOutsideRepository) {
			t.Errorf("%q: error is %v, want %v", stated, err, domain.ErrPathOutsideRepository)
		}
	}
}

func mustParseRepositoryPath(t *testing.T, path string) domain.RepositoryPath {
	t.Helper()

	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", path, err)
	}
	return parsed
}

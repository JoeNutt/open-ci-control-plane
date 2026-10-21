// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
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

func mustParseRepositoryPath(t *testing.T, path string) domain.RepositoryPath {
	t.Helper()

	parsed, err := domain.ParseRepositoryPath(path)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", path, err)
	}
	return parsed
}

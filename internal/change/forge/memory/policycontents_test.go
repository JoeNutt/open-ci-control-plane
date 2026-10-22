// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestPolicyContentsAnswersTheContentsRecordedForAFileAtAPolicyCommit(t *testing.T) {
	file := memory.PolicyFileAt{At: recordedPolicyCommit(t), Path: repositoryPathOf(t, "catalog/clean-architecture.yaml")}
	recorded := []byte("schema_version: 1\n")
	contents := memory.PolicyContents{Files: map[memory.PolicyFileAt][]byte{file: recorded}}

	answered, err := contents.ContentsAt(context.Background(), file.At, file.Path)

	if err != nil {
		t.Fatalf("contents at %+v: %v", file, err)
	}
	if !slices.Equal(answered, recorded) {
		t.Fatalf("contents are %q, want %q", answered, recorded)
	}
}

func repositoryPathOf(t *testing.T, stated string) domain.RepositoryPath {
	t.Helper()

	path, err := domain.ParseRepositoryPath(stated)
	if err != nil {
		t.Fatalf("parse repository path %q: %v", stated, err)
	}
	return path
}

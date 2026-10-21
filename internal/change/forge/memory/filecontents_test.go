// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/memory"
)

func TestFileContentsAnswersTheContentsRecordedAtACommit(t *testing.T) {
	file := domain.FileAtCommit{
		Repository: "acme/payments",
		Commit:     commitOf(t, "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"),
		Path:       "internal/auth/middleware.go",
	}
	recorded := []byte("package auth\n")
	contents := memory.FileContents{Files: map[domain.FileAtCommit][]byte{file: recorded}}

	answered, err := contents.ContentsAt(context.Background(), file)

	if err != nil {
		t.Fatalf("contents at %+v: %v", file, err)
	}
	if !slices.Equal(answered, recorded) {
		t.Fatalf("contents are %q, want %q", answered, recorded)
	}
}

func TestFileContentsRefusesAFileItHoldsNoContentsFor(t *testing.T) {
	file := domain.FileAtCommit{
		Repository: "acme/payments",
		Commit:     commitOf(t, "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"),
		Path:       "internal/auth/middleware.go",
	}

	_, err := memory.FileContents{}.ContentsAt(context.Background(), file)

	if !errors.Is(err, domain.ErrNoSuchFile) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchFile)
	}
}

func commitOf(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %q: %v", stated, err)
	}
	return commit
}

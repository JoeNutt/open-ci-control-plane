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
	file := middlewareAtHead(t)
	recorded := []byte("package auth\n")
	var contents domain.FileContents = memory.FileContents{
		Files: map[domain.FileAtCommit][]byte{file: recorded},
	}

	answered, err := contents.ContentsAt(context.Background(), file)

	if err != nil {
		t.Fatalf("contents at %+v: %v", file, err)
	}
	if !slices.Equal(answered, recorded) {
		t.Fatalf("contents are %q, want %q", answered, recorded)
	}
}

func TestFileContentsRefusesAFileItHoldsNoContentsFor(t *testing.T) {
	file := middlewareAtHead(t)

	_, err := memory.FileContents{}.ContentsAt(context.Background(), file)

	if !errors.Is(err, domain.ErrNoSuchFile) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchFile)
	}
}

func middlewareAtHead(t *testing.T) domain.FileAtCommit {
	t.Helper()

	repository, err := domain.ParseGovernedRepository("acme/payments")
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	file, err := domain.ParseFileAtCommit(repository,
		commitOf(t, "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"), "internal/auth/middleware.go")
	if err != nil {
		t.Fatalf("parse file at commit: %v", err)
	}
	return file
}

func commitOf(t *testing.T, stated string) domain.CommitSHA {
	t.Helper()

	commit, err := domain.ParseCommitSHA(stated)
	if err != nil {
		t.Fatalf("parse commit %q: %v", stated, err)
	}
	return commit
}

// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestFilesChangedRefusesAForgeThatNeverRunsOutOfPages(t *testing.T) {
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		write(t, writer, pageAsLongAsAsked(t, request))
	})
	source := sourceAt(t, server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrTooManyPages) {
		t.Fatalf("error is %v, want %v", err, github.ErrTooManyPages)
	}
}

func TestFilesChangedReadsEveryPageTheForgeSplitsTheFilesAcross(t *testing.T) {
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("page") == "2" {
			write(t, writer, `[{"filename": "infra/iam.tf", "status": "added"}]`)
			return
		}
		write(t, writer, pageAsLongAsAsked(t, request))
	})
	source := sourceAt(t, server)

	files, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
	if len(files) != github.FilesPerPage+1 {
		t.Fatalf("read %d files, want %d", len(files), github.FilesPerPage+1)
	}
	if files[github.FilesPerPage].File.Path != "infra/iam.tf" {
		t.Fatalf("the last file is %q, want infra/iam.tf", files[github.FilesPerPage].File.Path)
	}
}

func TestFilesChangedRefusesAStatusTheSchemaDoesNotDeclare(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "unchanged"}
	]`)
	source := sourceAt(t, server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrUnknownFileAction) {
		t.Fatalf("error is %v, want %v", err, domain.ErrUnknownFileAction)
	}
}

func TestFilesChangedReportsEveryFileTheForgeLists(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "modified",
		 "additions": 12, "deletions": 3, "patch": "@@ -1,3 +1,12 @@"}
	]`)
	source := sourceAt(t, server)

	files, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
	want := domain.File{
		Path:       "internal/auth/middleware.go",
		Action:     domain.FileActionModified,
		Additions:  12,
		Deletions:  3,
		PatchBytes: len("@@ -1,3 +1,12 @@"),
	}
	if len(files) != 1 || files[0].File != want {
		t.Fatalf("files are %+v, want exactly %+v", files, want)
	}
}

func TestFilesChangedCarriesThePatchAndThePreviousPathTheForgeStates(t *testing.T) {
	const patch = "@@ -1,1 +1,1 @@\n-old\n+new"
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/session.go", "status": "renamed",
		 "previous_filename": "internal/auth/token.go", "patch": "@@ -1,1 +1,1 @@\n-old\n+new"},
		{"filename": "assets/logo.png", "status": "modified"}
	]`)
	source := sourceAt(t, server)

	files, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
	renamed := domain.StatedFile{
		File:         domain.File{Path: "internal/auth/session.go", Action: domain.FileActionRenamed, PatchBytes: len(patch)},
		PreviousPath: "internal/auth/token.go", Patch: patch, IsPatchStated: true,
	}
	unpatched := domain.StatedFile{File: domain.File{Path: "assets/logo.png", Action: domain.FileActionModified}}
	if len(files) != 2 || files[0] != renamed || files[1] != unpatched {
		t.Fatalf("files are %+v, want exactly %+v and %+v", files, renamed, unpatched)
	}
}

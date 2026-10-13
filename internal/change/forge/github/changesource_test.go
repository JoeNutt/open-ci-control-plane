// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func locatorOf(t *testing.T, repository string, number int) domain.PullRequestLocator {
	t.Helper()

	locator, err := domain.ParsePullRequestLocator(repository, number)
	if err != nil {
		t.Fatalf("parse pull request locator: %v", err)
	}
	return locator
}

func serving(t *testing.T, path, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != path {
				t.Errorf("the adapter asked for %s, want %s", request.URL.Path, path)
				writer.WriteHeader(http.StatusNotFound)
				return
			}
			if _, written := writer.Write([]byte(body)); written != nil {
				t.Errorf("write response: %v", written)
			}
		}))
	t.Cleanup(server.Close)
	return server
}

func TestFilesChangedRefusesAStatusTheSchemaDoesNotDeclare(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "unchanged"}
	]`)
	source := github.ChangeSource{BaseURL: server.URL, Client: server.Client()}

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
	source := github.ChangeSource{BaseURL: server.URL, Client: server.Client()}

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
	if len(files) != 1 || files[0] != want {
		t.Fatalf("files are %+v, want exactly %+v", files, want)
	}
}

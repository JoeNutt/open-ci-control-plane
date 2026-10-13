// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func forge(t *testing.T, answer http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(answer)
	t.Cleanup(server.Close)
	return server
}

func serving(t *testing.T, path, body string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != path {
			t.Errorf("the adapter asked for %s, want %s", request.URL.Path, path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		write(t, writer, body)
	})
}

func answering(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(status)
		write(t, writer, body)
	})
}

func write(t *testing.T, writer http.ResponseWriter, body string) {
	t.Helper()

	if _, written := writer.Write([]byte(body)); written != nil {
		t.Errorf("write response: %v", written)
	}
}

func TestFilesChangedRefusesAnAnswerTheForgeDidNotServeWith200(t *testing.T) {
	server := answering(t, http.StatusInternalServerError, `[]`)
	source := github.ChangeSource{BaseURL: server.URL, Client: server.Client()}

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrUnexpectedStatus) {
		t.Fatalf("error is %v, want %v", err, github.ErrUnexpectedStatus)
	}
}

func fullPage(t *testing.T) string {
	t.Helper()

	listings := make([]string, 0, github.FilesPerPage)
	for position := range github.FilesPerPage {
		listings = append(listings,
			fmt.Sprintf(`{"filename": "internal/%d.go", "status": "added"}`, position))
	}
	return "[" + strings.Join(listings, ",") + "]"
}

func TestFilesChangedRefusesAForgeThatNeverRunsOutOfPages(t *testing.T) {
	page := fullPage(t)
	server := forge(t, func(writer http.ResponseWriter, _ *http.Request) {
		write(t, writer, page)
	})
	source := github.ChangeSource{BaseURL: server.URL, Client: server.Client()}

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrTooManyPages) {
		t.Fatalf("error is %v, want %v", err, github.ErrTooManyPages)
	}
}

func TestFilesChangedReadsEveryPageTheForgeSplitsTheFilesAcross(t *testing.T) {
	first := fullPage(t)
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("page") == "2" {
			write(t, writer, `[{"filename": "infra/iam.tf", "status": "added"}]`)
			return
		}
		write(t, writer, first)
	})
	source := github.ChangeSource{BaseURL: server.URL, Client: server.Client()}

	files, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("files changed: %v", err)
	}
	if len(files) != github.FilesPerPage+1 {
		t.Fatalf("read %d files, want %d", len(files), github.FilesPerPage+1)
	}
	if files[github.FilesPerPage].Path != "infra/iam.tf" {
		t.Fatalf("the last file is %q, want infra/iam.tf", files[github.FilesPerPage].Path)
	}
}

func TestFilesChangedReportsAForgeThatKnowsNoSuchPullRequest(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)
	source := github.ChangeSource{BaseURL: server.URL, Client: server.Client()}

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchPullRequest)
	}
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

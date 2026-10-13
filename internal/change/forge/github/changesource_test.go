// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
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

	return servingEach(t, map[string]string{path: body})
}

func servingEach(t *testing.T, bodies map[string]string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		body, served := bodies[request.URL.Path]
		if !served {
			t.Errorf("the adapter asked for %s, which this forge does not serve",
				request.URL.Path)
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
	source := sourceAt(server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrUnexpectedStatus) {
		t.Fatalf("error is %v, want %v", err, github.ErrUnexpectedStatus)
	}
}

func sourceAt(server *httptest.Server) github.ChangeSource {
	return github.ChangeSource{
		BaseURL:          server.URL,
		Client:           server.Client(),
		MaxResponseBytes: 1 << 20,
	}
}

func pageAsLongAsAsked(t *testing.T, request *http.Request) string {
	t.Helper()

	asked := request.URL.Query().Get("per_page")
	length, err := strconv.Atoi(asked)
	if err != nil {
		t.Errorf("the adapter asked for per_page=%q: %v", asked, err)
		return "[]"
	}
	listings := make([]string, 0, length)
	for position := range length {
		listings = append(listings,
			fmt.Sprintf(`{"filename": "internal/%d.go", "status": "added"}`, position))
	}
	return "[" + strings.Join(listings, ",") + "]"
}

func TestPullRequestAtReportsEveryApprovalTheForgeHolds(t *testing.T) {
	server := servingEach(t, map[string]string{
		"/repos/acme/payments/pulls/412": `{
			"number": 412,
			"user": {"login": "jo", "type": "User"},
			"author_association": "MEMBER"
		}`,
		"/repos/acme/payments/pulls/412/reviews": `[
			{"user": {"login": "sam", "type": "User"}, "state": "APPROVED"},
			{"user": {"login": "kim", "type": "User"}, "state": "COMMENTED"}
		]`,
	})
	source := sourceAt(server)

	pullRequest, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("pull request at: %v", err)
	}
	want := domain.Declared[domain.Approval]{{Login: "sam"}}
	if !reflect.DeepEqual(pullRequest.Approvals, want) {
		t.Fatalf("approvals are %+v, want %+v", pullRequest.Approvals, want)
	}
}

func TestPullRequestAtDropsAnApprovalItsReviewerWithdrew(t *testing.T) {
	server := servingEach(t, map[string]string{
		"/repos/acme/payments/pulls/412": `{
			"number": 412,
			"user": {"login": "jo", "type": "User"},
			"author_association": "MEMBER"
		}`,
		"/repos/acme/payments/pulls/412/reviews": `[
			{"user": {"login": "sam", "type": "User"}, "state": "APPROVED"},
			{"user": {"login": "sam", "type": "User"}, "state": "CHANGES_REQUESTED"},
			{"user": {"login": "kim", "type": "User"}, "state": "APPROVED"},
			{"user": {"login": "kim", "type": "User"}, "state": "COMMENTED"}
		]`,
	})
	source := sourceAt(server)

	pullRequest, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("pull request at: %v", err)
	}
	want := domain.Declared[domain.Approval]{{Login: "kim"}}
	if !reflect.DeepEqual(pullRequest.Approvals, want) {
		t.Fatalf("approvals are %+v, want %+v", pullRequest.Approvals, want)
	}
}

func TestPullRequestAtReportsTheContextTheForgeStates(t *testing.T) {
	server := servingEach(t, map[string]string{
		"/repos/acme/payments/pulls/412": `{
			"number": 412,
			"draft": false,
			"base": {"ref": "main"},
			"head": {"sha": "9f1c8d2e4b7a"},
			"user": {"login": "jo", "type": "User"},
			"author_association": "MEMBER",
			"labels": [{"name": "backend"}, {"name": "needs-review"}]
		}`,
		"/repos/acme/payments/pulls/412/reviews": `[]`,
	})
	source := sourceAt(server)

	pullRequest, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if err != nil {
		t.Fatalf("pull request at: %v", err)
	}
	want := domain.PullRequest{
		Number:     412,
		Repository: "acme/payments",
		BaseRef:    "main",
		HeadSHA:    "9f1c8d2e4b7a",
		Author: domain.Author{
			Login:       "jo",
			Association: domain.AuthorAssociationMember,
		},
		Labels:    domain.Declared[string]{"backend", "needs-review"},
		Approvals: domain.Declared[domain.Approval]{},
	}
	if !reflect.DeepEqual(pullRequest, want) {
		t.Fatalf("pull request is %+v, want %+v", pullRequest, want)
	}
}

func TestPullRequestAtRefusesAnAccountTypeItCannotClassify(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412", `{
		"number": 412,
		"user": {"login": "jo", "type": "Golem"}
	}`)
	source := sourceAt(server)

	_, err := source.PullRequestAt(
		context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrUnknownUserType) {
		t.Fatalf("error is %v, want %v", err, github.ErrUnknownUserType)
	}
}

func TestFilesChangedRefusesAnAnswerLargerThanTheResponseBound(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "modified"}
	]`)
	source := github.ChangeSource{
		BaseURL:          server.URL,
		Client:           server.Client(),
		MaxResponseBytes: 8,
	}

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, github.ErrResponseTooLarge) {
		t.Fatalf("error is %v, want %v", err, github.ErrResponseTooLarge)
	}
}

func TestFilesChangedRefusesAForgeThatNeverRunsOutOfPages(t *testing.T) {
	server := forge(t, func(writer http.ResponseWriter, request *http.Request) {
		write(t, writer, pageAsLongAsAsked(t, request))
	})
	source := sourceAt(server)

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
	source := sourceAt(server)

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
	source := sourceAt(server)

	_, err := source.FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

	if !errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchPullRequest)
	}
}

func TestFilesChangedRefusesAStatusTheSchemaDoesNotDeclare(t *testing.T) {
	server := serving(t, "/repos/acme/payments/pulls/412/files", `[
		{"filename": "internal/auth/middleware.go", "status": "unchanged"}
	]`)
	source := sourceAt(server)

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
	source := sourceAt(server)

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

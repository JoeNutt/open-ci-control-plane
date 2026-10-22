// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const contentsCommit = "4b9c4930c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6"

const middlewareSource = "package auth\n\nimport \"net/http\"\n"

type askedForContents struct {
	method        string
	escapedPath   string
	rawQuery      string
	authorization string
}

func recordingContents(t *testing.T, asked *askedForContents, body string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		*asked = askedForContents{
			method:        request.Method,
			escapedPath:   request.URL.EscapedPath(),
			rawQuery:      request.URL.RawQuery,
			authorization: request.Header.Get("Authorization"),
		}
		write(t, writer, body)
	})
}

func neverAsked(t *testing.T) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		t.Errorf("the adapter asked for %s", request.URL.EscapedPath())
		writer.WriteHeader(http.StatusNotFound)
	})
}

func fileAnswer(contents string) string {
	return statedFileAnswer(len(contents), base64.StdEncoding.EncodeToString([]byte(contents)))
}

func statedFileAnswer(size int, encoded string) string {
	return `{"type": "file", "encoding": "base64", "size": ` +
		strconv.Itoa(size) + `, "content": "` + encoded + `"}`
}

func contentsAt(t *testing.T, server *httptest.Server) github.FileContents {
	t.Helper()

	return contentsWith(t, settingsAt(server))
}

func contentsWith(t *testing.T, settings github.Settings) github.FileContents {
	t.Helper()

	contents, err := github.ParseFileContents(settings, domain.DeclaredLimits().MaxSourceBytes)
	if err != nil {
		t.Fatalf("parse file contents: %v", err)
	}
	return contents
}

func fileAt(t *testing.T, repository, path string) domain.FileAtCommit {
	t.Helper()

	commit, err := domain.ParseCommitSHA(contentsCommit)
	if err != nil {
		t.Fatalf("parse commit: %v", err)
	}
	return domain.FileAtCommit{Repository: repository, Commit: commit, Path: path}
}

func TestContentsAtReadsTheFileAtTheCommitAsTheInstallation(t *testing.T) {
	var asked askedForContents
	server := recordingContents(t, &asked, fileAnswer(middlewareSource))

	read, err := contentsAt(t, server).ContentsAt(
		context.Background(), fileAt(t, "acme/payments", "internal/auth/middleware.go"))

	if err != nil {
		t.Fatalf("contents at: %v", err)
	}
	if string(read) != middlewareSource {
		t.Errorf("read %q, want %q", read, middlewareSource)
	}
	want := askedForContents{
		method:        http.MethodGet,
		escapedPath:   "/repos/acme/payments/contents/internal/auth/middleware.go",
		rawQuery:      "ref=" + contentsCommit,
		authorization: "Bearer " + forgeToken,
	}
	if asked != want {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}

func TestContentsAtRefusesAPathThatCouldResolveOutsideItsSegmentsWithoutAsking(t *testing.T) {
	server := neverAsked(t)
	hostile := []string{
		"../secrets.go", "internal/../../secrets.go", "internal/..", "./main.go",
		"internal/./main.go", "internal//main.go", "/etc/passwd", "internal/", "",
	}

	for _, path := range hostile {
		_, err := contentsAt(t, server).ContentsAt(context.Background(), fileAt(t, "acme/payments", path))

		if !errors.Is(err, domain.ErrPathOutsideRepository) {
			t.Errorf("%q: error is %v, want %v", path, err, domain.ErrPathOutsideRepository)
		}
	}
}

func TestContentsAtRefusesARepositoryThatIsNotAnOwnerAndNameWithoutAsking(t *testing.T) {
	server := neverAsked(t)
	hostile := []string{
		"acme/payments/../secrets", "acme/pay?ref=main", "acme/pay#ments", "acme", "../acme/payments",
	}

	for _, repository := range hostile {
		_, err := contentsAt(t, server).ContentsAt(context.Background(), fileAt(t, repository, "main.go"))

		if !errors.Is(err, domain.ErrMalformedRepository) {
			t.Errorf("%q: error is %v, want %v", repository, err, domain.ErrMalformedRepository)
		}
	}
}

func TestContentsAtReportsAFileTheForgeDoesNotHoldAtThatCommit(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)

	_, err := contentsAt(t, server).ContentsAt(
		context.Background(), fileAt(t, "acme/payments", "internal/auth/middleware.go"))

	if !errors.Is(err, domain.ErrNoSuchFile) {
		t.Fatalf("error is %v, want %v", err, domain.ErrNoSuchFile)
	}
	if errors.Is(err, domain.ErrNoSuchPullRequest) {
		t.Errorf("error %v claims the pull request is missing", err)
	}
}

func TestContentsAtNamesTheCommitOnceWhenTheForgeCannotBeReached(t *testing.T) {
	server := answering(t, http.StatusOK, fileAnswer(middlewareSource))
	contents := contentsAt(t, server)
	server.Close()

	_, err := contents.ContentsAt(context.Background(), fileAt(t, "acme/payments", "internal/auth/middleware.go"))

	if err == nil {
		t.Fatal("read from a closed forge without error")
	}
	if named := strings.Count(err.Error(), contentsCommit); named != 1 {
		t.Errorf("error %q names the commit %d times, want once", err, named)
	}
}

func TestContentsAtRefusesAnAnswerThatIsNotAFile(t *testing.T) {
	answers := map[string]string{
		"directory": `[{"type": "file", "name": "middleware.go", "path": "internal/auth/middleware.go"}]`,
		"symlink":   `{"type": "symlink", "target": "../../secrets.go", "size": 16}`,
		"submodule": `{"type": "submodule", "submodule_git_url": "https://example.test/vendor.git"}`,
	}

	for kind, answer := range answers {
		server := answering(t, http.StatusOK, answer)

		_, err := contentsAt(t, server).ContentsAt(
			context.Background(), fileAt(t, "acme/payments", "internal/auth"))

		if !errors.Is(err, github.ErrNotAFile) {
			t.Errorf("%s: error is %v, want %v", kind, err, github.ErrNotAFile)
		}
	}
}

func TestContentsAtRefusesAFileTheForgeServesWithoutItsContent(t *testing.T) {
	server := answering(t, http.StatusOK,
		`{"type": "file", "encoding": "none", "size": 1000500, "content": ""}`)

	read, err := contentsAt(t, server).ContentsAt(
		context.Background(), fileAt(t, "acme/payments", "internal/generated.go"))

	if !errors.Is(err, github.ErrContentsNotServed) {
		t.Fatalf("read %d bytes with error %v, want %v", len(read), err, github.ErrContentsNotServed)
	}
}

func TestContentsAtReadsAFileOfExactlyTheBoundAndRefusesOneStatedOverItBeforeDecoding(t *testing.T) {
	limit := domain.DeclaredLimits().MaxSourceBytes
	atLimit := answering(t, http.StatusOK, fileAnswer(strings.Repeat("a", limit)))
	settings := settingsAt(atLimit)
	settings.MaxResponseBytes = 2 * limit
	overLimit := answering(t, http.StatusOK, statedFileAnswer(limit+1, "!"))
	file := fileAt(t, "acme/payments", "internal/generated.go")

	read, err := contentsWith(t, settings).ContentsAt(context.Background(), file)
	if err != nil || len(read) != limit {
		t.Fatalf("at the bound: read %d bytes with error %v, want %d bytes", len(read), err, limit)
	}
	_, err = contentsAt(t, overLimit).ContentsAt(context.Background(), file)
	if !errors.Is(err, github.ErrFileTooLarge) {
		t.Fatalf("one over the bound: error is %v, want %v", err, github.ErrFileTooLarge)
	}
}

func TestContentsAtSendsQuerySyntaxInAPathAsPartOfThePath(t *testing.T) {
	var asked askedForContents
	server := recordingContents(t, &asked, fileAnswer(middlewareSource))

	_, err := contentsAt(t, server).ContentsAt(
		context.Background(), fileAt(t, "acme/payments", "internal/a?ref=main#b%2e.go"))

	if err != nil {
		t.Fatalf("contents at: %v", err)
	}
	wantPath := "/repos/acme/payments/contents/internal/a%3Fref=main%23b%252e.go"
	if asked.escapedPath != wantPath || asked.rawQuery != "ref="+contentsCommit {
		t.Errorf("the adapter asked for path %q query %q, want path %q query %q",
			asked.escapedPath, asked.rawQuery, wantPath, "ref="+contentsCommit)
	}
}

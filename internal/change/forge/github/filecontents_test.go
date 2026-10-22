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

const middlewarePath = "internal/auth/middleware.go"

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
	governed, err := domain.ParseGovernedRepository(repository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	file, err := domain.ParseFileAtCommit(governed, commit, path)
	if err != nil {
		t.Fatalf("parse file at commit: %v", err)
	}
	return file
}

func TestParseFileContentsRefusesAFileByteBoundThatIsNotPositive(t *testing.T) {
	for _, bound := range []int{0, -1} {
		_, err := github.ParseFileContents(wellFormedSettings(), bound)

		if !errors.Is(err, github.ErrMalformedFileBound) {
			t.Errorf("bound %d: error is %v, want %v", bound, err, github.ErrMalformedFileBound)
		}
	}
}

func TestContentsAtReadsTheFileAtTheCommitAsTheInstallation(t *testing.T) {
	var asked askedForContents
	server := recordingContents(t, &asked, fileAnswer(middlewareSource))

	read, err := contentsAt(t, server).ContentsAt(
		context.Background(), fileAt(t, "acme/payments", middlewarePath))

	if err != nil {
		t.Fatalf("contents at: %v", err)
	}
	if string(read) != middlewareSource {
		t.Errorf("read %q, want %q", read, middlewareSource)
	}
	want := askedForContents{
		method:        http.MethodGet,
		escapedPath:   "/repos/acme/payments/contents/" + middlewarePath,
		rawQuery:      "ref=" + contentsCommit,
		authorization: "Bearer " + forgeToken,
	}
	if asked != want {
		t.Errorf("the adapter asked %+v, want %+v", asked, want)
	}
}

func TestContentsAtReportsAFileTheForgeDoesNotHoldAtThatCommit(t *testing.T) {
	server := answering(t, http.StatusNotFound, `{"message": "Not Found"}`)

	_, err := contentsAt(t, server).ContentsAt(
		context.Background(), fileAt(t, "acme/payments", middlewarePath))

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
	if !errors.Is(err, domain.ErrSourceTooLarge) {
		t.Fatalf("one over the bound: error is %v, want %v", err, domain.ErrSourceTooLarge)
	}
}

func TestContentsAtRefusesContentLongerOrShorterThanTheSizeTheForgeStates(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(middlewareSource))

	for _, stated := range []int{len(middlewareSource) - 1, len(middlewareSource) + 1} {
		server := answering(t, http.StatusOK, statedFileAnswer(stated, encoded))

		_, err := contentsAt(t, server).ContentsAt(
			context.Background(), fileAt(t, "acme/payments", middlewarePath))

		if !errors.Is(err, github.ErrContentsSizeMismatch) {
			t.Errorf("stated %d: error is %v, want %v", stated, err, github.ErrContentsSizeMismatch)
		}
	}
}

func TestContentsAtNamesTheFileInEveryRefusalButNeverTheCredentialOrTheContent(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(middlewareSource))
	unencoded := "package auth"
	echoing := `{"message": "` + forgeToken + `", "content": "` + encoded + `"}`
	refusals := map[string]*httptest.Server{
		"server error":  answering(t, http.StatusInternalServerError, echoing),
		"not found":     answering(t, http.StatusNotFound, echoing),
		"not a file":    answering(t, http.StatusOK, `{"type": "symlink", "content": "`+encoded+`"}`),
		"size mismatch": answering(t, http.StatusOK, statedFileAnswer(1, encoded)),
		"not base64":    answering(t, http.StatusOK, statedFileAnswer(len(unencoded), unencoded)),
	}

	for name, server := range refusals {
		_, err := contentsAt(t, server).ContentsAt(
			context.Background(), fileAt(t, "acme/payments", middlewarePath))

		if err == nil {
			t.Fatalf("%s: read without error", name)
		}
		if !strings.Contains(err.Error(), middlewarePath) {
			t.Errorf("%s: error %q does not name the file", name, err)
		}
		mustRevealNoneOf(t, name, err.Error(), []string{forgeToken, middlewareSource, encoded, unencoded})
	}
}

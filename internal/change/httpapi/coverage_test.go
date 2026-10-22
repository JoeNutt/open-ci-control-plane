// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	uploadedSuite = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	coveredFile   = "internal/change/domain/effect.go"
	uploadedLCOV  = "SF:" + coveredFile + "\nDA:1,1\nend_of_record\n"
	actionsToken  = "eyJhbGciOiJSUzI1NiJ9.the-workflows-token.c2lnbmF0dXJl"
	coveragePath  = "/v1/repositories/" + gatedRepository + "/coverage/" + uploadedSuite
)

func TestAValidUploadIsAnswered201WithTheSuiteAndItsFileEntryCountAndIsServed(t *testing.T) {
	uploading := newCoverageUploads(t)

	answered := uploading.answerTo(t, coverageUpload(t, "lcov", uploadedLCOV))

	if answered.Code != http.StatusCreated {
		t.Fatalf("status is %d, want %d: %s", answered.Code, http.StatusCreated, answered.Body.String())
	}
	var accepted struct {
		Suite          string `json:"suite"`
		FileEntryCount int    `json:"fileEntryCount"`
	}
	if err := json.NewDecoder(answered.Body).Decode(&accepted); err != nil {
		t.Fatalf("decode the answer: %v", err)
	}
	if accepted.Suite != uploadedSuite || accepted.FileEntryCount != 1 {
		t.Errorf("answered %+v, want suite %s with 1 file entry", accepted, uploadedSuite)
	}
	if !uploading.isStored(t) {
		t.Error("the accepted map is not served by CoverageMaps")
	}
}

func TestAnUploadDeclaringMoreThanTheCompressedBoundIsAnswered413NamingItUnread(t *testing.T) {
	uploading := newCoverageUploads(t)
	unread := &unreadUpload{}
	upload := coverageUpload(t, "lcov", uploadedLCOV)
	upload.Body, upload.ContentLength = unread, domain.DeclaredCoverageLimits().MaxCompressedBytes+1

	answered := uploading.answerTo(t, upload)

	if answered.Code != http.StatusRequestEntityTooLarge || !strings.Contains(answered.Body.String(), "67108864 compressed bytes") {
		t.Errorf("answered %d %s, want %d naming the compressed bound", answered.Code, answered.Body.String(), http.StatusRequestEntityTooLarge)
	}
	if unread.isRead {
		t.Error("the body of an upload declared over the bound was read")
	}
}

type unreadUpload struct{ isRead bool }

func (upload *unreadUpload) Read([]byte) (int, error) {
	upload.isRead = true
	return 0, errors.New("the upload was read")
}

func (*unreadUpload) Close() error { return nil }

type coverageUploads struct {
	store   *memory.CoverageMaps
	callers domain.CallerRepositories
	logged  *bytes.Buffer
}

func newCoverageUploads(t *testing.T) *coverageUploads {
	t.Helper()
	return &coverageUploads{
		store:   memory.NewCoverageMaps(domain.DeclaredCoverageLimits()),
		callers: tokensNaming{gatedRepository},
		logged:  &bytes.Buffer{},
	}
}

func (uploading *coverageUploads) answerTo(t *testing.T, upload *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rate, err := application.ParseUploadRate(domain.DeclaredCoverageLimits(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	settings := listenerSettings(slog.New(slog.NewJSONHandler(uploading.logged, nil)), &recordingGate{})
	settings.Coverage = application.ReplaceCoverage{
		Maps:    uploading.store,
		Paths:   repositoryHolding{coveredFile},
		Callers: uploading.callers,
		Rate:    rate,
	}
	settings.Artifacts = lcov.Artifacts{}
	answered := httptest.NewRecorder()
	parsedRoutes(t, settings).ServeHTTP(answered, upload)
	return answered
}

func (uploading *coverageUploads) isStored(t *testing.T) bool {
	t.Helper()
	repository, err := domain.ParseGovernedRepository(gatedRepository)
	if err != nil {
		t.Fatal(err)
	}
	suite, err := domain.ParseCoverageSuite(uploadedSuite)
	if err != nil {
		t.Fatal(err)
	}
	_, err = uploading.store.SuiteCoverageOf(t.Context(), repository, suite)
	if err != nil && !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("read the store: %v", err)
	}
	return err == nil
}

func coverageUpload(t *testing.T, format, artifact string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("format", format); err != nil {
		t.Fatal(err)
	}
	part, err := form.CreateFormFile("artifact", "coverage.lcov")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, artifact); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	upload := httptest.NewRequest(http.MethodPut, coveragePath, &body)
	upload.Header.Set("Content-Type", form.FormDataContentType())
	upload.Header.Set("Authorization", "Bearer "+actionsToken)
	return upload
}

type repositoryHolding []string

func (held repositoryHolding) KnownPathsOf(context.Context, domain.GovernedRepository) (domain.KnownPaths, error) {
	return domain.ParseKnownPaths(held)
}

type tokensNaming struct{ repository string }

func (naming tokensNaming) RepositoryOf(context.Context, domain.IdentityToken) (domain.GovernedRepository, error) {
	return domain.ParseGovernedRepository(naming.repository)
}

var errCoverageUnexercised = errors.New("this test uploads no coverage")

type unexercisedCoverage struct{}

func (unexercisedCoverage) Execute(context.Context, application.CoverageUpload) (application.ReplacedCoverage, error) {
	return application.ReplacedCoverage{}, errCoverageUnexercised
}

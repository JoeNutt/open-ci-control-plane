// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	uploadedSuite       = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	coveredFile         = "internal/change/domain/effect.go"
	uploadedTestSection = "TN:TestEffect\n"
	uploadedLCOV        = uploadedTestSection + "SF:" + coveredFile + "\nDA:1,1\nend_of_record\n"
	actionsToken        = "eyJhbGciOiJSUzI1NiJ9.the-workflows-token.c2lnbmF0dXJl"
	coveragePath        = "/v1/repositories/" + gatedRepository + "/coverage/" + uploadedSuite
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

	compressedLimit, isStated := domain.DeclaredCoverageLimits().LimitOf(domain.ErrCompressedArtifactTooLarge)
	if !isStated {
		t.Fatal("the domain states no compressed bound")
	}

	answered := uploading.answerTo(t, upload)

	if answered.Code != http.StatusRequestEntityTooLarge || !strings.Contains(answered.Body.String(), compressedLimit) {
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

func TestAnUploadWhoseTokenTheVerifierRefusesIsAnswered401WithoutItsBodyRead(t *testing.T) {
	uploading := newCoverageUploads(t)
	uploading.callers = unverifiedTokens{}
	unread := &unreadUpload{}
	upload := coverageUpload(t, "lcov", uploadedLCOV)
	upload.Body = unread

	answered := uploading.answerTo(t, upload)

	if answered.Code != http.StatusUnauthorized || unread.isRead || uploading.isStored(t) {
		t.Errorf("answered %d, read %t, stored %t: want %d, unread, nothing stored",
			answered.Code, unread.isRead, uploading.isStored(t), http.StatusUnauthorized)
	}
}

func TestAnUploadWithoutOneBearerTokenIsAnswered401(t *testing.T) {
	for form, authorization := range map[string]string{
		"no header": "", "another scheme": "Basic " + actionsToken, "an empty token": "Bearer ",
	} {
		t.Run(form, func(t *testing.T) {
			uploading := newCoverageUploads(t)
			upload := coverageUpload(t, "lcov", uploadedLCOV)
			upload.Header.Set("Authorization", authorization)

			answered := uploading.answerTo(t, upload)

			if answered.Code != http.StatusUnauthorized || uploading.isStored(t) {
				t.Errorf("answered %d, stored %t: want %d, nothing stored", answered.Code, uploading.isStored(t), http.StatusUnauthorized)
			}
		})
	}
}

func TestAnUploadWithATokenForAnotherRepositoryIsAnswered403LoggedAndStoresNothing(t *testing.T) {
	uploading := newCoverageUploads(t)
	uploading.callers = tokensNaming{"JoeNutt/open-ci-policy"}

	answered := uploading.answerTo(t, coverageUpload(t, "lcov", uploadedLCOV))

	if answered.Code != http.StatusForbidden || uploading.isStored(t) {
		t.Errorf("answered %d, stored %t: want %d, nothing stored", answered.Code, uploading.isStored(t), http.StatusForbidden)
	}
	if !strings.Contains(uploading.logged.String(), "foreign_repository") {
		t.Errorf("the 403 was not logged: %q", uploading.logged.String())
	}
}

func TestAnUploadDeclaringNoLengthIsAnswered413Unread(t *testing.T) {
	uploading := newCoverageUploads(t)
	unread := &unreadUpload{}
	upload := coverageUpload(t, "lcov", uploadedLCOV)
	upload.Body, upload.ContentLength, upload.TransferEncoding = unread, -1, []string{"chunked"}

	answered := uploading.answerTo(t, upload)

	if answered.Code != http.StatusRequestEntityTooLarge || unread.isRead {
		t.Errorf("answered %d, read %t: want %d, unread", answered.Code, unread.isRead, http.StatusRequestEntityTooLarge)
	}
}

func TestAnUploadThatIsNotAMultipartFormIsAnswered400(t *testing.T) {
	uploading := newCoverageUploads(t)
	upload := coverageUpload(t, "lcov", uploadedLCOV)
	upload.Header.Set("Content-Type", "application/octet-stream")

	answered := uploading.answerTo(t, upload)

	if answered.Code != http.StatusBadRequest || !strings.Contains(answered.Body.String(), `"field":"Content-Type"`) {
		t.Errorf("answered %d %s, want %d naming Content-Type", answered.Code, answered.Body.String(), http.StatusBadRequest)
	}
}

func TestAnUploadStatingAFormatOtherThanLCOVIsAnswered400NamingFormat(t *testing.T) {
	uploading := newCoverageUploads(t)

	answered := uploading.answerTo(t, coverageUpload(t, "cobertura", uploadedLCOV))

	if answered.Code != http.StatusBadRequest || !strings.Contains(answered.Body.String(), `"field":"format"`) || uploading.isStored(t) {
		t.Errorf("answered %d %s, want %d naming format", answered.Code, answered.Body.String(), http.StatusBadRequest)
	}
}

func TestAnUploadWhoseArtifactIsNotLCOVIsAnswered400NamingArtifactAndKeepsThePreviousMap(t *testing.T) {
	uploading := newCoverageUploads(t)
	if answered := uploading.answerTo(t, coverageUpload(t, "lcov", uploadedLCOV)); answered.Code != http.StatusCreated {
		t.Fatalf("the first upload was answered %d", answered.Code)
	}

	answered := uploading.answerTo(t, coverageUpload(t, "lcov", uploadedTestSection+"SF:"+coveredFile+"\nDA:1,1\n"))

	if answered.Code != http.StatusBadRequest || !strings.Contains(answered.Body.String(), `"field":"artifact"`) {
		t.Errorf("answered %d %s, want %d naming artifact", answered.Code, answered.Body.String(), http.StatusBadRequest)
	}
	if !uploading.isStored(t) {
		t.Error("the previous map stopped serving")
	}
}

func TestAnUploadNamingAnAbsoluteOrUnknownPathIsAnswered422AndStoresNothing(t *testing.T) {
	for _, path := range []string{"/home/runner/work/effect.go", "internal/change/domain/missing.go"} {
		uploading := newCoverageUploads(t)

		answered := uploading.answerTo(t, coverageUpload(t, "lcov", uploadedTestSection+"SF:"+path+"\nDA:1,1\nend_of_record\n"))

		if answered.Code != http.StatusUnprocessableEntity || uploading.isStored(t) {
			t.Errorf("%s: answered %d, stored %t: want %d, nothing stored", path, answered.Code, uploading.isStored(t), http.StatusUnprocessableEntity)
		}
	}
}

func TestAnUploadOverTheRateIsAnswered429WithRetryAfterInSeconds(t *testing.T) {
	uploading := newCoverageUploads(t)
	routes := uploading.routes(t)
	for range domain.DeclaredCoverageLimits().MaxUploadsPerMinute {
		routes.ServeHTTP(httptest.NewRecorder(), coverageUpload(t, "lcov", uploadedLCOV))
	}
	answered := httptest.NewRecorder()

	routes.ServeHTTP(answered, coverageUpload(t, "lcov", uploadedLCOV))

	if answered.Code != http.StatusTooManyRequests || answered.Header().Get("Retry-After") == "" {
		t.Errorf("answered %d with Retry-After %q, want %d with seconds", answered.Code, answered.Header().Get("Retry-After"), http.StatusTooManyRequests)
	}
}

func TestAnUploadToAPathC12DoesNotAdmitIsAnswered400NamingTheSegmentNotRedirected(t *testing.T) {
	for path, field := range map[string]string{
		"/v1/repositories/Joe_Nutt/open-ci-control-plane/coverage/" + uploadedSuite:      "owner/repo",
		"/v1/repositories/JoeNutt/open%20ci/coverage/" + uploadedSuite:                   "owner/repo",
		"/v1/repositories/" + gatedRepository + "/coverage/github.com/JoeNutt/bad+suite": "suite",
		"/v1/repositories/" + gatedRepository + "/coverage/github.com/JoeNutt/../domain": "path",
		"/v1/repositories/" + gatedRepository + "/coverage/" + uploadedSuite + "/":       "path",
	} {
		t.Run(path, func(t *testing.T) {
			uploading := newCoverageUploads(t)
			upload := coverageUpload(t, "lcov", uploadedLCOV)
			upload.URL.Path, upload.URL.RawPath = path, ""

			answered := uploading.answerTo(t, upload)

			if answered.Code != http.StatusBadRequest || !strings.Contains(answered.Body.String(), `"field":"`+field+`"`) || uploading.isStored(t) {
				t.Errorf("answered %d %s, want %d naming %s", answered.Code, answered.Body.String(), http.StatusBadRequest, field)
			}
		})
	}
}

func TestAnUploadTheReplacementFailsForAnUnclassifiedReasonIsAnswered500AndLoggedWithItsCause(t *testing.T) {
	var logged bytes.Buffer
	settings := listenerSettings(slog.New(slog.NewJSONHandler(&logged, nil)), &recordingGate{})
	answered := httptest.NewRecorder()

	parsedRoutes(t, settings).ServeHTTP(answered, coverageUpload(t, "lcov", uploadedLCOV))

	if answered.Code != http.StatusInternalServerError || !strings.Contains(logged.String(), errCoverageUnexercised.Error()) {
		t.Errorf("answered %d, logged %q: want %d with the cause logged", answered.Code, logged.String(), http.StatusInternalServerError)
	}
}

func TestEveryCoverageRefusalIsOneStructuredLineWithItsReasonAndNoToken(t *testing.T) {
	for form, refusing := range map[string]func(*coverageUploads, *http.Request){
		"foreign repository": func(uploading *coverageUploads, _ *http.Request) {
			uploading.callers = tokensNaming{"JoeNutt/open-ci-policy"}
		},
		"unverified token": func(uploading *coverageUploads, _ *http.Request) { uploading.callers = unverifiedTokens{} },
		"unknown format":   func(_ *coverageUploads, upload *http.Request) { upload.Header.Set("Content-Type", "text/plain") },
		"malformed suite":  func(_ *coverageUploads, upload *http.Request) { upload.URL.Path += "+" },
		"over the bound": func(_ *coverageUploads, upload *http.Request) {
			upload.ContentLength = domain.DeclaredCoverageLimits().MaxCompressedBytes + 1
		},
	} {
		t.Run(form, func(t *testing.T) {
			uploading := newCoverageUploads(t)
			upload := coverageUpload(t, "lcov", uploadedLCOV)
			refusing(uploading, upload)

			uploading.answerTo(t, upload)

			lines := strings.Split(strings.TrimSpace(uploading.logged.String()), "\n")
			var line map[string]any
			if len(lines) != 1 || json.Unmarshal([]byte(lines[0]), &line) != nil || line["msg"] != "coverage refused" || line["reason"] == nil {
				t.Errorf("logged %q, want one structured coverage refusal with its reason", uploading.logged.String())
			}
			if strings.Contains(uploading.logged.String(), actionsToken) {
				t.Errorf("a log line carries the token: %q", uploading.logged.String())
			}
		})
	}
}

type unverifiedTokens struct{}

func (unverifiedTokens) RepositoryOf(context.Context, domain.IdentityToken) (domain.GovernedRepository, error) {
	return domain.GovernedRepository{}, fmt.Errorf("verify the token: %w", domain.ErrUnverifiedIdentityToken)
}

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
	answered := httptest.NewRecorder()
	uploading.routes(t).ServeHTTP(answered, upload)
	return answered
}

func (uploading *coverageUploads) routes(t *testing.T) http.Handler {
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
	return parsedRoutes(t, settings)
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

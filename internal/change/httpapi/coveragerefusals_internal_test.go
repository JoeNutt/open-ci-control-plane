// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAnUploadFromAnotherRepositoryIsAnswered403(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("authorise: %w", application.ErrForeignRepository))

	if refused.answer.status() != http.StatusForbidden {
		t.Errorf("answered %#v, want %d", refused.answer, http.StatusForbidden)
	}
}

func TestAFormThatCannotBeReadAsLCOVIsAnswered400NamingThePart(t *testing.T) {
	for part, failure := range map[string]error{
		formatPart:   errUnknownFormat,
		artifactPart: fmt.Errorf("read the artifact: line 3: %w", domain.ErrMalformedCoverageReport),
	} {
		refused := coverageRefusalOf(failure)

		if refused.answer.status() != http.StatusBadRequest || refused.answer.(fieldProblem).Field != part {
			t.Errorf("%v: answered %#v, want %d naming %s", failure, refused.answer, http.StatusBadRequest, part)
		}
	}
}

func TestAFormMissingAPartIsAnswered400NamingIt(t *testing.T) {
	for part, missing := range map[string]error{formatPart: errNoFormatPart, artifactPart: errNoArtifactPart} {
		refused := coverageRefusalOf(fmt.Errorf("read the artifact: %w", missing))

		if refused.answer.status() != http.StatusBadRequest || refused.answer.(fieldProblem).Field != part {
			t.Errorf("answered %#v, want %d naming %s", refused.answer, http.StatusBadRequest, part)
		}
	}
}

func TestAnArtifactOverAnNFR5NFR13OrNFR14BoundIsAnswered413NamingThatBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	for breach, bound := range map[error]string{
		domain.ErrCompressedArtifactTooLarge:   fmt.Sprint(limits.MaxCompressedBytes),
		domain.ErrDecompressedArtifactTooLarge: fmt.Sprint(limits.MaxDecompressedBytes),
		domain.ErrTooManyFileEntries:           fmt.Sprint(limits.MaxFileEntries),
		domain.ErrTooManySuites:                fmt.Sprint(limits.MaxSuitesPerRepository),
		domain.ErrTooManyTestSections:          fmt.Sprint(limits.MaxTestSections),
		domain.ErrTooManyLineRecords:           fmt.Sprint(limits.MaxLineRecords),
		domain.ErrTooManyIndexedTests:          fmt.Sprint(limits.MaxIndexedTests),
		domain.ErrTooManyIndexEdges:            fmt.Sprint(limits.MaxIndexEdges),
	} {
		refused := coverageRefusalOf(fmt.Errorf("refuse coverage: %w", breach))

		problem, isLimit := refused.answer.(limitProblem)
		if !isLimit || problem.status() != http.StatusRequestEntityTooLarge || !strings.HasPrefix(problem.Limit, bound+" ") {
			t.Errorf("%v: answered %#v, want %d naming %s", breach, refused.answer, http.StatusRequestEntityTooLarge, bound)
		}
	}
}

func TestAnUploadRateRefusalIsAnswered429WithItsRetryAfterInSeconds(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("admit: %w", application.UploadRateRefusal{RetryAfter: 7 * time.Second}))
	answered := httptest.NewRecorder()

	answers{log: slog.New(slog.DiscardHandler)}.write(t.Context(), answered, refused.answer)

	if answered.Code != http.StatusTooManyRequests || answered.Header().Get("Retry-After") != "7" {
		t.Errorf("answered %d with Retry-After %q, want %d with 7", answered.Code, answered.Header().Get("Retry-After"), http.StatusTooManyRequests)
	}
}

func TestAnUnverifiedTokenIsAnswered401AsARefusal(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("verify: %w", domain.ErrUnverifiedIdentityToken))

	if refused.answer.status() != http.StatusUnauthorized || refused.level != slog.LevelWarn {
		t.Errorf("answered %#v at %s, want %d at %s", refused.answer, refused.level, http.StatusUnauthorized, slog.LevelWarn)
	}
}

func TestATokenWhoseIssuerCannotBeReadIsAnswered401AndLoggedAsOurOutage(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("verify: %w", domain.ErrIdentityIssuerUnavailable))

	if refused.answer.status() != http.StatusUnauthorized || refused.level != slog.LevelError {
		t.Errorf("answered %#v at %s, want %d at %s", refused.answer, refused.level, http.StatusUnauthorized, slog.LevelError)
	}
}

func TestAFailureC12DoesNotClassifyIsAnswered500AndLoggedAsAnError(t *testing.T) {
	refused := coverageRefusalOf(errors.New("the store is full"))

	if refused.answer.status() != http.StatusInternalServerError || refused.level != slog.LevelError {
		t.Errorf("answered %#v at %s, want %d at %s", refused.answer, refused.level, http.StatusInternalServerError, slog.LevelError)
	}
}

func TestAnArtifactNamingAPathOutsideTheRepositoryIsAnswered422(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("refuse coverage: line 1: %w", domain.ErrPathOutsideRepository))

	if refused.answer.status() != http.StatusUnprocessableEntity {
		t.Errorf("answered %#v, want %d", refused.answer, http.StatusUnprocessableEntity)
	}
}

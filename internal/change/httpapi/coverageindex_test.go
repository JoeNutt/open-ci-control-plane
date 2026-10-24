// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const secondSuitePath = "/v1/repositories/" + gatedRepository + "/coverage/github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"

func TestAnUploadTakingTheIndexOverItsTestBoundIsAnswered413NamingItAndKeepsThePreviousMaps(t *testing.T) {
	uploading := newCoverageUploads(t)
	limits := domain.DeclaredCoverageLimits()
	full := uploading.answerTo(t, coverageUpload(t, "lcov", lcovOfTests(limits.MaxIndexedTests)))
	if full.Code != http.StatusCreated {
		t.Fatalf("an index at the test bound was answered %d: %s", full.Code, full.Body.String())
	}
	oneMore := coverageUpload(t, "lcov", lcovOfTests(1))
	oneMore.URL.Path = secondSuitePath

	answered := uploading.answerTo(t, oneMore)

	testLimit, isStated := limits.LimitOf(domain.ErrTooManyIndexedTests)
	if !isStated || answered.Code != http.StatusRequestEntityTooLarge || !strings.Contains(answered.Body.String(), testLimit) {
		t.Errorf("answered %d %s, want %d naming %q", answered.Code, answered.Body.String(), http.StatusRequestEntityTooLarge, testLimit)
	}
	if !uploading.isStored(t) {
		t.Error("the map held before the refused upload stopped serving")
	}
}

func lcovOfTests(count int) string {
	var artifact strings.Builder
	for index := range count {
		fmt.Fprintf(&artifact, "TN:TestIndexed%d\n%s", index, uploadedRecord)
	}
	return artifact.String()
}

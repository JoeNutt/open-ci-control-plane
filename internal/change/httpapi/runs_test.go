// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
)

const (
	reportedHeadSHA    = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"
	reportedDecisionID = "dec_0123456789abcdef0123456789abcdef"
	runsPath           = "/v1/repositories/" + gatedRepository + "/runs/" + reportedHeadSHA
	reportedSuite      = "github.com/JoeNutt/open-ci-control-plane/internal/change/rules"
	passingReport      = `{"decisionId":"` + reportedDecisionID + `","tests":[` +
		`{"package":"` + reportedSuite + `","name":"TestReadFrom","outcome":"pass","seconds":0.01}]}`
)

func TestAValidReportIsAnswered201AndReachesTheRecordingParsed(t *testing.T) {
	reporting := &runReports{}

	answered := reporting.answerTo(t, runReport(t, passingReport))

	if answered.Code != http.StatusCreated {
		t.Fatalf("status is %d, want %d: %s", answered.Code, http.StatusCreated, answered.Body.String())
	}
	var accepted struct {
		HeadSHA  string `json:"headSha"`
		Recorded int    `json:"recorded"`
	}
	if err := json.NewDecoder(answered.Body).Decode(&accepted); err != nil {
		t.Fatalf("decode the answer: %v", err)
	}
	if accepted.HeadSHA != reportedHeadSHA || accepted.Recorded != 1 {
		t.Errorf("answered %+v, want head %s with 1 test recorded", accepted, reportedHeadSHA)
	}
	if len(reporting.received) != 1 {
		t.Fatalf("the recording received %d reports, want 1", len(reporting.received))
	}
	report := reporting.received[0].Report
	results := report.Results()
	if report.Repository().String() != gatedRepository || report.Head().String() != reportedHeadSHA ||
		report.DecisionID().String() != reportedDecisionID || len(results) != 1 ||
		results[0].Suite.String() != reportedSuite || results[0].Name != "TestReadFrom" {
		t.Errorf("the recording received %+v, want the parsed report", report)
	}
}

type runReports struct {
	refusal  error
	received []application.RunUpload
	logged   bytes.Buffer
}

func (reporting *runReports) Execute(_ context.Context, upload application.RunUpload) error {
	reporting.received = append(reporting.received, upload)
	return reporting.refusal
}

func (reporting *runReports) answerTo(t *testing.T, report *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	settings := listenerSettings(slog.New(slog.NewJSONHandler(&reporting.logged, nil)), &recordingGate{})
	settings.Runs = reporting
	answered := httptest.NewRecorder()
	parsedRoutes(t, settings).ServeHTTP(answered, report)
	return answered
}

func runReport(t *testing.T, body string) *http.Request {
	t.Helper()

	report := httptest.NewRequestWithContext(t.Context(), http.MethodPut, runsPath, strings.NewReader(body))
	report.Header.Set("Authorization", "Bearer "+actionsToken)
	report.Header.Set("Content-Type", "application/json")
	return report
}

func TestAReportToAMalformedPathIsAnswered400AndReachesNoRecording(t *testing.T) {
	for name, path := range map[string]string{
		"a head that is not a commit": "/v1/repositories/" + gatedRepository + "/runs/not-a-sha",
		"an uppercase head":           "/v1/repositories/" + gatedRepository + "/runs/" + strings.ToUpper(reportedHeadSHA),
		"an owner outside its set":    "/v1/repositories/Joe%20Nutt/open-ci-control-plane/runs/" + reportedHeadSHA,
	} {
		t.Run(name, func(t *testing.T) {
			reporting := &runReports{}
			report := runReport(t, passingReport)
			report.URL = mustParseURL(t, path)

			answered := reporting.answerTo(t, report)

			if answered.Code != http.StatusBadRequest || len(reporting.received) != 0 {
				t.Errorf("answered %d, recorded %d: want %d, nothing recorded", answered.Code, len(reporting.received), http.StatusBadRequest)
			}
		})
	}
}

func mustParseURL(t *testing.T, path string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(path)
	if err != nil {
		t.Fatalf("parse %q: %v", path, err)
	}
	return parsed
}

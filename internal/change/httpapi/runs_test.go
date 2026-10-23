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
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
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

func TestAReportDeclaringMoreThanTheByteBoundIsAnswered413NamingItUnread(t *testing.T) {
	reporting := &runReports{}
	unread := &unreadUpload{}
	report := runReport(t, passingReport)
	report.Body, report.ContentLength = unread, domain.MaxRunReportBytes+1

	answered := reporting.answerTo(t, report)

	if answered.Code != http.StatusRequestEntityTooLarge || !strings.Contains(answered.Body.String(), "1048576 bytes") {
		t.Errorf("answered %d %s, want %d naming the byte bound", answered.Code, answered.Body.String(), http.StatusRequestEntityTooLarge)
	}
	if unread.isRead || len(reporting.received) != 0 {
		t.Errorf("read %t, recorded %d: want the body unread and nothing recorded", unread.isRead, len(reporting.received))
	}
}

func TestAReportWhoseBodyRunsPastTheByteBoundIsAnswered413AndRecordsNothing(t *testing.T) {
	reporting := &runReports{}
	padded := `{"padding":"` + strings.Repeat("x", int(domain.MaxRunReportBytes)) + `",` + passingReport[1:]
	report := runReport(t, padded)
	report.ContentLength = -1

	answered := reporting.answerTo(t, report)

	if answered.Code != http.StatusRequestEntityTooLarge || len(reporting.received) != 0 {
		t.Errorf("answered %d, recorded %d: want %d, nothing recorded", answered.Code, len(reporting.received), http.StatusRequestEntityTooLarge)
	}
}

func TestAReportWithoutOneBearerTokenIsAnswered401Unread(t *testing.T) {
	for form, authorization := range map[string]string{
		"no header": "", "another scheme": "Basic " + actionsToken, "an empty token": "Bearer ",
	} {
		t.Run(form, func(t *testing.T) {
			reporting := &runReports{}
			unread := &unreadUpload{}
			report := runReport(t, passingReport)
			report.Header.Set("Authorization", authorization)
			report.Body = unread

			answered := reporting.answerTo(t, report)

			if answered.Code != http.StatusUnauthorized || unread.isRead || len(reporting.received) != 0 {
				t.Errorf("answered %d, read %t, recorded %d: want %d, unread, nothing recorded",
					answered.Code, unread.isRead, len(reporting.received), http.StatusUnauthorized)
			}
		})
	}
}

func TestAReportThatIsNotTheC13ShapeIsAnswered400AndRecordsNothing(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":                `decisionId=` + reportedDecisionID,
		"an array":                `[` + passingReport + `]`,
		"two objects":             passingReport + passingReport,
		"tests that are a string": `{"decisionId":"` + reportedDecisionID + `","tests":"TestReadFrom"}`,
	} {
		t.Run(name, func(t *testing.T) {
			reporting := &runReports{}

			answered := reporting.answerTo(t, runReport(t, body))

			if answered.Code != http.StatusBadRequest || len(reporting.received) != 0 {
				t.Errorf("answered %d, recorded %d: want %d, nothing recorded", answered.Code, len(reporting.received), http.StatusBadRequest)
			}
		})
	}
}

func TestAReportWhoseContentDoesNotParseIsAnswered400AndRecordsNothing(t *testing.T) {
	for name, body := range map[string]string{
		"a malformed decision":      strings.Replace(passingReport, reportedDecisionID, "dec_not-a-decision", 1),
		"an unknown outcome":        strings.Replace(passingReport, `"pass"`, `"passed"`, 1),
		"a subtest's name":          strings.Replace(passingReport, `"TestReadFrom"`, `"TestReadFrom/aliases"`, 1),
		"a package outside its set": strings.Replace(passingReport, reportedSuite, "../rules", 1),
	} {
		t.Run(name, func(t *testing.T) {
			reporting := &runReports{}

			answered := reporting.answerTo(t, runReport(t, body))

			if answered.Code != http.StatusBadRequest || len(reporting.received) != 0 {
				t.Errorf("answered %d, recorded %d: want %d, nothing recorded", answered.Code, len(reporting.received), http.StatusBadRequest)
			}
		})
	}
}

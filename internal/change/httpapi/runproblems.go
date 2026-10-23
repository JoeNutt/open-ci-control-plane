// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	oversizedReportCode     = "report_too_large"
	oversizedReportMessage  = "the report is longer than the byte bound; nothing was recorded"
	malformedReportCode     = "malformed_report"
	malformedReportMessage  = "the body is not one C-13 report object"
	unparseableFieldCode    = "unparseable_field"
	unparseableFieldMessage = "the field holds a value outside the set C-13 accepts"
	decisionIDField         = "decisionId"
	testsField              = "tests"
	tooManyTestsCode        = "too_many_tests"
	tooManyTestsMessage     = "the report names more tests than NFR-18 allows; nothing was recorded"
	unplannedRunCode        = "no_plan"
	unplannedRunMessage     = "no plan was dispatched for this head under this decision"
	reportedRunCode         = "already_reported"
	reportedRunMessage      = "this head's run was already reported; nothing was changed"
	failedRunCode           = "run_failed"
	failedRunMessage        = "the run could not be recorded"
)

var (
	reportBytesLimit = fmt.Sprintf("%d bytes", domain.MaxRunReportBytes)
	reportTestsLimit = fmt.Sprintf("%d tests", domain.MaxReportedTests)
)

func oversizedReportProblem() refusal {
	return limitProblem{Code: oversizedReportCode, Message: oversizedReportMessage, Limit: reportBytesLimit}
}

func malformedReportProblem() refusal {
	return fieldProblem{Code: malformedReportCode, Message: malformedReportMessage, Field: payloadField}
}

func unparseableReportFieldProblem(field string) refusal {
	return fieldProblem{Code: unparseableFieldCode, Message: unparseableFieldMessage, Field: field}
}

func tooManyTestsProblem() refusal {
	return limitProblem{Code: tooManyTestsCode, Message: tooManyTestsMessage, Limit: reportTestsLimit}
}

func reportLengthRefusalOf(declared int64) refusal {
	if declared > domain.MaxRunReportBytes {
		return oversizedReportProblem()
	}
	return nil
}

type unplannedProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (unplannedProblem) status() int { return http.StatusNotFound }

func (problem unplannedProblem) reason() string { return problem.Code }

type reportedProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (reportedProblem) status() int { return http.StatusConflict }

func (problem reportedProblem) reason() string { return problem.Code }

func unplannedRunProblem() refusal {
	return unplannedProblem{Code: unplannedRunCode, Message: unplannedRunMessage}
}

func reportedRunProblem() refusal {
	return reportedProblem{Code: reportedRunCode, Message: reportedRunMessage}
}

func failedRunProblem() serverProblem {
	return serverProblem{Code: failedRunCode, Message: failedRunMessage}
}

var runRefusals = []causedRefusal{
	{cause: domain.ErrUnverifiedIdentityToken, refusal: unverifiedTokenProblem(), level: slog.LevelWarn},
	{cause: domain.ErrIdentityIssuerUnavailable, refusal: unverifiedTokenProblem(), level: slog.LevelError},
	{cause: application.ErrForeignRepository, refusal: foreignRepositoryProblem(), level: slog.LevelWarn},
	{cause: domain.ErrNoPlanForHead, refusal: unplannedRunProblem(), level: slog.LevelWarn},
	{cause: domain.ErrRunAlreadyReported, refusal: reportedRunProblem(), level: slog.LevelWarn},
}

func runRefusalOf(failure error) loggedAnswer {
	row, isMatched := refusalCausing(runRefusals, failure)
	if !isMatched {
		return loggedAnswer{answer: failedRunProblem(), reason: failedRunCode, cause: failure, level: slog.LevelError}
	}
	return loggedAnswer{answer: row.refusal, reason: row.refusal.reason(), cause: failure, level: row.level}
}

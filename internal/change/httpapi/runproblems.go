// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

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

var runRefusals = []struct {
	cause   error
	refusal refusal
}{
	{domain.ErrUnverifiedIdentityToken, unverifiedTokenProblem()},
	{domain.ErrIdentityIssuerUnavailable, unverifiedTokenProblem()},
	{application.ErrForeignRepository, foreignRepositoryProblem()},
	{domain.ErrNoPlanForHead, unplannedProblem{Code: "no_plan", Message: "no plan was dispatched for this head"}},
	{domain.ErrRunAlreadyReported, reportedProblem{Code: "already_reported", Message: "this head's run was already reported"}},
}

func runRefusalOf(failure error) answerBody {
	matching := slices.IndexFunc(runRefusals, func(row struct {
		cause   error
		refusal refusal
	}) bool {
		return errors.Is(failure, row.cause)
	})
	if matching < 0 {
		return serverProblem{Code: "run_failed", Message: "the run could not be recorded"}
	}
	return runRefusals[matching].refusal
}

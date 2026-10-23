// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"

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

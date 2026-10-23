// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	oversizedReportCode    = "report_too_large"
	oversizedReportMessage = "the report declares a length above the byte bound"
)

var reportBytesLimit = fmt.Sprintf("%d bytes", domain.MaxRunReportBytes)

func oversizedReportProblem() refusal {
	return limitProblem{Code: oversizedReportCode, Message: oversizedReportMessage, Limit: reportBytesLimit}
}

func reportLengthRefusalOf(declared int64) refusal {
	if declared > domain.MaxRunReportBytes {
		return oversizedReportProblem()
	}
	return nil
}

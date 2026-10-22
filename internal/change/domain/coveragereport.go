// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrMalformedCoverageReport = errors.New("the coverage artifact is not parseable")

type CoverageReport interface {
	ReportedFiles(limits CoverageLimits) ([]ReportedFile, error)
}

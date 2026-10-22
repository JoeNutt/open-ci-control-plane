// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type CoverageReport interface {
	ReportedFiles(limits CoverageLimits) ([]ReportedFile, error)
}

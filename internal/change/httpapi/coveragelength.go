// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

func uploadLengthRefusalOf(declared int64) refusal {
	if declared == undeclaredLength {
		return undeclaredUploadLengthProblem()
	}
	if domain.DeclaredCoverageLimits().CompressedBytesBreach(declared) != nil {
		return oversizedUploadProblem()
	}
	return nil
}

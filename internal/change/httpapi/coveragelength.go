// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func uploadLengthRefusalOf(declared int64) refusal {
	limit := fmt.Sprintf("%d compressed bytes", domain.DeclaredCoverageLimits().MaxCompressedBytes)
	if declared == undeclaredLength {
		return limitProblem{Code: undeclaredLengthCode, Message: "the upload declares no length", Limit: limit}
	}
	if domain.DeclaredCoverageLimits().CompressedBytesBreach(declared) != nil {
		return limitProblem{Code: "artifact_too_large", Message: "too large", Limit: limit}
	}
	return nil
}

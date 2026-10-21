// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"

const undeclaredLength = -1

func lengthProblemIn(declared int64) (limitProblem, bool) {
	if declared == undeclaredLength {
		return undeclaredLengthProblem(), true
	}
	if declared > ingest.MaxDeliveryBytes {
		return oversizedBodyProblem(), true
	}
	return limitProblem{}, false
}

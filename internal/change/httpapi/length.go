// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"

const undeclaredLength = -1

func lengthRefusalOf(declared int64) refusal {
	if declared == undeclaredLength {
		return undeclaredLengthProblem()
	}
	if declared > ingest.MaxDeliveryBytes {
		return oversizedBodyProblem()
	}
	return nil
}

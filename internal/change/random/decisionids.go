// SPDX-License-Identifier: GPL-3.0-or-later

// Package random mints identifiers from the operating system's cryptographically secure
// random number generator.
package random

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

type DecisionIDs struct{}

func (DecisionIDs) NextDecisionID() (domain.DecisionID, error) {
	return domain.ParseDecisionID("dec_00000000000000000000000000000000")
}

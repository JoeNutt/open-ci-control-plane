// SPDX-License-Identifier: GPL-3.0-or-later

// Package random mints identifiers from the operating system's cryptographically secure
// random number generator.
package random

import (
	"crypto/rand"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type DecisionIDs struct{}

func (DecisionIDs) NextDecisionID() (domain.DecisionID, error) {
	var entropy [domain.DecisionIDEntropyBytes]byte
	// crypto/rand.Read never returns an error and always fills its buffer; on a failed
	// read it crashes the program irrecoverably. go doc crypto/rand.Read, go1.26.8.
	rand.Read(entropy[:])
	return domain.DecisionIDFrom(entropy), nil
}

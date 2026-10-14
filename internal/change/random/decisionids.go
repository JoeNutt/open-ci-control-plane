// SPDX-License-Identifier: GPL-3.0-or-later

// Package random mints identifiers from the operating system's cryptographically secure
// random number generator.
package random

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	decisionIDPrefix       = "dec_"
	decisionIDEntropyBytes = 16
)

type DecisionIDs struct{}

func (DecisionIDs) NextDecisionID() (domain.DecisionID, error) {
	entropy := make([]byte, decisionIDEntropyBytes)
	// crypto/rand.Read never returns an error and always fills its buffer; on a failed
	// read it crashes the program irrecoverably. go doc crypto/rand.Read, go1.26.8.
	rand.Read(entropy)
	return domain.ParseDecisionID(decisionIDPrefix + hex.EncodeToString(entropy))
}

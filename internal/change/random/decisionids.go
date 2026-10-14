// SPDX-License-Identifier: GPL-3.0-or-later

// Package random mints identifiers from the operating system's cryptographically secure
// random number generator.
package random

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type DecisionIDs struct{}

func (DecisionIDs) NextDecisionID() (domain.DecisionID, error) {
	b := make([]byte, 16)
	rand.Read(b)
	return domain.ParseDecisionID("dec_" + hex.EncodeToString(b))
}

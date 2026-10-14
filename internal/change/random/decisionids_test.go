// SPDX-License-Identifier: GPL-3.0-or-later

package random_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/random"
)

func TestNextDecisionIDMintsAWellFormedIdentifier(t *testing.T) {
	minted, err := random.DecisionIDs{}.NextDecisionID()
	if err != nil {
		t.Fatalf("mint decision identifier: %v", err)
	}

	if _, err := domain.ParseDecisionID(minted.String()); err != nil {
		t.Errorf("minted %q, which does not parse: %v", minted.String(), err)
	}
}

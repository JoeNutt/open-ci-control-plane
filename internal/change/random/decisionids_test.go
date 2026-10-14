// SPDX-License-Identifier: GPL-3.0-or-later

package random_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/random"
)

func mintedDecisionID(t *testing.T) domain.DecisionID {
	t.Helper()

	minted, err := random.DecisionIDs{}.NextDecisionID()
	if err != nil {
		t.Fatalf("mint decision identifier: %v", err)
	}
	return minted
}

func TestDecisionIDsStandsInWhereverTheDomainAsksForADecisionIDSource(t *testing.T) {
	var source domain.DecisionIDSource = random.DecisionIDs{}

	if _, err := source.NextDecisionID(); err != nil {
		t.Fatalf("mint decision identifier through the domain port: %v", err)
	}
}

func TestNextDecisionIDMintsAWellFormedIdentifier(t *testing.T) {
	minted := mintedDecisionID(t)

	if _, err := domain.ParseDecisionID(minted.String()); err != nil {
		t.Errorf("minted %q, which does not parse: %v", minted.String(), err)
	}
}

func TestNextDecisionIDNeverRepeatsAnIdentifier(t *testing.T) {
	const mintings = 1000
	seen := make(map[domain.DecisionID]int, mintings)

	for minting := range mintings {
		minted := mintedDecisionID(t)
		if earlier, repeated := seen[minted]; repeated {
			t.Fatalf("minting %d repeated %q from minting %d", minting, minted.String(), earlier)
		}
		seen[minted] = minting
	}
}

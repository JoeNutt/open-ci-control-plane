// SPDX-License-Identifier: GPL-3.0-or-later

package random_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
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

func TestEveryDigitOfAMintedIdentifierVaries(t *testing.T) {
	const mintings = 64
	seen := map[int]map[rune]bool{}

	for range mintings {
		for position, character := range mintedDecisionID(t).String() {
			if seen[position] == nil {
				seen[position] = map[rune]bool{}
			}
			seen[position][character] = true
		}
	}

	varying := 0
	for _, characters := range seen {
		if len(characters) > 1 {
			varying++
		}
	}
	if want := 2 * domain.DecisionIDEntropyBytes; varying < want {
		t.Errorf("%d characters varied across %d mintings, want the %d random hex digits",
			varying, mintings, want)
	}
}

func TestRandomnessComesOnlyFromTheCryptographicGenerator(t *testing.T) {
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list the package's files: %v", err)
	}
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), source, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", source, err)
		}
		for _, spec := range parsed.Imports {
			if strings.HasPrefix(strings.Trim(spec.Path.Value, `"`), "math/rand") {
				t.Errorf("%s imports %s, which is predictable", source, spec.Path.Value)
			}
		}
	}
}

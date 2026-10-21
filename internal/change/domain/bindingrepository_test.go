// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseGovernedRepositoryKeepsTheRepository(t *testing.T) {
	parsed, err := domain.ParseGovernedRepository(thisRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	if parsed.String() != thisRepository {
		t.Errorf("repository is %q, want %s", parsed, thisRepository)
	}
}

func TestParseGovernedRepositoryRefusesWhatIsNotAnOwnerNamePair(t *testing.T) {
	refused := map[string]string{
		"empty":            "",
		"no owner":         "open-ci-control-plane",
		"three segments":   "JoeNutt/open-ci/control-plane",
		"a leading hyphen": "-JoeNutt/open-ci-control-plane",
		"a page of name":   "JoeNutt/" + strings.Repeat("a", domain.DeclaredLimits().MaxPathBytes),
	}
	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseGovernedRepository(stated); !errors.Is(err, domain.ErrMalformedRepository) {
				t.Fatalf("%s gave %v, want ErrMalformedRepository", form, err)
			}
		})
	}
}

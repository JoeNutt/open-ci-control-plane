// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const policyRepository = "JoeNutt/open-ci-policy"

func TestParsePolicyRepositoryKeepsItsOwnerAndName(t *testing.T) {
	parsed, err := domain.ParsePolicyRepository(policyRepository)
	if err != nil {
		t.Fatalf("parse policy repository: %v", err)
	}
	if parsed.Owner() != "JoeNutt" || parsed.Name() != "open-ci-policy" || parsed.String() != policyRepository {
		t.Errorf("policy repository is %q owned by %q named %q, want %s", parsed, parsed.Owner(), parsed.Name(), policyRepository)
	}
}

func TestParsePolicyRepositoryRefusesWhatIsNotAnOwnerNamePair(t *testing.T) {
	if _, err := domain.ParsePolicyRepository("open-ci-policy"); !errors.Is(err, domain.ErrMalformedRepository) {
		t.Fatalf("a name with no owner gave %v, want ErrMalformedRepository", err)
	}
}

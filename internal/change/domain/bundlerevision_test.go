// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const wellFormedRevision = "4f368b8c2e1d0a9b8c7d6e5f4a3b2c1d0e9f8a7b"

func TestParseBundleRevisionKeepsTheRevisionStated(t *testing.T) {
	parsed, err := domain.ParseBundleRevision(wellFormedRevision)

	if err != nil {
		t.Fatalf("parse %q: %v", wellFormedRevision, err)
	}
	if parsed.String() != wellFormedRevision {
		t.Errorf("parsed to %q, want %q", parsed.String(), wellFormedRevision)
	}
}

func TestParseBundleRevisionRefusesAnEmptyRevision(t *testing.T) {
	if _, err := domain.ParseBundleRevision(""); !errors.Is(err, domain.ErrMissingBundleRevision) {
		t.Fatalf("empty revision gave %v, want ErrMissingBundleRevision", err)
	}
}

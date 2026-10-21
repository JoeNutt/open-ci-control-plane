// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseBundleRevisionKeepsTheRevisionStated(t *testing.T) {
	const stated = "4f368b8c2e1d0a9b8c7d6e5f4a3b2c1d0e9f8a7b"

	parsed, err := domain.ParseBundleRevision(stated)

	if err != nil {
		t.Fatalf("parse %q: %v", stated, err)
	}
	if parsed.String() != stated {
		t.Errorf("parsed to %q, want %q", parsed.String(), stated)
	}
}

func TestParseBundleRevisionRefusesAnEmptyRevision(t *testing.T) {
	if _, err := domain.ParseBundleRevision(""); !errors.Is(err, domain.ErrMissingBundleRevision) {
		t.Fatalf("empty revision gave %v, want ErrMissingBundleRevision", err)
	}
}

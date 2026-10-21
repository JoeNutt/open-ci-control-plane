// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
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

func TestParseBundleRevisionRefusesAnythingButAFullCommitSHA(t *testing.T) {
	refused := map[string]string{
		"abbreviated":       wellFormedRevision[:7],
		"upper-case hex":    strings.ToUpper(wellFormedRevision),
		"a branch name":     "main",
		"a forged log line": wellFormedRevision[:39] + "\n",
		"far too long":      strings.Repeat(wellFormedRevision, 1000),
	}

	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseBundleRevision(stated); !errors.Is(err, domain.ErrMalformedCommitSHA) {
				t.Fatalf("%q gave %v, want ErrMalformedCommitSHA", stated, err)
			}
		})
	}
}

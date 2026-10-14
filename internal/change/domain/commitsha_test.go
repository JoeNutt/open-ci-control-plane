// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const wellFormedCommitSHA = "9f1cabc0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6"

func TestParseCommitSHAKeepsAFullLowerHexObjectName(t *testing.T) {
	parsed, err := domain.ParseCommitSHA(wellFormedCommitSHA)

	if err != nil {
		t.Fatalf("parse %q: %v", wellFormedCommitSHA, err)
	}
	if parsed.String() != wellFormedCommitSHA {
		t.Errorf("parsed to %q, want %q", parsed.String(), wellFormedCommitSHA)
	}
}

func TestParseCommitSHARefusesAnythingElse(t *testing.T) {
	refused := map[string]string{
		"empty":           "",
		"abbreviated":     wellFormedCommitSHA[:7],
		"one digit short": wellFormedCommitSHA[:39],
		"one digit long":  wellFormedCommitSHA + "0",
		"upper case":      strings.ToUpper(wellFormedCommitSHA),
		"not hex":         wellFormedCommitSHA[:39] + "g",
		"a path":          "../../../../user/repos?per_page=100000000",
		"a branch name":   "refs/heads/main",
	}

	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseCommitSHA(stated); !errors.Is(err, domain.ErrMalformedCommitSHA) {
				t.Fatalf("%q gave %v, want ErrMalformedCommitSHA", stated, err)
			}
		})
	}
}

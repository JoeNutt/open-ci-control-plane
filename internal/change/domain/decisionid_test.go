// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const wellFormedDecisionID = "dec_0123456789abcdef0123456789abcdef"

func TestParseDecisionIDKeepsAWellFormedIdentifier(t *testing.T) {
	parsed, err := domain.ParseDecisionID(wellFormedDecisionID)

	if err != nil {
		t.Fatalf("parse %q: %v", wellFormedDecisionID, err)
	}
	if parsed.String() != wellFormedDecisionID {
		t.Errorf("parsed to %q, want %q", parsed.String(), wellFormedDecisionID)
	}
}

func TestParseDecisionIDRefusesAnythingElse(t *testing.T) {
	refused := map[string]string{
		"empty":           "",
		"no prefix":       "0123456789abcdef0123456789abcdef",
		"another prefix":  "bg_0123456789abcdef0123456789abcdef",
		"one digit short": strings.TrimSuffix(wellFormedDecisionID, "f"),
		"one digit long":  wellFormedDecisionID + "0",
		"upper case hex":  "dec_0123456789ABCDEF0123456789abcdef",
		"not hex":         "dec_0123456789abcdef0123456789abcdeg",
		"a path":          "dec_../../../etc/passwd000000000000",
	}

	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseDecisionID(stated); !errors.Is(err, domain.ErrMalformedDecisionID) {
				t.Fatalf("%q gave %v, want ErrMalformedDecisionID", stated, err)
			}
		})
	}
}

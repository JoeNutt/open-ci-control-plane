// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseSuiteNameRefusesANameThatIsNotAnImportPath(t *testing.T) {
	refused := map[string]string{
		"empty":                "",
		"a dot-dot segment":    "github.com/JoeNutt/../secrets",
		"only dot-dot":         "..",
		"a dot segment":        "internal/./change",
		"an empty segment":     "internal//change",
		"a leading separator":  "/internal/change",
		"a trailing separator": "internal/change/",
		"a space":              "internal/change domain",
		"a backslash":          `internal\change`,
		"a colon":              "internal:change",
		"a percent escape":     "internal%2fchange",
		"a non-ASCII letter":   "internal/chånge",
		"a NUL byte":           "internal\x00change",
	}

	for form, name := range refused {
		t.Run(form, func(t *testing.T) {
			_, err := domain.ParseSuiteName(name)

			if !errors.Is(err, domain.ErrMalformedSuiteName) {
				t.Fatalf("want ErrMalformedSuiteName for %q, got %v", name, err)
			}
		})
	}
}

func TestParseSuiteNameRefusesANameOverTheBound(t *testing.T) {
	overlong := strings.Repeat("a", domain.MaxSuiteNameBytes+1)

	_, err := domain.ParseSuiteName(overlong)

	if !errors.Is(err, domain.ErrMalformedSuiteName) {
		t.Fatalf("want ErrMalformedSuiteName, got %v", err)
	}
}

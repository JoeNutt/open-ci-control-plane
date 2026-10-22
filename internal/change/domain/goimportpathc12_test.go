// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseGoImportPathC12AcceptsAGoImportPath(t *testing.T) {
	accepted := map[string]string{
		"a package of this module": "github.com/JoeNutt/open-ci-control-plane/internal/change/domain",
		"a name at the bound":      strings.Repeat("a", domain.MaxGoImportPathBytes),
	}

	for form, name := range accepted {
		t.Run(form, func(t *testing.T) {
			suite, err := domain.ParseGoImportPath(name)

			if err != nil {
				t.Fatalf("parse %q: %v", name, err)
			}
			if suite.String() != name {
				t.Fatalf("suite reads %q, want %q", suite.String(), name)
			}
		})
	}
}

func TestParseGoImportPathC12RefusesANameThatIsNotAnImportPath(t *testing.T) {
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
			_, err := domain.ParseGoImportPath(name)

			if !errors.Is(err, domain.ErrMalformedGoImportPath) {
				t.Fatalf("want ErrMalformedGoImportPath for %q, got %v", name, err)
			}
		})
	}
}

func TestParseGoImportPathC12RefusesANameOverTheBound(t *testing.T) {
	overlong := strings.Repeat("a", domain.MaxGoImportPathBytes+1)

	_, err := domain.ParseGoImportPath(overlong)

	if !errors.Is(err, domain.ErrMalformedGoImportPath) {
		t.Fatalf("want ErrMalformedGoImportPath, got %v", err)
	}
}

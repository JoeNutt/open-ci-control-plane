// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseBindingAdmitsOnlyDirectoryPatternsInPaths(t *testing.T) {
	refused := map[string]string{
		"a file glob":             "internal/change/domain/*.go",
		"a bare spanning pattern": "**",
		"a character class":       "internal/[ab]/**",
		"one level only":          "internal/change/*",
		"a wildcard mid-path":     "internal/**/domain/**",
		"no wildcard":             "internal/change/domain",
		"an absolute directory":   "/internal/change/**",
		"a parent segment":        "../elsewhere/**",
	}
	for form, glob := range refused {
		t.Run(form, func(t *testing.T) {
			stated := thisRepositorysBinding()
			stated.Policies[0].Paths["domain"] = []string{glob}
			if _, err := domain.ParseBinding(stated); !errors.Is(err, domain.ErrUnsupportedPathPattern) {
				t.Fatalf("%s gave %v, want ErrUnsupportedPathPattern", form, err)
			}
		})
	}
}

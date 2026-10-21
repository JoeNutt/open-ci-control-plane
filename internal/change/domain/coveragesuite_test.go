// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseCoverageSuiteRefusesAnImportPathOutsideC12sCharacters(t *testing.T) {
	refused := map[string]string{
		"a plus":  "example.com/c++/parser",
		"a tilde": "example.com/~user/parser",
	}

	for form, suite := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseGoImportPath(suite); err != nil {
				t.Fatalf("%s is a Go import path, so this test needs one: %v", suite, err)
			}

			_, err := domain.ParseCoverageSuite(suite)

			if !errors.Is(err, domain.ErrMalformedCoverageSuite) {
				t.Fatalf("want ErrMalformedCoverageSuite for %q, got %v", suite, err)
			}
		})
	}
}

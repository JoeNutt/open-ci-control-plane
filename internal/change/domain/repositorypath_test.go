// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseRepositoryPathRefusesAnAbsolutePath(t *testing.T) {
	_, err := domain.ParseRepositoryPath("/home/runner/work/internal/change/domain/effect.go")

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
}

func TestParseRepositoryPathRefusesAPathNotInCanonicalForm(t *testing.T) {
	refused := map[string]string{
		"empty":                 "",
		"climbing out":          "../secrets/key.pem",
		"climbing out mid-path": "internal/../../key.pem",
		"a current segment":     "internal/./effect.go",
		"an empty segment":      "internal//effect.go",
		"a trailing separator":  "internal/",
	}

	for form, path := range refused {
		t.Run(form, func(t *testing.T) {
			_, err := domain.ParseRepositoryPath(path)

			if !errors.Is(err, domain.ErrPathOutsideRepository) {
				t.Fatalf("want ErrPathOutsideRepository for %q, got %v", path, err)
			}
		})
	}
}

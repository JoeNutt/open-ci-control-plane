// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseGovernedRepositoryAcceptsAGitHubRepository(t *testing.T) {
	const governed = "JoeNutt/open-ci-control-plane"

	repository, err := domain.ParseGovernedRepository(governed)

	if err != nil {
		t.Fatalf("parse %s: %v", governed, err)
	}
	if repository.String() != governed {
		t.Fatalf("repository reads %q, want %q", repository.String(), governed)
	}
}

func TestParseGovernedRepositoryRefusesAnythingButGitHubNameCharacters(t *testing.T) {
	refused := map[string]string{
		"no separator":      "open-ci-control-plane",
		"empty owner":       "/open-ci-control-plane",
		"empty name":        "JoeNutt/",
		"three segments":    "JoeNutt/open-ci-control-plane/coverage",
		"traversal":         "../open-ci-control-plane",
		"a space":           "JoeNutt/open ci",
		"an underscore":     "Joe_Nutt/open-ci-control-plane",
		"a dots-only name":  "JoeNutt/..",
		"an owner too long": strings.Repeat("a", 40) + "/open-ci-control-plane",
	}

	for form, repository := range refused {
		t.Run(form, func(t *testing.T) {
			_, err := domain.ParseGovernedRepository(repository)

			if !errors.Is(err, domain.ErrMalformedRepository) {
				t.Fatalf("want ErrMalformedRepository for %q, got %v", repository, err)
			}
		})
	}
}

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

func TestParseGovernedRepositorySegmentsReadsAsTheOwnerAndNameItWasGiven(t *testing.T) {
	repository, err := domain.ParseGovernedRepositorySegments("JoeNutt", "open-ci-control-plane")

	if err != nil {
		t.Fatalf("parse segments: %v", err)
	}
	if repository.Owner() != "JoeNutt" || repository.Name() != "open-ci-control-plane" {
		t.Fatalf("read %s/%s, want JoeNutt/open-ci-control-plane", repository.Owner(), repository.Name())
	}
}

func TestParseGovernedRepositorySegmentsRefusesASeparatorInsideEitherSegment(t *testing.T) {
	for _, segments := range [][2]string{{"JoeNutt/open-ci", "control-plane"}, {"JoeNutt", "open-ci/control-plane"}} {
		if _, err := domain.ParseGovernedRepositorySegments(segments[0], segments[1]); !errors.Is(err, domain.ErrMalformedRepository) {
			t.Errorf("segments %q: error is %v, want %v", segments, err, domain.ErrMalformedRepository)
		}
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

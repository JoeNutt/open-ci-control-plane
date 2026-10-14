// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParsePullRequestLocatorAcceptsADeclaredRepository(t *testing.T) {
	locator, err := domain.ParsePullRequestLocator("acme/payments", 412)

	if err != nil {
		t.Fatalf("parse acme/payments#412: %v", err)
	}
	if locator.Owner() != "acme" {
		t.Errorf("owner is %q, want acme", locator.Owner())
	}
	if locator.Name() != "payments" {
		t.Errorf("name is %q, want payments", locator.Name())
	}
	if locator.Number() != 412 {
		t.Errorf("number is %d, want 412", locator.Number())
	}
}

func TestParsePullRequestLocatorRefusesAMalformedRepository(t *testing.T) {
	refused := map[string]string{
		"no separator":    "payments",
		"empty owner":     "/payments",
		"empty name":      "acme/",
		"three segments":  "acme/payments/deep",
		"traversal":       "../../etc/passwd",
		"relative owner":  "../payments",
		"space":           "acme/pay ments",
		"url":             "https://github.com/acme/payments",
		"over the length": strings.Repeat("a", 101) + "/payments",
	}

	for form, repository := range refused {
		t.Run(form, func(t *testing.T) {
			locator, err := domain.ParsePullRequestLocator(repository, 412)

			if !errors.Is(err, domain.ErrMalformedRepository) {
				t.Fatalf("want ErrMalformedRepository for %q, got %v", repository, err)
			}
			if locator != (domain.PullRequestLocator{}) {
				t.Fatalf("want no locator alongside the refusal, got %+v", locator)
			}
		})
	}
}

func TestParsePullRequestLocatorRefusesANumberNoForgeIssues(t *testing.T) {
	for _, number := range []int{0, -1} {
		t.Run(strconv.Itoa(number), func(t *testing.T) {
			locator, err := domain.ParsePullRequestLocator("acme/payments", number)

			if !errors.Is(err, domain.ErrMalformedPullRequestNumber) {
				t.Fatalf("want ErrMalformedPullRequestNumber for %d, got %v", number, err)
			}
			if locator != (domain.PullRequestLocator{}) {
				t.Fatalf("want no locator alongside the refusal, got %+v", locator)
			}
		})
	}
}

func TestParsePullRequestLocatorAcceptsARepositoryBeginningWithAPeriod(t *testing.T) {
	locator, err := domain.ParsePullRequestLocator("acme/.github", 7)

	if err != nil {
		t.Fatalf("parse acme/.github#7: %v", err)
	}
	if locator.Name() != ".github" {
		t.Errorf("name is %q, want .github", locator.Name())
	}
}

func TestParsePullRequestLocatorRefusesANameOfNothingButPeriods(t *testing.T) {
	refused := map[string]string{
		"parent":  "acme/..",
		"current": "acme/.",
		"three":   "acme/...",
	}

	for form, repository := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParsePullRequestLocator(repository, 1); !errors.Is(err, domain.ErrMalformedRepository) {
				t.Fatalf("%q gave %v, want ErrMalformedRepository", repository, err)
			}
		})
	}
}

func TestParsePullRequestLocatorRefusesAnOwnerOutsideTheForgesCharacterSet(t *testing.T) {
	refused := map[string]string{
		"underscore": "my_org/payments",
		"period":     "my.org/payments",
	}

	for form, repository := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParsePullRequestLocator(repository, 1); !errors.Is(err, domain.ErrMalformedRepository) {
				t.Fatalf("%q gave %v, want ErrMalformedRepository", repository, err)
			}
		})
	}
}

func TestParsePullRequestLocatorRefusesAnOwnerOverTheForgesLength(t *testing.T) {
	owner := strings.Repeat("a", 40)

	_, err := domain.ParsePullRequestLocator(owner+"/payments", 1)

	if !errors.Is(err, domain.ErrMalformedRepository) {
		t.Fatalf("a %d-byte owner gave %v, want ErrMalformedRepository", len(owner), err)
	}
}

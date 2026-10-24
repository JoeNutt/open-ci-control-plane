// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	domainSuite = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	lcovSuite   = "github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
)

func TestRepositoryCoverageServesTheMapASuiteWasReplacedWith(t *testing.T) {
	suite := mustParseSuite(t, domainSuite)
	uploaded := twoFileSuiteOf(t)

	replaced := mustReplace(t, domain.RepositoryCoverage{}, suite, uploaded.coverage)

	served, err := replaced.SuiteCoverageOf(suite)
	if err != nil {
		t.Fatalf("serve the suite's map: %v", err)
	}
	if !reflect.DeepEqual(served, uploaded.coverage) {
		t.Fatalf("served a map other than the one the suite was replaced with")
	}
}

func TestRepositoryCoverageRefusesASuiteItHoldsNoMapFor(t *testing.T) {
	_, err := domain.RepositoryCoverage{}.SuiteCoverageOf(mustParseSuite(t, domainSuite))

	if !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("want ErrNoCoverageMap, got %v", err)
	}
}

func TestRepositoryCoverageReplacingOneSuiteKeepsTheOthersAndChangesNothingHeld(t *testing.T) {
	first := mustParseSuite(t, domainSuite)
	second := mustParseSuite(t, lcovSuite)
	uploaded := twoFileSuiteOf(t)
	before := mustReplace(t, domain.RepositoryCoverage{}, first, uploaded.coverage)

	after := mustReplace(t, before, second, uploaded.coverage)

	if _, err := after.SuiteCoverageOf(first); err != nil {
		t.Errorf("replacing one suite lost another: %v", err)
	}
	if _, err := before.SuiteCoverageOf(second); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Errorf("replacing changed the value already held: %v", err)
	}
}

func TestRepositoryCoverageRefusesANewSuiteBeyondTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	full := coverageOfSuites(t, limits.MaxSuitesPerRepository)

	_, err := full.Replacing(generatedSuite(t, limits.MaxSuitesPerRepository), twoFileSuiteOf(t).coverage, limits)

	if !errors.Is(err, domain.ErrTooManySuites) {
		t.Fatalf("want ErrTooManySuites, got %v", err)
	}
}

func TestRepositoryCoverageReplacesAHeldSuiteAtTheBound(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	full := coverageOfSuites(t, limits.MaxSuitesPerRepository)

	_, err := full.Replacing(generatedSuite(t, 0), twoFileSuiteOf(t).coverage, limits)

	if err != nil {
		t.Fatalf("want a held suite replaced at the bound, got %v", err)
	}
}

func coverageOfSuites(t *testing.T, count int) domain.RepositoryCoverage {
	t.Helper()

	coverage := twoFileSuiteOf(t).coverage
	var held domain.RepositoryCoverage
	for index := range count {
		held = mustReplace(t, held, generatedSuite(t, index), coverage)
	}
	return held
}

func mustReplace(
	t *testing.T, held domain.RepositoryCoverage, suite domain.CoverageSuite, coverage domain.SuiteCoverage,
) domain.RepositoryCoverage {
	t.Helper()

	replaced, err := held.Replacing(suite, coverage, domain.DeclaredCoverageLimits())
	if err != nil {
		t.Fatalf("replace suite %s: %v", suite, err)
	}
	return replaced
}

func generatedSuite(t *testing.T, index int) domain.CoverageSuite {
	t.Helper()

	return mustParseSuite(t, fmt.Sprintf("github.com/JoeNutt/open-ci-control-plane/generated/suite%d", index))
}

func mustParseSuite(t *testing.T, name string) domain.CoverageSuite {
	t.Helper()

	parsed, err := domain.ParseCoverageSuite(name)
	if err != nil {
		t.Fatalf("parse suite name %q: %v", name, err)
	}
	return parsed
}

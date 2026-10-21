// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	domainSuite = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	lcovSuite   = "github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/lcov"
)

func TestRepositoryCoverageServesTheMapASuiteWasReplacedWith(t *testing.T) {
	suite := mustParseSuiteName(t, domainSuite)
	uploaded := twoFileSuiteOf(t)

	replaced := domain.RepositoryCoverage{}.Replacing(suite, uploaded.coverage)

	served, err := replaced.SuiteCoverageOf(suite)
	if err != nil {
		t.Fatalf("serve the suite's map: %v", err)
	}
	if !served.Covers(uploaded.exercised) || served.FileEntryCount() != uploaded.reportedCount {
		t.Fatalf("served a map other than the one the suite was replaced with")
	}
}

func TestRepositoryCoverageRefusesASuiteItHoldsNoMapFor(t *testing.T) {
	_, err := domain.RepositoryCoverage{}.SuiteCoverageOf(mustParseSuiteName(t, domainSuite))

	if !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Fatalf("want ErrNoCoverageMap, got %v", err)
	}
}

func TestRepositoryCoverageReplacingOneSuiteKeepsTheOthersAndChangesNothingHeld(t *testing.T) {
	first := mustParseSuiteName(t, domainSuite)
	second := mustParseSuiteName(t, lcovSuite)
	uploaded := twoFileSuiteOf(t)
	before := domain.RepositoryCoverage{}.Replacing(first, uploaded.coverage)

	after := before.Replacing(second, uploaded.coverage)

	if _, err := after.SuiteCoverageOf(first); err != nil {
		t.Errorf("replacing one suite lost another: %v", err)
	}
	if _, err := before.SuiteCoverageOf(second); !errors.Is(err, domain.ErrNoCoverageMap) {
		t.Errorf("replacing changed the value already held: %v", err)
	}
}

func mustParseSuiteName(t *testing.T, name string) domain.SuiteName {
	t.Helper()

	parsed, err := domain.ParseSuiteName(name)
	if err != nil {
		t.Fatalf("parse suite name %q: %v", name, err)
	}
	return parsed
}

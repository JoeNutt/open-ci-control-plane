// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestRepositoryCoverageServesTheMapASuiteWasReplacedWith(t *testing.T) {
	suite := mustParseSuiteName(t, "github.com/JoeNutt/open-ci-control-plane/internal/change/domain")
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

func mustParseSuiteName(t *testing.T, name string) domain.SuiteName {
	t.Helper()

	parsed, err := domain.ParseSuiteName(name)
	if err != nil {
		t.Fatalf("parse suite name %q: %v", name, err)
	}
	return parsed
}

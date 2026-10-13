// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
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

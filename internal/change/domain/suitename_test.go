// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseSuiteNameRefusesANameOverTheBound(t *testing.T) {
	overlong := strings.Repeat("a", domain.MaxSuiteNameBytes+1)

	_, err := domain.ParseSuiteName(overlong)

	if !errors.Is(err, domain.ErrMalformedSuiteName) {
		t.Fatalf("want ErrMalformedSuiteName, got %v", err)
	}
}

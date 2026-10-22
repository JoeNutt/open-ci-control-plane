// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseBoundedSourceRefusesASourceOneByteOverTheLimit(t *testing.T) {
	limits := domain.DeclaredLimits()

	_, err := domain.ParseBoundedSource(make([]byte, limits.MaxSourceBytes+1), limits)

	if !errors.Is(err, domain.ErrSourceTooLarge) {
		t.Fatalf("error is %v, want %v", err, domain.ErrSourceTooLarge)
	}
}

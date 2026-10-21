// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseFileImportsRefusesOneImportOverTheLimit(t *testing.T) {
	limits := domain.DeclaredLimits()

	_, err := domain.ParseFileImports(make([]string, limits.MaxImportsPerFile+1), limits)

	if !errors.Is(err, domain.ErrTooManyImportsInFile) {
		t.Fatalf("error is %v, want %v", err, domain.ErrTooManyImportsInFile)
	}
}

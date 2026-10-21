// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseRepositoryPathRefusesAnAbsolutePath(t *testing.T) {
	_, err := domain.ParseRepositoryPath("/home/runner/work/internal/change/domain/effect.go")

	if !errors.Is(err, domain.ErrPathOutsideRepository) {
		t.Fatalf("want ErrPathOutsideRepository, got %v", err)
	}
}

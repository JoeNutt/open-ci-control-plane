// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestBundleDeclaringTheCurrentMajorIsAccepted(t *testing.T) {
	major, err := domain.AcceptBundleMajor(int(domain.CurrentSchemaMajor))

	if err != nil {
		t.Fatalf("accept major %d: %v", domain.CurrentSchemaMajor, err)
	}
	if major != domain.CurrentSchemaMajor {
		t.Fatalf("accepted major %d, want %d", major, domain.CurrentSchemaMajor)
	}
}

func TestBundleDeclaringAnotherMajorIsRefused(t *testing.T) {
	declared := int(domain.CurrentSchemaMajor) + 1

	major, err := domain.AcceptBundleMajor(declared)

	if !errors.Is(err, domain.ErrSchemaMajorMismatch) {
		t.Fatalf("want ErrSchemaMajorMismatch for major %d, got %v", declared, err)
	}
	if major != 0 {
		t.Fatalf("want no accepted major alongside the refusal, got %d", major)
	}
}

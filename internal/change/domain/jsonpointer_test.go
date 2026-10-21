// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseJSONPointerUnescapesEachReferenceToken(t *testing.T) {
	parsed, err := domain.ParseJSONPointer("/diff/a~1b/c~0d/0")

	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"diff", "a/b", "c~d", "0"}
	if !slices.Equal(parsed.Tokens(), want) {
		t.Errorf("tokens %q, want %q", parsed.Tokens(), want)
	}
}

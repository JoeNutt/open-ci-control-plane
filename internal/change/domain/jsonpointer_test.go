// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
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

func TestParseJSONPointerRefusesAnythingThatNamesNoField(t *testing.T) {
	refused := map[string]string{
		"the whole document": "",
		"no leading slash":   "diff/files",
		"tilde then two":     "/diff/a~2b",
		"trailing tilde":     "/diff/a~",
	}

	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseJSONPointer(stated); !errors.Is(err, domain.ErrMalformedJSONPointer) {
				t.Fatalf("%q gave %v, want ErrMalformedJSONPointer", stated, err)
			}
		})
	}
}

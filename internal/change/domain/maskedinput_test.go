// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestMaskedInputOfOmitsAFieldTheMaskNames(t *testing.T) {
	document := domain.Document{}
	document.PR.Author.Login = "octocat"
	document.Diff.Sensitive = domain.Declared[string]{"/pr/author/login"}

	masked, err := domain.MaskedInputOf(document)
	if err != nil {
		t.Fatalf("mask: %v", err)
	}

	encoded, err := json.Marshal(masked)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(string(encoded), "octocat") {
		t.Errorf("masked input still carries the sensitive value: %s", encoded)
	}
}

func TestMaskedInputOfOmitsAFieldInsideAnArrayElement(t *testing.T) {
	document := domain.Document{}
	document.Analyzers.Findings = domain.Declared[domain.Finding]{
		{Path: "internal/kept.go"}, {Path: "internal/leaked.go"},
	}
	document.Diff.Sensitive = domain.Declared[string]{"/analyzers/findings/1/path"}

	masked, err := domain.MaskedInputOf(document)
	if err != nil {
		t.Fatalf("mask: %v", err)
	}

	encoded, err := json.Marshal(masked)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(string(encoded), "leaked.go") {
		t.Errorf("masked input still carries the sensitive value: %s", encoded)
	}
	if !strings.Contains(string(encoded), "kept.go") {
		t.Errorf("masked input lost a value the mask did not name: %s", encoded)
	}
}

func TestMaskedInputOfOmitsAWholeArrayElement(t *testing.T) {
	document := domain.Document{}
	document.Analyzers.Findings = domain.Declared[domain.Finding]{
		{Path: "internal/kept.go"}, {Path: "internal/leaked.go"},
	}
	document.Diff.Sensitive = domain.Declared[string]{"/analyzers/findings/1"}

	masked, err := domain.MaskedInputOf(document)
	if err != nil {
		t.Fatalf("mask: %v", err)
	}

	encoded, err := json.Marshal(masked)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(string(encoded), "leaked.go") {
		t.Errorf("masked input still carries the sensitive element: %s", encoded)
	}
	if !strings.Contains(string(encoded), "kept.go") {
		t.Errorf("masked input lost an element the mask did not name: %s", encoded)
	}
}

func TestMaskedInputOfRefusesAnArrayIndexRFC6901Forbids(t *testing.T) {
	forbidden := map[string]string{
		"leading zero": "/analyzers/findings/01/path",
		"plus sign":    "/analyzers/findings/+1/path",
		"minus sign":   "/analyzers/findings/-1/path",
	}

	for form, stated := range forbidden {
		t.Run(form, func(t *testing.T) {
			document := domain.Document{}
			document.Analyzers.Findings = domain.Declared[domain.Finding]{{}, {Path: "internal/leaked.go"}}
			document.Diff.Sensitive = domain.Declared[string]{stated}

			if _, err := domain.MaskedInputOf(document); !errors.Is(err, domain.ErrMalformedJSONPointer) {
				t.Fatalf("mask %q gave %v, want ErrMalformedJSONPointer", stated, err)
			}
		})
	}
}

func TestMaskedInputOfSkipsAPointerToAFieldThatCannotExist(t *testing.T) {
	unreachable := map[string]string{
		"through a missing member": "/pr/missing/login",
		"through a number":         "/pr/number/login",
	}

	for form, stated := range unreachable {
		t.Run(form, func(t *testing.T) {
			document := domain.Document{}
			document.Diff.Sensitive = domain.Declared[string]{stated}

			if _, err := domain.MaskedInputOf(document); err != nil {
				t.Fatalf("mask %q: %v", stated, err)
			}
		})
	}
}

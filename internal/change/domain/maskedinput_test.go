// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestMaskedInputOfOmitsTheAuthorLogin(t *testing.T) {
	document := domain.Document{}
	document.PR.Author.Login = "octocat"
	document.Diff.Sensitive = domain.Declared[string]{"/pr/author/login"}

	encoded := encodedMaskOf(t, document)

	if strings.Contains(encoded, "octocat") {
		t.Errorf("masked input still carries the author login: %s", encoded)
	}
}

func TestMaskedInputOfOmitsAWholeAnalyzerFinding(t *testing.T) {
	document := documentWithFindings("internal/kept.go", "internal/leaked.go")
	document.Diff.Sensitive = domain.Declared[string]{"/analyzers/findings/1"}

	encoded := encodedMaskOf(t, document)

	if strings.Contains(encoded, "leaked.go") {
		t.Errorf("masked input still carries the sensitive finding: %s", encoded)
	}
	if !strings.Contains(encoded, "kept.go") {
		t.Errorf("masked input lost a finding the mask did not name: %s", encoded)
	}
}

func TestMaskedInputOfListsOnlyThePointersItRemoved(t *testing.T) {
	document := documentWithFindings("internal/leaked.go")
	document.Diff.Sensitive = domain.Declared[string]{"/analyzers/findings/0", "/analyzers/findings/5"}

	masked, err := domain.MaskedInputOf(document)
	if err != nil {
		t.Fatalf("mask: %v", err)
	}

	want := []string{"/analyzers/findings/0"}
	if !slices.Equal(masked.RedactedPointers(), want) {
		t.Errorf("redacted %q, want %q", masked.RedactedPointers(), want)
	}
}

func TestMaskedInputOfRefusesAFieldThatCannotBeMasked(t *testing.T) {
	unmaskable := map[string]string{
		"a scalar the spec never marks sensitive": "/pr/number",
		"a member that does not exist":            "/pr/missing",
		"a field inside a finding":                "/analyzers/findings/1/path",
		"a whole approval":                        "/pr/approvals/0",
	}

	for form, stated := range unmaskable {
		t.Run(form, func(t *testing.T) {
			document := documentWithFindings("internal/kept.go", "internal/leaked.go")
			document.Diff.Sensitive = domain.Declared[string]{stated}

			if _, err := domain.MaskedInputOf(document); !errors.Is(err, domain.ErrUnmaskableField) {
				t.Fatalf("mask %q gave %v, want ErrUnmaskableField", stated, err)
			}
		})
	}
}

func TestMaskedInputOfRefusesAnArrayIndexRFC6901Forbids(t *testing.T) {
	forbidden := map[string]string{
		"leading zero": "/analyzers/findings/01",
		"plus sign":    "/analyzers/findings/+1",
		"minus sign":   "/analyzers/findings/-1",
	}

	for form, stated := range forbidden {
		t.Run(form, func(t *testing.T) {
			document := documentWithFindings("internal/kept.go", "internal/leaked.go")
			document.Diff.Sensitive = domain.Declared[string]{stated}

			if _, err := domain.MaskedInputOf(document); !errors.Is(err, domain.ErrMalformedJSONPointer) {
				t.Fatalf("mask %q gave %v, want ErrMalformedJSONPointer", stated, err)
			}
		})
	}
}

func documentWithFindings(paths ...string) domain.Document {
	document := domain.Document{}
	for _, path := range paths {
		document.Analyzers.Findings = append(document.Analyzers.Findings, domain.Finding{Path: path})
	}
	return document
}

func encodedMaskOf(t *testing.T, document domain.Document) string {
	t.Helper()
	masked, err := domain.MaskedInputOf(document)
	if err != nil {
		t.Fatalf("mask: %v", err)
	}
	encoded, err := json.Marshal(masked)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return string(encoded)
}

// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseGoImportPathKeepsThePath(t *testing.T) {
	const stated = "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

	parsed, err := domain.ParseGoImportPath(stated)
	if err != nil {
		t.Fatalf("parse import path: %v", err)
	}
	if parsed.String() != stated {
		t.Errorf("import path is %q, want %q", parsed, stated)
	}
}

func TestParseGoImportPathRefusesWhatIsNotAnImportPath(t *testing.T) {
	refused := map[string]string{
		"empty":               "",
		"a leading slash":     "/github.com/JoeNutt",
		"a trailing slash":    "github.com/JoeNutt/",
		"an empty segment":    "github.com//JoeNutt",
		"a parent segment":    "github.com/../JoeNutt",
		"a current segment":   "github.com/./JoeNutt",
		"a space":             "github.com/Joe Nutt",
		"a backslash":         `github.com\JoeNutt`,
		"one byte over bound": strings.Repeat("a", domain.MaxGoImportPathBytes+1),
	}
	for form, stated := range refused {
		t.Run(form, func(t *testing.T) {
			if _, err := domain.ParseGoImportPath(stated); !errors.Is(err, domain.ErrMalformedGoImportPath) {
				t.Fatalf("%s gave %v, want ErrMalformedGoImportPath", form, err)
			}
		})
	}
}

func TestParseGoImportPathAdmitsAPathAtTheBound(t *testing.T) {
	if _, err := domain.ParseGoImportPath(strings.Repeat("a", domain.MaxGoImportPathBytes)); err != nil {
		t.Fatalf("an import path at the bound was refused: %v", err)
	}
}

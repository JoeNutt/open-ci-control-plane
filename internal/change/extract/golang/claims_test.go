// SPDX-License-Identifier: GPL-3.0-or-later

package golang_test

import (
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
)

func TestExtractorClaimsOnlyGoSourceFiles(t *testing.T) {
	claimed := map[string]bool{
		"internal/change/domain/document.go":      true,
		"internal/change/domain/document_test.go": true,
		"go.mod":                        false,
		"README.md":                     false,
		"schema/ci.diff.v1.schema.json": false,
		"internal/change/domain/go":     false,
	}
	var extractor domain.ImportExtractor = golang.Extractor{}

	for path, want := range claimed {
		t.Run(path, func(t *testing.T) {
			if extractor.Claims(path) != want {
				t.Fatalf("claims %s is %v, want %v", path, !want, want)
			}
		})
	}
}

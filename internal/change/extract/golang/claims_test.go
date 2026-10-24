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

func TestExtractorReadsTestsOnlyFromTestFilesTheGoToolBuilds(t *testing.T) {
	declared := map[string]bool{
		"internal/change/domain/document_test.go":           true,
		"document_test.go":                                  true,
		"internal/change/domain/document.go":                false,
		"internal/change/domain/document_test.go.orig":      false,
		"internal/change/extract/golang/testdata/x_test.go": false,
		"internal/_scratch/x_test.go":                       false,
		"internal/.hidden/x_test.go":                        false,
		"internal/change/domain/_x_test.go":                 false,
	}

	for path, want := range declared {
		t.Run(path, func(t *testing.T) {
			if (golang.Extractor{}).DeclaresTests(path) != want {
				t.Fatalf("declares tests %s is %v, want %v", path, !want, want)
			}
		})
	}
}

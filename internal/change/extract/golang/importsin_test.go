// SPDX-License-Identifier: GPL-3.0-or-later

package golang_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
)

func TestImportsInReadsEveryImportAsWritten(t *testing.T) {
	imports, err := golang.ImportsIn(fixtureNamed(t, "grouped.go"))

	if err != nil {
		t.Fatalf("imports in grouped.go: %v", err)
	}
	want := []string{
		"context",
		"embed",
		"github.com/JoeNutt/open-ci-control-plane/internal/change/domain",
	}
	if !slices.Equal(imports, want) {
		t.Fatalf("imports are %q, want %q", imports, want)
	}
}

func fixtureNamed(t *testing.T, name string) []byte {
	t.Helper()

	source, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return source
}

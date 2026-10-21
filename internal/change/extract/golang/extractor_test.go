// SPDX-License-Identifier: GPL-3.0-or-later

package golang_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/extract/golang"
)

func TestImportsInReadsEveryImportAsWritten(t *testing.T) {
	imports, err := golang.Extractor{}.ImportsIn(boundedFrom(t, fixtureNamed(t, "grouped.go")))

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

func TestImportsInReadsASourceExactlyAtTheSourceByteBound(t *testing.T) {
	const declaration = "package fixture\n\nimport \"context\"\n\nvar padding = `"
	const closing = "`\n"
	limit := domain.DeclaredLimits().MaxSourceBytes
	padding := strings.Repeat("x", limit-len(declaration)-len(closing))
	source := []byte(declaration + padding + closing)

	imports, err := golang.Extractor{}.ImportsIn(boundedFrom(t, source))

	if err != nil {
		t.Fatalf("imports in a %d-byte source: %v", len(source), err)
	}
	if !slices.Equal(imports, []string{"context"}) {
		t.Fatalf("imports are %q, want [\"context\"]", imports)
	}
}

func TestImportsInRefusesASourceThatDoesNotParseWithoutQuotingIt(t *testing.T) {
	const quotedFromTheSource = "hunter2ApiKey"
	source := []byte("package fixture\n\nimport (\n\t\"context\" " + quotedFromTheSource + "\n)\n")

	_, err := golang.Extractor{}.ImportsIn(boundedFrom(t, source))

	if !errors.Is(err, golang.ErrUnparsableSource) {
		t.Fatalf("error is %v, want %v", err, golang.ErrUnparsableSource)
	}
	if strings.Contains(err.Error(), quotedFromTheSource) {
		t.Fatalf("the error quotes the source: %v", err)
	}
}

func TestImportsInRefusesAnImportThatIsNotAGoImportPathWithoutQuotingIt(t *testing.T) {
	const quotedFromTheSource = "hunter2 ApiKey"
	source := []byte("package fixture\n\nimport \"" + quotedFromTheSource + "\"\n")

	_, err := golang.Extractor{}.ImportsIn(boundedFrom(t, source))

	if !errors.Is(err, golang.ErrUnparsableSource) {
		t.Fatalf("error is %v, want %v", err, golang.ErrUnparsableSource)
	}
	if strings.Contains(err.Error(), quotedFromTheSource) {
		t.Fatalf("the error quotes the source: %v", err)
	}
}

func boundedFrom(t *testing.T, source []byte) domain.BoundedSource {
	t.Helper()

	bounded, err := domain.ParseBoundedSource(source, domain.DeclaredLimits())
	if err != nil {
		t.Fatalf("bound a %d-byte source: %v", len(source), err)
	}
	return bounded
}

func fixtureNamed(t *testing.T, name string) []byte {
	t.Helper()

	source, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return source
}

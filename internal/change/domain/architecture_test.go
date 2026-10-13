// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestOutwardImportIsDetected(t *testing.T) {
	root := moduleRoot(t)
	fixture := filepath.Join(root, "internal", "change", "domain", "testdata", "outward")

	outward := outwardImportsIn(t, fixture, modulePath(t, root))

	want := []string{
		"github.com/gin-gonic/gin",
		"github.com/google/go-github/v60/github",
		"github.com/jackc/pgx/v5",
		"github.com/open-policy-agent/opa/v1/sdk",
	}
	for _, imported := range want {
		if !slices.Contains(outward, imported) {
			t.Errorf("the guard did not report %s as outward, reported %v", imported, outward)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test's working directory")
		}
		dir = parent
	}
}

func modulePath(t *testing.T, root string) string {
	t.Helper()

	declaration, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for line := range strings.SplitSeq(string(declaration), "\n") {
		if path, declared := strings.CutPrefix(strings.TrimSpace(line), "module "); declared {
			return strings.TrimSpace(path)
		}
	}
	t.Fatalf("%s/go.mod declares no module path", root)
	return ""
}

func outwardImportsIn(t *testing.T, dir, module string) []string {
	t.Helper()

	var outward []string
	walked := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		for _, imported := range importsOf(t, path) {
			if pointsOutward(imported, module) {
				outward = append(outward, imported)
			}
		}
		return nil
	})
	if walked != nil {
		t.Fatalf("walk %s: %v", dir, walked)
	}
	return outward
}

func importsOf(t *testing.T, file string) []string {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("%s: import path %s: %v", file, spec.Path.Value, err)
		}
		imports = append(imports, imported)
	}
	return imports
}

func pointsOutward(imported, module string) bool {
	if isStandardLibrary(imported) {
		return false
	}
	return !isDomainPackage(imported, module)
}

func isStandardLibrary(imported string) bool {
	firstSegment, _, _ := strings.Cut(imported, "/")
	return !strings.Contains(firstSegment, ".")
}

func isDomainPackage(imported, module string) bool {
	return strings.HasPrefix(imported, module+"/") && strings.Contains(imported, "/domain")
}

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

func TestDomainImportsNothingOutward(t *testing.T) {
	root := moduleRoot(t)
	module := modulePath(t, root)

	dirs := domainDirsIn(t, filepath.Join(root, "internal"))
	if len(dirs) == 0 {
		t.Fatal("found no domain package, so this test proves nothing")
	}
	for _, dir := range dirs {
		for _, imported := range outwardImportsIn(t, dir, module) {
			t.Errorf("%s imports %s, so the dependency points outward", dir, imported)
		}
	}
}

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
	for _, file := range sourceFilesIn(t, dir) {
		for _, imported := range importsOf(t, file) {
			if pointsOutward(imported, module) {
				outward = append(outward, imported)
			}
		}
	}
	return outward
}

func sourceFilesIn(t *testing.T, dir string) []string {
	t.Helper()

	return pathsUnder(t, dir, func(path string, entry fs.DirEntry) bool {
		return !entry.IsDir() &&
			strings.HasSuffix(path, ".go") &&
			!strings.HasSuffix(path, "_test.go")
	})
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

func domainDirsIn(t *testing.T, tree string) []string {
	t.Helper()

	return pathsUnder(t, tree, func(_ string, entry fs.DirEntry) bool {
		return entry.IsDir() && entry.Name() == "domain"
	})
}

func pathsUnder(t *testing.T, tree string, selected func(string, fs.DirEntry) bool) []string {
	t.Helper()

	var paths []string
	walked := filepath.WalkDir(tree, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.IsDir() && entry.Name() == "testdata":
			return fs.SkipDir
		case selected(path, entry):
			paths = append(paths, path)
		}
		return nil
	})
	if walked != nil {
		t.Fatalf("walk %s: %v", tree, walked)
	}
	return paths
}

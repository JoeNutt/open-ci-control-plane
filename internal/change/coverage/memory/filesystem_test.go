// SPDX-License-Identifier: GPL-3.0-or-later

package memory_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var coveragePackages = []string{".", filepath.Join("..", "lcov")}

var filesystemImports = []string{"os", "path/filepath", "io/ioutil"}

func TestCoverageNeverReachesTheFilesystem(t *testing.T) {
	for _, dir := range coveragePackages {
		for _, imported := range productionImportsIn(t, dir) {
			if slices.Contains(filesystemImports, imported) {
				t.Errorf("%s imports %s; a suite or repository name must never become a file path", dir, imported)
			}
		}
	}
}

func productionImportsIn(t *testing.T, dir string) []string {
	t.Helper()

	files := productionFilesIn(t, dir)
	if len(files) == 0 {
		t.Fatalf("found no production Go file in %s, so this test proves nothing", dir)
	}
	var imports []string
	for _, file := range files {
		imports = append(imports, importsOf(t, file)...)
	}
	return imports
}

func productionFilesIn(t *testing.T, dir string) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("list %s: %v", dir, err)
	}
	isTest := func(file string) bool { return strings.HasSuffix(file, "_test.go") }
	return slices.DeleteFunc(files, isTest)
}

func importsOf(t *testing.T, file string) []string {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("unquote import in %s: %v", file, err)
		}
		imports = append(imports, path)
	}
	return imports
}

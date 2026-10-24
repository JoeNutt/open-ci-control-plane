// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"testing"
)

func TestTestPackageIsTheImportPathTheDeepestRootGivesTheFilesDirectory(t *testing.T) {
	roots := []ImportRoot{
		importRootFrom(t, ImportRootStatement{Prefix: "example.com/app", Path: ""}),
		importRootFrom(t, ImportRootStatement{Prefix: "example.com/tools", Path: "tools"}),
	}
	for path, want := range map[string]string{
		"internal/auth/owner_test.go":   "example.com/app/internal/auth",
		"owner_test.go":                 "example.com/app",
		"tools/lint/rules_test.go":      "example.com/tools/lint",
		"tools/rules_test.go":           "example.com/tools",
		"toolsmith/forge/forge_test.go": "example.com/app/toolsmith/forge",
	} {
		t.Run(path, func(t *testing.T) {
			parsed, err := ParseRepositoryPath(path)
			if err != nil {
				t.Fatalf("parse repository path: %v", err)
			}

			tested, _, err := testPackageOf(parsed, roots)

			if err != nil {
				t.Fatalf("test package of %s: %v", path, err)
			}
			if tested.String() != want {
				t.Fatalf("test package of %s is %s, want %s", path, tested, want)
			}
		})
	}
}

func TestTestPackageIsUnnamedForAFileNoImportRootHolds(t *testing.T) {
	roots := []ImportRoot{importRootFrom(t, ImportRootStatement{Prefix: "example.com/tools", Path: "tools"})}
	file, err := ParseRepositoryPath("internal/auth/owner_test.go")
	if err != nil {
		t.Fatalf("parse repository path: %v", err)
	}

	_, isNamed, err := testPackageOf(file, roots)

	if err != nil || isNamed {
		t.Fatalf("package named %v with %v, want unnamed and no error", isNamed, err)
	}
}

func TestTestPackageIsRefusedWhereTwoRootsAtOneDepthNameItDifferently(t *testing.T) {
	roots := []ImportRoot{
		importRootFrom(t, ImportRootStatement{Prefix: "example.com/app", Path: "internal"}),
		importRootFrom(t, ImportRootStatement{Prefix: "example.com/app", Path: "internal"}),
		importRootFrom(t, ImportRootStatement{Prefix: "example.com/fork", Path: "internal"}),
	}
	file, err := ParseRepositoryPath("internal/auth/owner_test.go")
	if err != nil {
		t.Fatalf("parse repository path: %v", err)
	}

	_, _, err = testPackageOf(file, roots)

	if !errors.Is(err, errAmbiguousTestPackage) {
		t.Fatalf("error is %v, want %v", err, errAmbiguousTestPackage)
	}
}

func importRootFrom(t *testing.T, stated ImportRootStatement) ImportRoot {
	t.Helper()

	root, err := parseImportRoot(stated)
	if err != nil {
		t.Fatalf("parse import root: %v", err)
	}
	return root
}

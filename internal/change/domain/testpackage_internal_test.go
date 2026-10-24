// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "testing"

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

			tested, err := testPackageOf(parsed, roots)

			if err != nil {
				t.Fatalf("test package of %s: %v", path, err)
			}
			if tested.String() != want {
				t.Fatalf("test package of %s is %s, want %s", path, tested, want)
			}
		})
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

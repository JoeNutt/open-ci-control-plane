// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type ImportRootStatement struct {
	Prefix string
	Path   string
}

type ImportRoot struct {
	prefix    GoImportPath
	directory RepositoryDirectory
}

func parseImportRoot(stated ImportRootStatement) (ImportRoot, error) {
	prefix, err := ParseGoImportPath(stated.Prefix)
	if err != nil {
		return ImportRoot{}, fmt.Errorf("import root prefix: %w", err)
	}
	directory, err := ParseRepositoryDirectory(stated.Path)
	if err != nil {
		return ImportRoot{}, fmt.Errorf("import root %s: %w", prefix, err)
	}
	return ImportRoot{prefix: prefix, directory: directory}, nil
}

func (root ImportRoot) Prefix() GoImportPath { return root.prefix }

func (root ImportRoot) Directory() RepositoryDirectory { return root.directory }

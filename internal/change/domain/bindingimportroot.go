// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ErrSharedImportRootPrefix = errors.New("two import roots share one prefix")

type ImportRootStatement struct {
	Prefix string
	Path   string
}

type ImportRoot struct {
	prefix    GoImportPath
	directory RepositoryDirectory
}

func (root ImportRoot) Prefix() GoImportPath { return root.prefix }

func (root ImportRoot) Directory() RepositoryDirectory { return root.directory }

func importRootsOf(stated []ImportRootStatement) ([]ImportRoot, error) {
	roots := make([]ImportRoot, 0, len(stated))
	for _, written := range stated {
		root, err := parseImportRoot(written)
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(roots, root.sharesPrefixWith) {
			return nil, fmt.Errorf("%w: %s", ErrSharedImportRootPrefix, root.prefix)
		}
		roots = append(roots, root)
	}
	return roots, nil
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

func (root ImportRoot) sharesPrefixWith(other ImportRoot) bool { return root.prefix == other.prefix }

func (root ImportRoot) holds(directory []string) bool {
	rooted := root.directory.segments()
	return len(rooted) <= len(directory) && slices.Equal(directory[:len(rooted)], rooted)
}

func (root ImportRoot) depth() int { return len(root.directory.segments()) }

func (root ImportRoot) importPathOf(directory []string) (GoImportPath, error) {
	written := append([]string{root.prefix.String()}, directory[root.depth():]...)
	return ParseGoImportPath(strings.Join(written, pathSeparator))
}

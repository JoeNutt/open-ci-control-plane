// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

var (
	errNoImportRootHoldsTheTest = errors.New("no import root holds the test file")
	errAmbiguousTestPackage     = errors.New("two import roots name the test file's package differently")
)

func testPackageOf(file RepositoryPath, roots []ImportRoot) (GoImportPath, error) {
	directory := file.directorySegments()
	isElsewhere := func(root ImportRoot) bool { return !root.holds(directory) }
	holding := slices.DeleteFunc(slices.Clone(roots), isElsewhere)
	if len(holding) == 0 {
		return GoImportPath{}, errNoImportRootHoldsTheTest
	}
	deepest := slices.MaxFunc(holding, func(one, other ImportRoot) int { return cmp.Compare(one.depth(), other.depth()) })
	for _, root := range holding {
		if root.depth() == deepest.depth() && root.prefix != deepest.prefix {
			return GoImportPath{}, fmt.Errorf("%w: %s and %s", errAmbiguousTestPackage, root.prefix, deepest.prefix)
		}
	}
	return deepest.importPathOf(directory)
}

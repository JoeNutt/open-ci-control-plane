// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"errors"
	"slices"
)

var errNoImportRootHoldsTheTest = errors.New("no import root holds the test file")

func testPackageOf(file RepositoryPath, roots []ImportRoot) (GoImportPath, error) {
	directory := file.directorySegments()
	isElsewhere := func(root ImportRoot) bool { return !root.holds(directory) }
	holding := slices.DeleteFunc(slices.Clone(roots), isElsewhere)
	if len(holding) == 0 {
		return GoImportPath{}, errNoImportRootHoldsTheTest
	}
	deepest := slices.MaxFunc(holding, func(one, other ImportRoot) int { return cmp.Compare(one.depth(), other.depth()) })
	return deepest.importPathOf(directory)
}

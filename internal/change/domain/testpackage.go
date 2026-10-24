// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

var errAmbiguousTestPackage = errors.New("two import roots name the test file's package differently")

func testPackageOf(file RepositoryPath, roots []ImportRoot) (pkg GoImportPath, isNamed bool, err error) {
	directory := file.directorySegments()
	isElsewhere := func(root ImportRoot) bool { return !root.holds(directory) }
	holding := slices.DeleteFunc(slices.Clone(roots), isElsewhere)
	if len(holding) == 0 {
		return GoImportPath{}, false, nil
	}
	deepest := slices.MaxFunc(holding, ImportRoot.comparedByDepth)
	if rival := slices.IndexFunc(holding, deepest.isRivalledBy); rival >= 0 {
		return GoImportPath{}, false, fmt.Errorf("%w: %s and %s", errAmbiguousTestPackage, deepest.prefix, holding[rival].prefix)
	}
	pkg, err = deepest.importPathOf(directory)
	return pkg, err == nil, err
}

func (root ImportRoot) comparedByDepth(other ImportRoot) int {
	return cmp.Compare(root.depth(), other.depth())
}

func (root ImportRoot) isRivalledBy(other ImportRoot) bool {
	return root.depth() == other.depth() && !root.sharesPrefixWith(other)
}

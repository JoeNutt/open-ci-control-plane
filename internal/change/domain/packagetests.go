// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

var errNoTestNames = errors.New("a package's named tests name no test")

type PackageTests struct {
	pkg   GoImportPath
	names []TopLevelTestName
}

func namedTestsOf(pkg GoImportPath, names []TopLevelTestName) (PackageTests, error) {
	if len(names) == 0 {
		return PackageTests{}, fmt.Errorf("%w: %s", errNoTestNames, pkg)
	}
	return PackageTests{pkg: pkg, names: distinctInOrder(names)}, nil
}

func distinctInOrder(names []TopLevelTestName) []TopLevelTestName {
	held := slices.Clone(names)
	slices.SortFunc(held, TopLevelTestName.compare)
	return slices.Compact(held)
}

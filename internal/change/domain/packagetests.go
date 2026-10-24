// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
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
	held := slices.Clone(names)
	slices.SortFunc(held, func(one, other TopLevelTestName) int { return strings.Compare(one.name, other.name) })
	return PackageTests{pkg: pkg, names: slices.Compact(held)}, nil
}

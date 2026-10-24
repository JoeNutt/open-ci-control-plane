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

func NewPackageTests(pkg GoImportPath, names []TopLevelTestName) PackageTests {
	if len(names) == 0 {
		return wholePackageOf(pkg)
	}
	return PackageTests{pkg: pkg, names: distinctInOrder(names)}
}

func wholePackageOf(pkg GoImportPath) PackageTests { return PackageTests{pkg: pkg} }

func namedTestsOf(pkg GoImportPath, names []TopLevelTestName) (PackageTests, error) {
	if len(names) == 0 {
		return PackageTests{}, fmt.Errorf("%w: %s", errNoTestNames, pkg)
	}
	return NewPackageTests(pkg, names), nil
}

func distinctInOrder(names []TopLevelTestName) []TopLevelTestName {
	held := slices.Clone(names)
	slices.SortFunc(held, TopLevelTestName.compare)
	return slices.Compact(held)
}

func (tests PackageTests) IsWholePackage() bool { return len(tests.names) == 0 }

func (tests PackageTests) Package() GoImportPath { return tests.pkg }

func (tests PackageTests) runsPackageOf(suite CoverageSuite) bool {
	return tests.pkg == suite.importPath
}

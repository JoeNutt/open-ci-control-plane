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
	ordered := slices.Clone(names)
	slices.SortFunc(ordered, TopLevelTestName.compare)
	return slices.Compact(ordered)
}

func (tests PackageTests) IsWholePackage() bool { return len(tests.names) == 0 }

func (tests PackageTests) Package() GoImportPath { return tests.pkg }

func (tests PackageTests) Names() []TopLevelTestName { return slices.Clone(tests.names) }

func (tests PackageTests) runs(result TestResult) bool {
	return tests.pkg == result.Suite.importPath && (tests.IsWholePackage() || tests.namesTest(result.Name))
}

func (tests PackageTests) namesTest(name string) bool {
	return slices.ContainsFunc(tests.names, func(held TopLevelTestName) bool { return held.name == name })
}

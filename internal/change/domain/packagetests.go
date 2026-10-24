// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

type PackageTests struct {
	pkg   GoImportPath
	names []TopLevelTestName
}

func namedTestsOf(pkg GoImportPath, names []TopLevelTestName) (PackageTests, error) {
	if len(names) == 0 {
		return PackageTests{}, errors.New("no names")
	}
	return PackageTests{pkg: pkg, names: names}, nil
}

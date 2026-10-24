// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "testing"

func TestNamedTestsOfRefusesAPackageWithNoNames(t *testing.T) {
	parser, err := ParseGoImportPath("example.com/parser")
	if err != nil {
		t.Fatal(err)
	}

	_, err = namedTestsOf(parser, nil)

	if err == nil {
		t.Fatal("want a refusal for a package named with no tests, got none")
	}
}

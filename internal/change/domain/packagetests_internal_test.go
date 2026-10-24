// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"testing"
)

func parsedImportPath(t *testing.T, stated string) GoImportPath {
	t.Helper()
	path, err := ParseGoImportPath(stated)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNamedTestsOfRefusesAPackageWithNoNames(t *testing.T) {
	_, err := namedTestsOf(parsedImportPath(t, "example.com/parser"), nil)

	if !errors.Is(err, errNoTestNames) {
		t.Fatalf("want errNoTestNames, got %v", err)
	}
}

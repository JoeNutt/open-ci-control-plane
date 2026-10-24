// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "testing"

func TestTestFunctionHoldsItsNameAndTheLinesItSpans(t *testing.T) {
	function, err := ParseTestFunction("TestAdmitsAnOwner", 12, 30)

	if err != nil {
		t.Fatalf("parse test function: %v", err)
	}
	if function.name.name != "TestAdmitsAnOwner" || function.firstLine != 12 || function.lastLine != 30 {
		t.Fatalf("function is %+v, want TestAdmitsAnOwner over lines 12 to 30", function)
	}
}

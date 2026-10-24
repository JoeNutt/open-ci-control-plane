// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "testing"

func TestTestFunctionHoldsItsNameAndTheLinesItSpans(t *testing.T) {
	function, err := ParseTestFunction("TestAdmitsAnOwner", 12, 30)

	if err != nil {
		t.Fatalf("parse test function: %v", err)
	}
	if function.name.name != "TestAdmitsAnOwner" || function.lines != (lineSpan{first: 12, last: 30}) {
		t.Fatalf("function is %+v, want TestAdmitsAnOwner over lines 12 to 30", function)
	}
}

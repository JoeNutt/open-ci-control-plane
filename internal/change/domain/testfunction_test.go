// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestParseTestFunctionRefusesASpanThatIsNotLinesOfAFile(t *testing.T) {
	for _, span := range []struct{ first, last int }{{0, 4}, {-1, 4}, {9, 8}} {
		_, err := domain.ParseTestFunction("TestAdmitsAnOwner", span.first, span.last)

		if !errors.Is(err, domain.ErrMalformedLineSpan) {
			t.Errorf("lines %d to %d: error is %v, want %v", span.first, span.last, err, domain.ErrMalformedLineSpan)
		}
	}
}

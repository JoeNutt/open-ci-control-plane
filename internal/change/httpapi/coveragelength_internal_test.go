// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAnUploadDeclaringAtMostTheCompressedBoundIsAdmitted(t *testing.T) {
	for _, declared := range []int64{0, domain.DeclaredCoverageLimits().MaxCompressedBytes} {
		if refused := uploadLengthRefusalOf(declared); refused != nil {
			t.Errorf("declared %d: refused %#v", declared, refused)
		}
	}
}

func TestAnUploadDeclaringNoLengthOrOneOverTheCompressedBoundIsRefused413NamingTheBound(t *testing.T) {
	for _, declared := range []int64{undeclaredLength, domain.DeclaredCoverageLimits().MaxCompressedBytes + 1} {
		refused := uploadLengthRefusalOf(declared)

		if refused == nil || refused.status() != http.StatusRequestEntityTooLarge || refused.(limitProblem).Limit == "" {
			t.Errorf("declared %d: refused %#v, want a 413 naming the bound", declared, refused)
		}
	}
}

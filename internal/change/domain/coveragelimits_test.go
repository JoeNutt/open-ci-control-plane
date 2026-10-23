// SPDX-License-Identifier: GPL-3.0-or-later

package domain_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestCoverageLimitsStateEachBoundAsItsAllowedCountAndTheLabelItsBreachNames(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	for bound, allowed := range map[error]int64{
		domain.ErrCompressedArtifactTooLarge:   limits.MaxCompressedBytes,
		domain.ErrDecompressedArtifactTooLarge: limits.MaxDecompressedBytes,
		domain.ErrTooManyFileEntries:           int64(limits.MaxFileEntries),
		domain.ErrTooManySuites:                int64(limits.MaxSuitesPerRepository),
		domain.ErrTooManyTestSections:          int64(limits.MaxTestSections),
		domain.ErrTooManyLineRecords:           int64(limits.MaxLineRecords),
	} {
		stated, isStated := limits.LimitOf(bound)

		count, label, _ := strings.Cut(stated, " ")
		if !isStated || count != fmt.Sprint(allowed) || !strings.HasSuffix(bound.Error(), ": "+label) {
			t.Errorf("%v: stated %q (%t), want %d and the label its breach names", bound, stated, isStated, allowed)
		}
	}
}

func TestCoverageLimitsStateNoLimitForAnErrorThatIsNoBound(t *testing.T) {
	_, isStated := domain.DeclaredCoverageLimits().LimitOf(errors.New("the store is full"))

	if isStated {
		t.Error("stated a limit for an error that is no bound")
	}
}

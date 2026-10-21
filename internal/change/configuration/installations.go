// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"fmt"
	"slices"
)

const firstInstallationID = 1

var errTooManyInstallations = errors.New("the allowlist names too many installations")

var errNonPositiveInstallation = errors.New("an installation ID is not positive")

var errRepeatedInstallation = errors.New("an installation ID is named twice")

func installationIDsIn(stated []int64) ([]int64, error) {
	if len(stated) == 0 {
		return nil, installationIDs.missing()
	}
	if len(stated) > MaxInstallations {
		return nil, installationIDs.invalid(fmt.Errorf("%w: more than %d", errTooManyInstallations, MaxInstallations))
	}
	if slices.Min(stated) < firstInstallationID {
		return nil, installationIDs.invalid(errNonPositiveInstallation)
	}
	if ordered := slices.Sorted(slices.Values(stated)); len(slices.Compact(ordered)) != len(stated) {
		return nil, installationIDs.invalid(errRepeatedInstallation)
	}
	return slices.Clone(stated), nil
}

// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"errors"
	"fmt"
	"slices"
)

var errTooManyInstallations = errors.New("the allowlist names too many installations")

var errRepeatedInstallation = errors.New("an installation ID is named twice")

func installationIDsIn(stated statedItems) ([]int64, error) {
	installationIDs, err := valueOf[[]int64](stated, installationIDsItem)
	if err != nil {
		return nil, err
	}
	if len(installationIDs) == 0 {
		return nil, installationIDsItem.invalid(errEmptyList)
	}
	if len(installationIDs) > MaxInstallations {
		return nil, installationIDsItem.invalid(
			fmt.Errorf("%w: more than %d", errTooManyInstallations, MaxInstallations))
	}
	if slices.Min(installationIDs) < firstGitHubID {
		return nil, installationIDsItem.invalid(errNonPositiveID)
	}
	if repeatsAnEntry(installationIDs) {
		return nil, installationIDsItem.invalid(errRepeatedInstallation)
	}
	return installationIDs, nil
}

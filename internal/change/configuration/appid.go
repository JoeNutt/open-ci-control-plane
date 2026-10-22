// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

func appIDIn(stated statedItems) (int64, error) {
	appID, err := valueOf[int64](stated, appIDItem)
	if err != nil {
		return 0, err
	}
	if appID < firstGitHubID {
		return 0, appIDItem.invalid(errNonPositiveID)
	}
	return appID, nil
}

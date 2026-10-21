// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

func appIDIn(stated *int64) (int64, error) {
	if stated == nil {
		return 0, appID.missing()
	}
	return *stated, nil
}

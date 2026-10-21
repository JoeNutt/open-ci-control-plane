// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import "errors"

func appIDIn(stated *int64) (int64, error) {
	if stated == nil {
		return 0, appID.missing()
	}
	if *stated <= 0 {
		return 0, appID.invalid(errors.New("not positive"))
	}
	return *stated, nil
}

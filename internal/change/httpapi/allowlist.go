// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "slices"

type allowlist []int64

func (allowed allowlist) admits(stated payload) bool {
	installationID, isStated := stated.installationID()
	return isStated && slices.Contains(allowed, installationID)
}

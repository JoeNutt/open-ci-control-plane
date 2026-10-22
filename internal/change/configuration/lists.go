// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"cmp"
	"slices"
)

func repeatsAnEntry[Entry cmp.Ordered](stated []Entry) bool {
	return len(slices.Compact(slices.Sorted(slices.Values(stated)))) != len(stated)
}

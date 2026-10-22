// SPDX-License-Identifier: GPL-3.0-or-later

package structural

import _ "embed"

//go:embed ci/structural/direction.rego
var directionRule string

func Modules() map[string]string {
	return map[string]string{"ci/structural/direction.rego": directionRule}
}

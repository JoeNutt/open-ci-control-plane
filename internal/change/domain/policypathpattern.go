// SPDX-License-Identifier: GPL-3.0-or-later

package domain

func (pattern PathPattern) translatesLike(other PathPattern) bool {
	return pattern.expression.String() == other.expression.String()
}

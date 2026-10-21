// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

func familiesPresentIn(stated []string) error {
	if len(stated) == 0 {
		return families.missing()
	}
	return nil
}

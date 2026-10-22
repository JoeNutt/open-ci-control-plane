// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

func appKeyIn(stated statedItems) (Secret, error) {
	content, err := fileContentOf(stated, appKeyFileItem, MaxAppKeyFileBytes)
	if err != nil {
		return Secret{}, err
	}
	return secretOf(content), nil
}

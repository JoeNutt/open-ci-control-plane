// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

func appKeyIn(stated statedItems) (Secret, error) {
	content, err := fileContentOf(stated, appKeyFileItem, MaxSecretFileBytes)
	if err != nil {
		return Secret{}, err
	}
	return secretOf(content), nil
}

func AppKeyParsedBy[Key any](configured Configuration, parse func([]byte) (Key, error)) (Key, error) {
	key, err := parse(configured.appKey.Reveal())
	if err != nil {
		var none Key
		return none, appKeyFileItem.invalid(err)
	}
	return key, nil
}

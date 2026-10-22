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
	return parsedFileItem(appKeyFileItem, configured.appKey.Reveal(), parse)
}

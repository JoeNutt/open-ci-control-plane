// SPDX-License-Identifier: GPL-3.0-or-later

package configuration

import (
	"bytes"
	"errors"
)

const (
	textFileEnd         = "\n"
	firstPrintableASCII = ' '
	lastPrintableASCII  = '~'
)

var errEmptySecret = errors.New("the secret is empty once its line break is removed")

var errUnprintableSecret = errors.New("the secret holds a character outside printable ASCII")

func webhookSecretIn(stated statedItems) (Secret, error) {
	content, err := fileContentOf(stated, webhookSecretFileItem, MaxSecretFileBytes)
	if err != nil {
		return Secret{}, err
	}
	secret := bytes.TrimSuffix(content, []byte(textFileEnd))
	if len(secret) == 0 {
		return Secret{}, webhookSecretFileItem.invalid(errEmptySecret)
	}
	if bytes.ContainsFunc(secret, isOutsidePrintableASCII) {
		return Secret{}, webhookSecretFileItem.invalid(errUnprintableSecret)
	}
	return secretOf(secret), nil
}

func isOutsidePrintableASCII(character rune) bool {
	return character < firstPrintableASCII || character > lastPrintableASCII
}

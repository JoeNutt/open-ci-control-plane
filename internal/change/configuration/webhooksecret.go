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

func webhookSecretIn(path string) (Secret, error) {
	content, err := webhookSecretFile.contentAt(path, MaxSecretFileBytes)
	if err != nil {
		return Secret{}, err
	}
	secret := bytes.TrimSuffix(content, []byte(textFileEnd))
	if len(secret) == 0 {
		return Secret{}, webhookSecretFile.invalid(errEmptySecret)
	}
	if bytes.ContainsFunc(secret, isControlOrBeyondASCII) {
		return Secret{}, webhookSecretFile.invalid(errUnprintableSecret)
	}
	return secretOf(secret), nil
}

func isControlOrBeyondASCII(character rune) bool {
	return character < firstPrintableASCII || character > lastPrintableASCII
}
